package cmdline

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func HoldUntilInt() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <- ch
	log.Printf("Hit the %s signal", s.String())
}
