package server

import (
	buf2 "bore/internal/buf"
	"bore/internal/config"
	"bore/internal/message"
	"bore/internal/util"
	"encoding/binary"
	"log"
	"math/rand"
	"net"
	"time"
)

type SessionState interface {
	Join(s *Session)
	SentAck(s *Session)
	Dial(s *Session, ip net.IP, port uint16)
	WaitCall(s *Session)
}

type Session struct {
	conn        net.Conn
	server      *P2PServer
	state       SessionState
	ackId       uint32
	udpAddrChan chan interface{}
	UDPAddr     *net.UDPAddr
	retryCount  int
	Closed      bool
}

func NewSession(server *P2PServer, conn net.Conn) *Session {
	return &Session{server: server, conn: conn, state: &stateInit}
}

func (s *Session) Conn() net.Conn           { return s.conn }
func (s *Session) Server() *P2PServer       { return s.server }
func (s *Session) setState(ss SessionState) { s.state = ss }

func (s *Session) Join()                       { s.state.Join(s) }
func (s *Session) SentAck()                    { s.state.SentAck(s) }
func (s *Session) Dial(ip net.IP, port uint16) { s.state.Dial(s, ip, port) }
func (s *Session) WaitCall()                   { s.state.WaitCall(s) }

func (s *Session) Send(buf *buf2.BufEntry) error {
	_, err := buf2.WriteBuf(s.Conn(), buf)
	if err != nil {
		addr := s.conn.RemoteAddr()
		log.Printf("[%s] Send response failed: %s", addr, err)
		util.Debug.Logf("[%s] Message: %s", addr, buf.HexString())
	}
	return err
}

func (s *Session) close() {
	if s.Closed {
		return
	}

	s.Closed = true
	if err := s.conn.Close(); err != nil {
		util.Debug.Logf("[%s] Close connection error: %s", s.conn.RemoteAddr(), err)
	}
	s.conn = nil

	ch := s.udpAddrChan
	ackId := s.ackId
	if ackId != 0 && ch != nil {
		s.Server().Ack().Unregister(ackId, ch)
		select {
		case ch <- nil:
		default:
		}
	}
	s.ackId = 0
	s.udpAddrChan = nil
	s.UDPAddr = nil
	return
}

func (s *Session) sendReject(flags ...byte) {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(4)
	defer alloc.Dealloc(buf)

	flag := message.MakeFlagsByte(message.SerReject, flags)

	message.AddFlowProtoHeader(buf.Flow()).
		Byte(flag).
		FatalIfError()

	if err := s.Send(buf); err != nil {
		addr := s.conn.RemoteAddr()
		util.Debug.Logf("[%s] Send REJECT failed: %s", addr, err)
	}
}

func (s *Session) sendClose(flags ...byte) {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(4)
	defer alloc.Dealloc(buf)

	flag := message.MakeFlagsByte(message.SerClose, flags)

	message.AddFlowProtoHeader(buf.Flow()).
		Byte(flag).
		FatalIfError()

	if err := s.Send(buf); err != nil {
		util.Debug.Logf("[%s] Send CLOSE failed: %s", s.conn.RemoteAddr(), err)
	}
}

func (s *Session) Reject() {
	if !s.Closed {
		util.Debug.Logf("[%s] Rejecting", s.conn.RemoteAddr())
		s.sendReject()
		s.close()
	}
}

func (s *Session) Close() {
	if !s.Closed {
		util.Debug.Logf("[%s] Closing", s.conn.RemoteAddr())
		s.sendClose()
		s.close()
	}
	return
}

func (s *Session) SendAckId(port int, ackId uint32) error {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(16)
	defer alloc.Dealloc(buf)

	message.AddFlowProtoHeader(buf.Flow()).
		Byte(message.SerOK).
		Uint16(uint16(port), binary.BigEndian).
		Uint32(ackId, binary.LittleEndian).
		FatalIfError()

	return s.Send(buf)
}

func (s *Session) SendAddr(ip net.IP, port int) error {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(32)
	defer alloc.Dealloc(buf)

	message.AddFlowProtoHeader(buf.Flow()).
		Byte(message.SerOK).
		Byte(byte(len(ip))).
		Bytes(ip).
		Uint16(uint16(port), binary.BigEndian).
		FatalIfError()

	return s.Send(buf)
}

func (s *Session) SendCallerAddr(addr *net.UDPAddr) error {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(32)
	defer alloc.Dealloc(buf)

	message.AddFlowProtoHeader(buf.Flow()).
		Byte(message.SerConnect).
		Byte(byte(len(addr.IP))).
		Bytes(addr.IP).
		Uint16(uint16(addr.Port), binary.BigEndian).
		FatalIfError()

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

	message.AddFlowProtoHeader(buf.Flow()).
		Byte(byte(flag)).
		FatalIfError()

	return s.Send(buf)
}

func (s *Session) SendOk() error {
	alloc := s.Server().Allocator
	buf := alloc.Alloc(16)
	defer alloc.Dealloc(buf)

	message.AddFlowProtoHeader(buf.Flow()).
		Byte(message.SerOK).
		FatalIfError()

	return s.Send(buf)
}

type (
	initSessionState   byte
	ackSessionState    byte
	obtainSessionState byte
)

var (
	stateInit   initSessionState
	stateAck    ackSessionState
	stateObtain obtainSessionState
)

func (*initSessionState) Join(session *Session) {
	port := session.Server().UDPAddr.Port

	ackId := rand.Uint32() | 0x1
	if err := session.SendAckId(port, ackId); err != nil {
		session.Close()
		return
	}

	udpAddrChan := make(chan interface{})
	session.ackId = ackId
	session.Server().Ack().Register(ackId, udpAddrChan)
	session.udpAddrChan = udpAddrChan

	session.setState(&stateAck)
}

func (*ackSessionState) SentAck(s *Session) {
	udpAddrChan := s.udpAddrChan
	select {
	case raw := <-udpAddrChan:
		udpAddr := raw.(*net.UDPAddr)
		if s.UDPAddr = udpAddr; udpAddr == nil {
			s.Reject()
			return
		}

		log.Printf("[%s] Determined UDP Address: %s", s.conn.RemoteAddr(), udpAddr)
		if err := s.SendAddr(udpAddr.IP, udpAddr.Port); err != nil {
			s.Close()
			return
		}
		s.setState(&stateObtain)

	case <-time.NewTimer(config.ChannelTimeout).C:
		s.Server().Ack().Unregister(s.ackId, udpAddrChan)
		if s.retryCount++; s.retryCount >= config.MaxRetryCount {
			log.Printf("[%s] Timeout", s.conn.RemoteAddr())
			s.Reject()
			return
		}

		if err := s.SendRetry(true); err != nil {
			s.Close()
			return
		}
		s.setState(&stateInit)
	}
	s.udpAddrChan = nil
	s.ackId = 0
	return
}

func (*obtainSessionState) WaitCall(s *Session) {
	if s.UDPAddr == nil {
		s.Reject()
		return
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
	for !s.Closed {
		select {
		case raw := <-callerChannel:
			if raw == nil {
				s.Reject()
				return
			}

			callerAddr := raw.(*net.UDPAddr)
			log.Printf("[%s] Incoming call: %s", remoteAddr, callerAddr)
			if s.SendCallerAddr(callerAddr) != nil {
				s.Reject()
			}
			// TODO: change state
			return

		// reset timer
		case <-keepAliveChan:
			beatCount++
			if beatCount >= config.MaxHeartbeatCount {
				log.Printf("[%s] Waiting for call timeout", remoteAddr)
				s.Server().Operator().Unregister(addrId, callerChannel)
				s.Reject()
			}

		case <-time.NewTimer(config.ChannelTimeout).C:
			log.Printf("[%s] Heartbest timeout", remoteAddr)
			s.Server().Operator().Unregister(addrId, callerChannel)
			s.Reject()
		}
	}
}

func (*obtainSessionState) Dial(s *Session, ip net.IP, port uint16) {
	addrId := util.HashAddr(ip, port)
	if ok := s.Server().Operator().Notify(addrId, s.UDPAddr); ok {
		if err := s.SendOk(); err != nil {
			util.Debug.Logf("[%s] SendOk fail: %s", s.conn.RemoteAddr(), err)
			s.Reject()
		}
		// TODO: ensure dial success
	} else {
		s.Reject()
	}
}

func (*obtainSessionState) SentAck(s *Session) { s.Reject() }

func (*initSessionState) SentAck(s *Session)                  { s.Reject() }
func (*initSessionState) Dial(s *Session, _ net.IP, _ uint16) { s.Reject() }
func (*initSessionState) WaitCall(s *Session)                 { s.Reject() }

func (*ackSessionState) Join(s *Session)                     { s.Reject() }
func (*ackSessionState) Dial(s *Session, _ net.IP, _ uint16) { s.Reject() }
func (*ackSessionState) WaitCall(s *Session)                 { s.Reject() }

func (*obtainSessionState) Join(s *Session) { s.Reject() }
