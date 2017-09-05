package log

import (
	"os"
	"log"
)

var std = log.New(os.Stdout, "", log.LstdFlags)

var debug = false

func SetDebugMode() {
	debug = true
}

//func Printf(format string, para ...interface{}) {
//	std.Printf(format, para...)
//}

func Debugln(e ...interface{}) {
	if debug {
		std.Printf("\033[33m%v\033[0m\n", e)
	}
}

func Debugf(format string, para ...interface{}) {
	if debug {
		std.Printf("\033[33m"+format+"\033[0m", para...)
	}
}

func Infof(format string, para ...interface{}) {
	std.Printf("\033[32m"+format+"\033[0m", para...)
}

func Warnf(format string, para ...interface{}) {
	std.Printf("\033[35m"+format+"\033[0m", para...)
}

func Errorf(format string, para ...interface{}) {
	std.Printf("\033[31m"+format+"\033[0m", para...)
}

func Errorln(e interface{}) {
	std.Printf("\033[31m%v\033[0m\n", e)
}

func Fatalf(format string, para ...interface{}) {
	std.Fatalf("\033[31m"+format+"\033[0m", para...)
}
