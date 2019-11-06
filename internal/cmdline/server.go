package cmdline

import "flag"

const (
	defaultServerAddress = ":8751"
	defaultUDPAddress    = ":8752"
)

type ServerConfig struct {
	TCPAddress *Address
	UDPAddress *Address
	Debug      bool
	Help       bool
}

func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		TCPAddress: NewAddress(defaultServerAddress),
		UDPAddress: NewAddress(defaultUDPAddress),
		Help: false,
		Debug: false,
	}
}

func InitialServerConfig() *ServerConfig {
	config := DefaultServerConfig()

	flag.Var(config.TCPAddress, "l", "server address")
	flag.Var(config.UDPAddress, "u", "UDP AckServer address")

	flag.BoolVar(&config.Help, "h", false, "show the help message")
	flag.BoolVar(&config.Debug, "d", false, "debug mode")

	flag.Parse()
	return config
}
