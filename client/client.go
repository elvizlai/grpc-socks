package main

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/elvizlai/grpc-socks/log"
	"github.com/elvizlai/grpc-socks/pb"
)

var callOptions = make([]grpc.CallOption, 0)

func DialFunc(ctx context.Context, network, addr string) (net.Conn, error) {
	log.Debugf("%q<-%s->%q", ctx.Value(nameCtxKey), network, addr)

	tcpAddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		return nil, err
	}

	if isLocal(tcpAddr) {
		return net.DialTCP(network, nil, tcpAddr)
	}

	// ctx = metadata.AppendToOutgoingContext(ctx, "url", ctx.Value(nameCtxKey).(string))
	stream, err := proxyClient.Pump(ctx, callOptions...)
	if err != nil {
		return nil, err
	}

	err = stream.Send(&pb.Payload{Data: []byte(tcpAddr.String())})
	if err != nil {
		return nil, err
	}

	return &client{addr: tcpAddr, stream: stream}, nil
}

var _ net.Conn = &client{}

type client struct {
	addr   net.Addr
	stream pb.Proxy_PumpClient
}

func (c *client) Read(b []byte) (n int, err error) {
	p, err := c.stream.Recv()
	if err != nil {
		return 0, err
	}

	return copy(b, p.Data), nil
}

func (c *client) Write(b []byte) (n int, err error) {
	p := &pb.Payload{
		Data: b,
	}

	return len(b), c.stream.Send(p)
}

func (c *client) Close() error {
	return c.stream.CloseSend()
}

func (c *client) LocalAddr() net.Addr {
	return c.addr
}

func (c *client) RemoteAddr() net.Addr {
	return nil
}

// TODO impl
func (c *client) SetDeadline(t time.Time) error {
	return nil
}

// TODO impl
func (c *client) SetReadDeadline(t time.Time) error {
	return nil
}

// TODO impl
func (c *client) SetWriteDeadline(t time.Time) error {
	return nil
}
