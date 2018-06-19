package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"grpc-socks/lib"
)

func main() {
	conn, err := net.Dial("udp", "127.0.0.1:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	go func() {
		for i := 0; ; i++ {
			reqStr := "ping" + fmt.Sprint(i)
			log.Println("req", reqStr)
			_, err := conn.Write([]byte(reqStr))
			if err != nil {
				log.Fatal(err)
				break
			}
			<-time.After(time.Second)
		}
	}()

	buff := make([]byte, lib.UDPMaxSize)
	for {
		n, err := conn.Read(buff)
		if err != nil {
			continue
		}
		log.Println("resp", string(buff[:n]))
	}

}
