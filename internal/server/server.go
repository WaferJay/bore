package server

import (
	"encoding/binary"
	"errors"
	"log"
	"net"
	"time"

	"bore/internal/message"
	"bore/internal/server/udp"
	"bore/internal/util"
)

const (
	statusCreated = 0
	statusRunning = 1
	statusClosed  = 2
)

var (
	ErrAlreadyStarted = errors.New("already started")
	ErrAlreadyStopped = errors.New("already stopped")
	ErrNotRunning     = errors.New("not running")
)

type P2PServer struct {
	addr      *net.TCPAddr
	UDPAddr   *net.UDPAddr
	config    *P2PServerConfig
	listener  *net.TCPListener
	Allocator *message.BufAllocator
	ackServer *udp.AckServer
	operator  *util.DefaultObservable
	status    int
}

func (s *P2PServer) Ack() util.Observable {
	return s.ackServer.Ack
}

func (s *P2PServer) Heartbeat() util.Observable {
	return s.ackServer.Heartbeat
}

func (s *P2PServer) Operator() util.Observable {
	return s.operator
}

func NewP2PServer(tcpAddrStr, udpAddrStr string) (*P2PServer, error) {
	var err error
	var tcpAddr *net.TCPAddr
	var udpAddr *net.UDPAddr
	tcpAddr, err = net.ResolveTCPAddr("tcp", tcpAddrStr)
	udpAddr, err = net.ResolveUDPAddr("udp", udpAddrStr)

	if err != nil {
		return nil, err
	}

	s := &P2PServer{
		addr:      tcpAddr,
		UDPAddr:   udpAddr,
		config:    DefaultP2PServerConfig(),
		Allocator: message.NewBufAllocator(),
		operator:  util.NewDefaultObservable(true, true, time.Minute),
		status:    statusCreated,
	}
	return s, nil
}

func (s *P2PServer) Start() error {
	if s.status == statusRunning {
		return ErrAlreadyStarted
	}

	l, err := net.ListenTCP("tcp", s.addr)

	if err != nil {
		return err
	}

	ackServer := udp.NewAckServer(s.UDPAddr, s.Allocator)
	if err := ackServer.Start(); err != nil {
		_ = l.Close()
		s.status = statusClosed
		return err
	} else {
		s.listener = l
		s.ackServer = ackServer
	}
	return nil
}

func (s *P2PServer) Close() (err error) {
	if s.status == statusCreated {
		return ErrNotRunning
	} else if s.status == statusClosed {
		return ErrAlreadyStopped
	} else {
		s.status = statusClosed
		err = s.ackServer.Close()
		err = s.listener.Close()
		return err
	}
}

func (s *P2PServer) Loop() {
	go s.ackServer.Loop()

	for {
		if s.status == statusClosed {
			break
		}

		conn, err := s.listener.Accept()
		if err != nil {
			log.Println("Accept() error:", err)
			continue
		}

		util.Debug.Log("Accepted connection:", conn.RemoteAddr())
		go s.handleConnection(conn)
	}
}

func (s *P2PServer) handleConnection(conn net.Conn) {

	config := s.config
	alloc := s.Allocator

	buf := alloc.Alloc(260)
	defer alloc.Dealloc(buf)

	remoteAddr := conn.RemoteAddr()
	session := NewSession(s, conn)
	defer func() {
		addr := session.Conn().RemoteAddr()
		if err := session.Close(); err != nil {
			log.Printf("Close [%s] error: %s", addr, err)
		}
		log.Printf("Closed session [%s]", addr)
	}()

	log.Printf("Start session [%s]", remoteAddr)

	for {
		buf.Clear()

		util.SetReadTimeout(conn, config.ReadTimeout)

		if _, err := util.ReadAtLeast(conn, buf, 3); err != nil {
			util.Debug.Logf("Recv from %s failed: %s", remoteAddr, err)
			return
		}

		if !message.CheckProtoHeader(buf) {
			if util.Debug {
				util.Debug.Logf("Incorrect protocol: 0x%X", buf.B[:2])
			}
			return
		}

		var exc error

		switch cmd, _ := buf.ReadByte(); cmd {
		case message.CliHeartbeat:
			session.SendOk()

		case message.CliJoin:
			log.Printf("[%s] CMD: JOIN", remoteAddr)
			exc = session.Join()

		case message.CliSentAck:
			log.Printf("[%s] CMD: SENT_ACK", remoteAddr)
			exc = session.SentAck()

		case message.CliDial:
			log.Printf("[%s] CMD: DIAL", remoteAddr)
			var IPlen byte
			if IPlen, exc = buf.ReadByte(); exc != nil {
				session.Reject()
				return
			}

			IPBytes := make([]byte, IPlen)
			buf.ReadBytes(IPBytes)
			ip := net.IP(IPBytes)
			port, err := buf.ReadUint16(binary.BigEndian)
			if err != nil {
				exc = ErrReject
			}

			exc = session.Dial(ip, port)

		case message.CliWaitCall:
			log.Printf("[%s] CMD: WAIT_CALL", remoteAddr)
			exc = session.WaitCall()

		default:
			log.Printf("Unknown cmd: 0x%X", cmd)
			exc = ErrReject
		}

		if exc == ErrReject {
			session.Reject()
			log.Printf("[%s] Rejected", remoteAddr)
			return
		} else if exc == ErrSessionEnd {
			util.Debug.Logf("[%s] End of session", remoteAddr)
			return
		} else {
			util.Debug.Log("Error:", exc)
		}
	}
}
