package log

import (
	"log"
	"os"
)

var std = log.New(os.Stdout, "", log.LstdFlags)

var debug = false

// SetDebugMode set debug mode
func SetDebugMode() {
	debug = true
}

// Debugln Debugln
func Debugln(e ...interface{}) {
	if debug {
		std.Printf("\033[33m%v\033[0m\n", e)
	}
}

// Debugf with format
func Debugf(format string, para ...interface{}) {
	if debug {
		std.Printf("\033[33m"+format+"\033[0m", para...)
	}
}

// Infof with format
func Infof(format string, para ...interface{}) {
	std.Printf("\033[32m"+format+"\033[0m", para...)
}

// Warnf with format
func Warnf(format string, para ...interface{}) {
	std.Printf("\033[35m"+format+"\033[0m", para...)
}

// Errorf with format
func Errorf(format string, para ...interface{}) {
	std.Printf("\033[31m"+format+"\033[0m", para...)
}

// Errorln Errorln
func Errorln(e interface{}) {
	std.Printf("\033[31m%v\033[0m\n", e)
}

// Fatalf with format
func Fatalf(format string, para ...interface{}) {
	std.Fatalf("\033[31m"+format+"\033[0m", para...)
}
