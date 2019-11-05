package udp

import (
	"bore/internal/config"
	"encoding/binary"
	"log"
	"net"

	"bore/internal/message"
	"bore/internal/util"
)

type AckServer struct {
	addr      *net.UDPAddr
	conn      *net.UDPConn
	allocator *message.BufAllocator
	Heartbeat util.Observable
	Ack       util.Observable
}

func NewAckServer(addr *net.UDPAddr, alloc *message.BufAllocator) *AckServer {
	return &AckServer{
		addr:      addr,
		allocator: alloc,
		Ack:       util.NewDefaultObservable(true, true, config.ChannelTimeout),
		Heartbeat: util.NewDefaultObservable(false, true, config.ChannelTimeout),
	}
}

func (us *AckServer) Start() error {
	conn, err := net.ListenUDP("udp", us.addr)
	us.conn = conn
	return err
}

func (us *AckServer) Loop() {
	buf := us.allocator.Alloc(8)
	defer us.allocator.Dealloc(buf)

	for {
		buf.Clear()

		n, addr, err := us.conn.ReadFromUDP(buf.B)
		buf.SetWriterIndex(n)

		if err != nil {
			log.Println("UDP Read error:", err)
			continue
		}

		if n <= 6 || !message.CheckProtoHeader(buf) {
			header := buf.B[:message.ProtoHeaderLen]
			util.Debug.Logf("UDP [R/%s] - Incorrect packet header: 0x%X", addr, header)
			continue
		}

		var ackId uint32
		ackId, err = buf.ReadUint32(binary.LittleEndian)
		if err != nil {
			content := buf.B[message.ProtoHeaderLen:n]
			util.Debug.Logf("UDP [R/%s] - Incorrect ack packet: %X", addr, content)
			continue
		}

		if ackId != 0 {
			util.Debug.Logf("UDP [R/%s] - Received ack 0x%x", addr, ackId)
			if !us.Ack.Notify(ackId, addr) {
				log.Printf("UDP [R/%s] - Notified nobody of ack", addr)
			}
		} else {
			util.Debug.Logf("UDP [R/%s] - Received heartbeat", addr)
			id := util.HashUDPAddr(addr)
			if !us.Heartbeat.Notify(id, 1) {
				log.Printf("UDP [R/%s] - Notified nobody of heartbeat", addr)
			}
		}
	}
}

func (us *AckServer) Close() (err error) {
	err = us.conn.Close()
	us.Ack.Clear()
	us.Heartbeat.Clear()
	return
}
