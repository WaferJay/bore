package config

import "time"

const (
	MaxRetryCount     = 3
	MaxHeartbeatCount = 15
	ChannelTimeout    = 20 * time.Second
)
