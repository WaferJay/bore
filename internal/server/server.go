package server

import (
	buf2 "bore/internal/buf"
	"errors"
	"io"
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
	Allocator *buf2.Allocator
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
		Allocator: buf2.NewBufAllocator(),
		operator:  util.NewDefaultObservable(true, true, time.Minute),
		status:    statusCreated,
	}
	return s, nil
}

func (s *P2PServer) Start() error {
	if s.status == statusRunning {
		return ErrAlreadyStarted
	}
	s.status = statusRunning

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
			util.Debug.Log("Accept() error:", err)
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

	log.Printf("Start session [%s]", remoteAddr)

	for !session.Closed {
		buf.Clear()

		util.SetReadTimeout(conn, config.ReadTimeout)

		if _, err := buf2.ReadAtLeast(conn, buf, 3); err != nil {
			if err != io.EOF {
				util.Debug.Logf("Recv from %s failed: %s", remoteAddr, err)
				session.Reject()
			}
			return
		}

		if !message.CheckProtoHeader(buf) {
			if util.Debug {
				util.Debug.Logf("Incorrect protocol: 0x%X", buf.B[:2])
			}
			session.Reject()
			return
		}

		switch cmd, _ := buf.ReadByte(); cmd {
		case message.CliTest:
			util.Debug.Logf("[%s] CMD: TEST", remoteAddr)
			if session.SendOk() != nil {
				session.Close()
			}

		case message.CliJoin:
			log.Printf("[%s] CMD: JOIN", remoteAddr)
			session.Join()

		case message.CliSentAck:
			log.Printf("[%s] CMD: SENT_ACK", remoteAddr)
			session.SentAck()

		case message.CliDial:
			log.Printf("[%s] CMD: DIAL", remoteAddr)
			ip, port, err := message.ReadAddr(buf)
			if err != nil {
				session.Reject()
				return
			}
			session.Dial(ip, port)

		case message.CliWaitCall:
			log.Printf("[%s] CMD: WAIT_CALL", remoteAddr)
			session.WaitCall()

		case message.CliClose:
			log.Printf("[%s] CLOSE", remoteAddr)
			session.Close()

		default:
			log.Printf("[%s] Unknown cmd: 0x%X", remoteAddr, cmd)
			session.Reject()
		}
	}

	log.Printf("Closed session [%s]", remoteAddr)
}
