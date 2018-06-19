package main

import (
	"log"
	"net"
	"net/http"
	"time"

	"grpc-socks/lib"
)

func main() {
	udpLn, err := net.ListenPacket("udp", ":50051")
	if err != nil {
		log.Fatal(err)
		return
	}

	buff := make([]byte, lib.UDPMaxSize)

	for {
		n, addr, err := udpLn.ReadFrom(buff)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Println("addr:", addr, "data:", string(buff[:n]))

		go func() {
			_, err = udpLn.WriteTo([]byte(time.Now().Format(http.TimeFormat)+" ---> "+string(buff[:n])), addr)
			if err != nil {
				log.Println(err)
			}
		}()
	}
}
