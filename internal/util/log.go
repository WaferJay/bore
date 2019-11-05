package util

import (
	"log"
	"os"
)

type debugLog bool

var Debug debugLog
var logger = log.New(os.Stdout, "[DEBUG] ", log.Ldate | log.Ltime | log.Lmicroseconds)

func SetDebug(debug debugLog) {
	Debug = debug
}

func (d debugLog) Logf(format string, v ...interface{}) {
	if d {
		logger.Printf(format, v...)
	}
}

func (d debugLog) Log(v ...interface{}) {
	if d {
		logger.Println(v...)
	}
}
