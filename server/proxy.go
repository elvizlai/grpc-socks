package main

import (
	"bytes"
	"io"
	"net"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/peer"

	"github.com/elvizlai/grpc-socks/lib"
	"github.com/elvizlai/grpc-socks/log"
	"github.com/elvizlai/grpc-socks/pb"
)

type proxy struct {
	serverToken []byte
}

const leakyBufSize = 4108 // data.len(2) + hmacsha1(10) + data(4096)
const maxNBuf = 2048

var leakyBuf = lib.NewLeakyBuf(maxNBuf, leakyBufSize)

func (p *proxy) Echo(ctx context.Context, req *pb.Payload) (*pb.Payload, error) {
	if len(p.serverToken) == 0 {
		buff := &bytes.Buffer{}

		buff.WriteString(version)
		buff.WriteByte('@')
		buff.WriteString(buildAt)

		p.serverToken = buff.Bytes()
	}
	return &pb.Payload{Data: p.serverToken}, nil
}

func (p *proxy) Pipeline(stream pb.ProxyService_PipelineServer) error {
	frame := &pb.Payload{}

	err := stream.RecvMsg(frame)
	if err != nil {
		log.Errorf("tcp first frame err: %s", err)
		return err
	}

	addr := string(frame.Data)

	conn, err := net.DialTimeout("tcp", addr, time.Second*10)
	if err != nil {
		log.Errorf("tcp dial %q err: %s", addr, err)
		return err
	}
	defer conn.Close()

	conn.(*net.TCPConn).SetKeepAlive(true)

	ctx := stream.Context()
	// get peer info from ctx, maybe it won't be nil is this case
	info, ok := peer.FromContext(ctx)
	if ok {
		defer log.Debugf("tcp close %q<-->%q<-->%q", info.Addr.String(), addr, conn.RemoteAddr())
		log.Debugf("tcp conn %q<-->%q<-->%q", info.Addr.String(), addr, conn.RemoteAddr())
	} else {
		defer log.Debugf("tcp close %q<-->%q<-->%q", conn.LocalAddr(), addr, conn.RemoteAddr())
		log.Debugf("tcp conn %q<-->%q<-->%q", conn.LocalAddr(), addr, conn.RemoteAddr())
	}

	go func() {
		for {
			p, err := stream.Recv()

			if err != nil {
				if ctx.Err() != context.Canceled && err != io.EOF {
					log.Errorf("stream recv err: %s", err)
				}
				break
			}

			_, err = conn.Write(p.Data)
			if err != nil {
				log.Errorf("tcp conn write err: %s", err)
				break
			}
		}
		conn.Close() // close conn
	}()

	buff := leakyBuf.Get()
	defer leakyBuf.Put(buff)

	for {
		n, err := conn.Read(buff)
		if err != nil {
			break
		}

		if n > 0 {
			frame.Data = buff[:n]
			err = stream.Send(frame)
			if err != nil {
				log.Errorf("stream send err: %s", err)
				break
			}
		}
	}

	return nil
}

func (p *proxy) PipelineUDP(stream pb.ProxyService_PipelineUDPServer) error {
	frame := &pb.Payload{}

	err := stream.RecvMsg(frame)
	if err != nil {
		log.Errorf("udp first frame err: %s", err)
		return err
	}

	addr := string(frame.Data)

	log.Debugf("recv udp addr: %s", addr)

	conn, err := net.Dial("udp", addr)
	if err != nil {
		log.Errorf("udp dial %s err: %s", addr, err)
		return err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Second * 600))

	ctx := stream.Context()

	go func() {
		buff := make([]byte, lib.UDPMaxSize)

		for {
			n, err := conn.Read(buff)
			if n > 0 {
				frame.Data = buff[:n]
				err = stream.Send(frame)
				if err != nil {
					log.Errorf("stream send err: %s", err)
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
			if ctx.Err() == context.Canceled {
				break
			}
			log.Errorf("stream recv err: %s", err)
			return err
		}

		_, err = conn.Write(p.Data)
		if err != nil {
			log.Errorf("udp conn write err: %s", err)
			return err
		}
	}

	return nil
}
