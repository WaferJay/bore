package message

import (
	"bore/internal/buf"
	"encoding/binary"
	"net"
)

const (
	ProtoHeader    = 0xd7c1
	ProtoHeaderLen = 2
)

const (
	CliJoin     = iota
	CliSentAck
	CliDial
	CliWaitCall
	CliClose
	CliTest
)

const (
	SerOK      = 0
	SerReject  = 1 << iota
	SerRetry
	SerTimeout
	SerConnect
	SerClose
)

func AddProtoHeader(entry *buf.BufEntry) error {
	return entry.PutUint16(ProtoHeader, binary.BigEndian)
}

func AddFlowProtoHeader(flow *buf.BufWriteFlow) *buf.BufWriteFlow {
	return flow.Uint16(ProtoHeader, binary.BigEndian)
}

func CheckProtoHeader(entry *buf.BufEntry) bool {
	if v, err := entry.ReadUint16(binary.BigEndian);
		err != nil || v != ProtoHeader {

		return false
	}
	return true
}

func ReadUDPAddr(entry *buf.BufEntry) (*net.UDPAddr, error) {
	ip, u16Port, err := ReadAddr(entry)
	if err != nil {
		return nil, err
	}

	return &net.UDPAddr{
		IP:   ip,
		Port: int(u16Port),
	}, nil
}

func ReadAddr(entry *buf.BufEntry) (net.IP, uint16, error) {
	ipLen, err := entry.ReadByte()
	if err != nil {
		return nil, 0, err
	}

	ipBytes := make([]byte, ipLen)
	if n := entry.ReadBytes(ipBytes); n != int(ipLen) {
		return nil, 0, buf.ErrOutBound
	}

	ip := net.IP(ipBytes)
	u16Port, err := entry.ReadUint16(binary.BigEndian)
	if err != nil {
		return nil, 0, err
	}

	return ip, u16Port, err
}

func MakeFlagsByte(first byte, bs []byte) byte {
	var flag = int(first)
	for b := range bs {
		flag |= b
	}
	return byte(flag)
}
