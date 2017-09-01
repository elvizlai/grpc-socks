package main

import (
	"io"
	"log"
	"net"
	"time"

	"../lib"
	"../pb"

	"golang.org/x/net/context"
)

type proxy struct {
}

const leakyBufSize = 4108 // data.len(2) + hmacsha1(10) + data(4096)
const maxNBuf = 2048

var leakyBuf = lib.NewLeakyBuf(maxNBuf, leakyBufSize)

func (p *proxy) Echo(ctx context.Context, req *pb.Payload) (*pb.Payload, error) {
	return req, nil
}

func (p *proxy) Pipeline(stream pb.ProxyService_PipelineServer) error {
	frame := &pb.Payload{}

	err := stream.RecvMsg(frame)
	if err != nil {
		log.Printf("tcp first frame err: %s\n", err)
		return err
	}

	addr := string(frame.Data)

	if debug {
		log.Printf("recv tcp addr: %s\n", addr)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("tcp dail %s err: %s\n", addr, err)
		return err
	}
	defer conn.Close()

	// set read deadline
	conn.SetReadDeadline(time.Now().Add(time.Second * 600))

	go func() {
		buff := leakyBuf.Get()
		defer leakyBuf.Put(buff)

		for {
			n, err := conn.Read(buff)

			if n > 0 {
				frame.Data = buff[:n]
				err = stream.Send(frame)
				if err != nil {
					log.Printf("stream send err: %s\n", err)
					break
				}
			}

			if err != nil {
				break
			}
		}
	}()

	for {
		p, err := stream.Recv()

		if err == io.EOF {
			return nil
		}

		if err != nil {
			log.Printf("stream recv err: %s\n", err)
			return err
		}

		_, err = conn.Write(p.Data)
		if err != nil {
			log.Printf("tcp conn write err: %s\n", err)
			return err
		}
	}

	return nil
}

func (p *proxy) PipelineUDP(stream pb.ProxyService_PipelineUDPServer) error {
	frame := &pb.Payload{}

	err := stream.RecvMsg(frame)
	if err != nil {
		log.Printf("udp first frame err: %s\n", err)
		return err
	}

	addr := string(frame.Data)

	if debug {
		log.Printf("recv udp addr: %s\n", addr)
	}

	conn, err := net.Dial("udp", addr)
	if err != nil {
		log.Printf("udp dail %s err: %s\n", addr, err)
		return err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Second * 600))

	go func() {
		buff := make([]byte, lib.UDPMaxSize)

		for {
			n, err := conn.Read(buff)
			if n > 0 {
				frame.Data = buff[:n]
				err = stream.Send(frame)
				if err != nil {
					log.Printf("stream send err: %s\n", err)
					break
				}
			}

			if err != nil {
				break
			}
		}
	}()

	for {
		p, err := stream.Recv()

		if err == io.EOF {
			return nil
		}

		if err != nil {
			log.Printf("stream recv err: %s\n", err)
			return err
		}

		_, err = conn.Write(p.Data)
		if err != nil {
			log.Printf("udp conn write err: %s\n", err)
			return err
		}
	}

	return nil
}
