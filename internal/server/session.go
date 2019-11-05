package server

import (
	"bore/internal/config"
	"bore/internal/message"
	"bore/internal/util"
	"encoding/binary"
	"errors"
	"log"
	"math/rand"
	"net"
	"time"
)

var (
	ErrSessionEnd = errors.New("end")
	ErrReject     = errors.New("reject")
)

type SessionState interface {
	Join(s *Session) error
	SentAck(s *Session) error
	Dial(s *Session, ip net.IP, port uint16) error
	WaitCall(s *Session) error
}

type Session struct {
	conn        net.Conn
	server      *P2PServer
	state       SessionState
	ackId       uint32
	udpAddrChan chan interface{}
	UDPAddr     *net.UDPAddr
	retryCount  int
}

func NewSession(server *P2PServer, conn net.Conn) *Session {
	return &Session{server: server, conn: conn, state: &stateInit}
}

func (s *Session) Conn() net.Conn           { return s.conn }
func (s *Session) Server() *P2PServer       { return s.server }
func (s *Session) setState(ss SessionState) { s.state = ss }

func (s *Session) Join() error                       { return s.state.Join(s) }
func (s *Session) SentAck() error                    { return s.state.SentAck(s) }
func (s *Session) Dial(ip net.IP, port uint16) error { return s.state.Dial(s, ip, port) }
func (s *Session) WaitCall() error                   { return s.state.WaitCall(s) }

func (s *Session) Send(buf *message.BufEntry) error {
	_, err := util.WriteBuf(s.Conn(), buf)
	return err
}

func (s *Session) Reject(flags ...byte) error {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(4)
	defer alloc.Dealloc(buf)

	flag := message.MakeFlagsByte(message.SerReject, flags)
	message.AddProtoHeader(buf)
	buf.PutByte(flag)

	return s.Send(buf)
}

func (s *Session) Close() (err error) {
	if s.conn != nil {
		s.conn.Write([]byte(""))
		err = s.conn.Close()
		s.conn = nil

		ch := s.udpAddrChan
		ackId := s.ackId
		if ackId != 0 && ch != nil{
			s.Server().Ack().Unregister(ackId, ch)
			select {
			case ch <- nil:
			default:
			}
		}
		s.ackId = 0
		s.udpAddrChan = nil
		s.UDPAddr = nil
	}
	return
}

func (s *Session) SendAckId(port int, ackId uint32) error {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(16)
	defer alloc.Dealloc(buf)

	flow := message.AddFlowProtoHeader(buf.Flow()).
		Byte(message.SerOK).
		Uint16(uint16(port), binary.BigEndian).
		Uint32(ackId, binary.LittleEndian)

	if err := flow.Error(); err != nil {
		flow.LogError()
		return err
	}

	return s.Send(buf)
}

func (s *Session) SendAddr(ip net.IP, port int) error {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(32)
	defer alloc.Dealloc(buf)

	message.AddProtoHeader(buf)
	buf.PutByte(message.SerOK)
	buf.PutByte(byte(len(ip)))
	buf.PutBytes(ip)
	buf.PutUint16(uint16(port), binary.BigEndian)

	return s.Send(buf)
}

func (s *Session) SendCallerAddr(addr *net.UDPAddr) error {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(32)
	defer alloc.Dealloc(buf)

	message.AddProtoHeader(buf)
	buf.PutByte(message.SerConnect)
	buf.PutByte(byte(len(addr.IP)))
	buf.PutBytes(addr.IP)
	buf.PutUint16(uint16(addr.Port), binary.BigEndian)

	return s.Send(buf)
}

func (s *Session) SendRetry(timeout bool) error {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(16)
	defer alloc.Dealloc(buf)

	flag := message.SerRetry
	if timeout {
		flag |= message.SerTimeout
	}

	message.AddProtoHeader(buf)
	buf.PutByte(byte(flag))

	return s.Send(buf)
}

func (s *Session) SendOk() error {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(16)
	defer alloc.Dealloc(buf)

	message.AddProtoHeader(buf)
	buf.PutByte(message.SerOK)

	return s.Send(buf)
}

type (
	initSessionState byte
	ackSessionState byte
	obtainSessionState byte
)

var (
	stateInit   initSessionState
	stateAck    ackSessionState
	stateObtain obtainSessionState
)

func (*initSessionState) Join(session *Session) error {
	port := session.Server().UDPAddr.Port

	ackId := rand.Uint32() | 0x1
	err := session.SendAckId(port, ackId)
	if err != nil {
		return ErrSessionEnd
	}

	udpAddrChan := make(chan interface{})
	session.ackId = ackId
	session.Server().Ack().Register(ackId, udpAddrChan)
	session.udpAddrChan = udpAddrChan

	session.setState(&stateAck)
	return nil
}

func (*ackSessionState) SentAck(s *Session) error {
	udpAddrChan := s.udpAddrChan
	select {
	case raw := <-udpAddrChan:
		udpAddr := raw.(*net.UDPAddr)
		if s.UDPAddr = udpAddr; udpAddr == nil {
			return ErrReject
		}
		if s.SendAddr(udpAddr.IP, udpAddr.Port) != nil {
			return ErrSessionEnd
		}
		s.setState(&stateObtain)

	case <-time.NewTimer(config.ChannelTimeout).C:
		s.Server().Ack().Unregister(s.ackId, udpAddrChan)
		if s.retryCount++; s.retryCount >= config.MaxRetryCount {
			return ErrReject
		}

		if s.SendRetry(true) != nil {
			return ErrSessionEnd
		}
		s.setState(&stateInit)
	}
	s.udpAddrChan = nil
	s.ackId = 0
	return nil
}

func (*obtainSessionState) WaitCall(s *Session) error {
	if s.UDPAddr == nil {
		return ErrReject
	}

	heartbeat := s.Server().Heartbeat()

	addrId := util.HashUDPAddr(s.UDPAddr)
	keepAliveChan := make(chan interface{})

	heartbeat.Register(addrId, keepAliveChan)
	defer heartbeat.Unregister(addrId, keepAliveChan)

	callerChannel := make(chan interface{})
	s.Server().Operator().Register(addrId, callerChannel)

	remoteAddr := s.Conn().RemoteAddr()
	var beatCount int
	for {
		select {
		case raw := <-callerChannel:
			if raw == nil {
				return ErrSessionEnd
			}

			callerAddr := raw.(*net.UDPAddr)
			log.Printf("[%s] Incoming call: %s", remoteAddr, callerAddr)
			return s.SendCallerAddr(callerAddr)

		// reset timer
		case <-keepAliveChan:
			beatCount++
			if beatCount >= config.MaxHeartbeatCount {
				log.Printf("[%s] Waiting for call timeout", remoteAddr)
				s.Server().Operator().Unregister(addrId, callerChannel)
				return ErrReject
			}

		case <-time.NewTimer(config.ChannelTimeout).C:
			log.Printf("[%s] Heartbest timeout", remoteAddr)
			s.Server().Operator().Unregister(addrId, callerChannel)
			return ErrReject
		}
	}
}

func (*obtainSessionState) Dial(s *Session, ip net.IP, port uint16) error {
	addrId := util.HashAddr(ip, port)
	if ok := s.Server().Operator().Notify(addrId, s.UDPAddr); ok {
		if err := s.SendOk(); err != nil {
			util.Debug.Log("SendOk fail:", err)
		}
		return ErrSessionEnd
	}
	return ErrReject
}

func (*obtainSessionState) SentAck(s *Session) error {
	return stateAck.SentAck(s)
}

func (*initSessionState) SentAck(*Session) error              { return ErrReject }
func (*initSessionState) Dial(*Session, net.IP, uint16) error { return ErrReject }
func (*initSessionState) WaitCall(*Session) error             { return ErrReject }

func (*ackSessionState) Join(*Session) error                 { return ErrReject }
func (*ackSessionState) Dial(*Session, net.IP, uint16) error { return ErrReject }
func (*ackSessionState) WaitCall(*Session) error             { return ErrReject }

func (*obtainSessionState) Join(*Session) error { return ErrReject }
