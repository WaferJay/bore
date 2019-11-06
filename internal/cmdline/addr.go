package cmdline

import (
	"bore/internal/util"
	"net"
	"strconv"
)

type Address struct {
	IP net.IP
	Port int
}

func NewAddress(v string) *Address {
	addr := &Address{}

	if err := addr.Set(v); err != nil {
		panic(err)
	}

	return addr
}

func (a *Address) Set(v string) error {
	host, port, err := util.ParseHostPort(v)
	if err != nil {
		return err
	}
	if host == nil {
		host = net.IPv6zero
	}
	a.IP = host
	a.Port = port
	return nil
}

func (a *Address) String() string {
	str := ":" + strconv.Itoa(a.Port)
	if a.IP != nil {
		strIP := a.IP.String()
		if len(a.IP) == net.IPv6len {
			strIP = "[" + strIP + "]"
		}
		str = strIP + str
	}
	return str
}
