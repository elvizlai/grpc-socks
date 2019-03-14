package main

import (
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

var leakyBuf = lib.NewLeakyBuf(2048, leakyBufSize)

func (p *proxy) Echo(ctx context.Context, req *pb.Payload) (*pb.Payload, error) {
	return &pb.Payload{Data: p.serverToken}, nil
}

// TODO buff
func (p *proxy) ResolveIP(ctx context.Context, req *pb.IPAddr) (*pb.IPAddr, error) {
	ipAddr, err := net.ResolveIPAddr("ip", req.Address)
	if err != nil {
		return nil, err
	}

	req.Data = ipAddr.IP
	req.Zone = ipAddr.Zone

	return req, nil
}

func (p *proxy) Pump(stream pb.Proxy_PumpServer) error {
	frame := &pb.Payload{}

	// first frame
	err := stream.RecvMsg(frame)
	if err != nil {
		log.Errorf("tcp first frame err: %s", err)
		return err
	}

	addr := string(frame.Data)
	// addr must be a name?

	conn, err := net.DialTimeout("tcp", addr, time.Second*15)
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

func (p *proxy) PipelineUDP(stream pb.Proxy_PipelineUDPServer) error {
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
