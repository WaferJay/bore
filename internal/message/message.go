package message

import (
	"encoding/binary"
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
	CliHeartbeat
)

const (
	SerOK      = 0
	SerReject  = 1 << iota
	SerRetry
	SerTimeout
	SerConnect
)

func AddProtoHeader(entry *BufEntry) error {
	return entry.PutUint16(ProtoHeader, binary.BigEndian)
}

func AddFlowProtoHeader(flow *BufWriteFlow) *BufWriteFlow {
	return flow.Uint16(ProtoHeader, binary.BigEndian)
}

func CheckProtoHeader(entry *BufEntry) bool {
	if v, err := entry.ReadUint16(binary.BigEndian);
		err != nil || v != ProtoHeader {

		return false
	}
	return true
}

func MakeFlagsByte(first byte, bs []byte) byte {
	var flag = int(first)
	for b := range bs {
		flag |= b
	}
	return byte(flag)
}
