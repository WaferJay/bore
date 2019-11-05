package server

import "time"

type P2PServerConfig struct {
	ReadTimeout       time.Duration
	ConnectionTimeout time.Duration
}

const (
	defConnectionTimeout = 15 * time.Second
)

func DefaultP2PServerConfig() *P2PServerConfig {
	return &P2PServerConfig{
		ReadTimeout:       defConnectionTimeout,
		ConnectionTimeout: defConnectionTimeout,
	}
}
