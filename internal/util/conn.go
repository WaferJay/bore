package util

import (
	"net"
	"time"
	"strconv"
	"encoding/binary"
	"crypto/md5"
)

func errAddr(addr, msg string) error {
	return &net.AddrError{Addr: addr, Err: msg}
}

func ParseAddr(addr net.Addr) (net.IP, int, error) {
	return ParseHostPort(addr.String())
}

func ParseHostPort(hostport string) (net.IP, int, error) {
	strHost, strPort, err := net.SplitHostPort(hostport)
	if err != nil {
		return nil, 0, err
	}

	ip := net.ParseIP(strHost)
	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, 0, errAddr(hostport, "invalid port")
	}

	if port > 0xffff {
		return nil, 0, errAddr(hostport, "invalid port")
	}

	return ip, port, nil
}

func HashUDPAddr(addr *net.UDPAddr) uint64 {
	return HashAddr(addr.IP, uint16(addr.Port))
}

func HashAddr(ip net.IP, port uint16) uint64 {
	ins := md5.New()
	ins.Write(ip)
	binary.Write(ins, binary.BigEndian, port)
	result := make([]byte, 16)
	ins.Sum(result)
	return binary.LittleEndian.Uint64(result[4:12])
}

func SetReadTimeout(conn net.Conn, duration time.Duration) {
	conn.SetReadDeadline(time.Now().Add(duration))
}

func SetWriteTimeout(conn net.Conn, duration time.Duration) {
	conn.SetWriteDeadline(time.Now().Add(duration))
}

func SetTimeout(conn net.Conn, duration time.Duration) {
	conn.SetDeadline(time.Now().Add(duration))
}
