package main

import (
	"flag"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/elvizlai/grpc-socks/lib"
	"github.com/elvizlai/grpc-socks/log"
	"github.com/elvizlai/grpc-socks/pb"
	"google.golang.org/grpc/resolver"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

var addr = ":50050"
var remoteAddr = ""
var debug bool
var compress bool
var showVersion bool
var tolerant uint
var period uint

var version = "self-build"
var buildAt = ""

func init() {
	flag.StringVar(&addr, "l", addr, "listen addr")
	flag.StringVar(&remoteAddr, "r", remoteAddr, "remote addr, if multi, using ',' to split")
	flag.BoolVar(&compress, "cp", compress, "enable snappy compress")
	flag.UintVar(&tolerant, "t", 500, "tolerant of delay (ms), 0 means no check")
	flag.UintVar(&period, "p", 15, "period of delay check (minute), 0 means check once")
	flag.BoolVar(&debug, "d", false, "debug mode")
	flag.BoolVar(&showVersion, "v", false, "show version then exit")

	flag.Parse()

	if showVersion {
		log.Infof("version:%s, build at %q using %s", version, buildAt, runtime.Version())
		os.Exit(0)
	}

	if debug {
		log.SetDebugMode()
	}

	if remoteAddr == "" {
		log.Fatalf("remote addr can not be empty")
	}

	if compress {
		encoding.RegisterCompressor(lib.Snappy())
		callOptions = append(callOptions, grpc.UseCompressor("snappy"))
	}
}

func main() {
	// try to establish gRPC conn
	client, err := gRPCClient()
	if err != nil {
		log.Fatalf(err.Error())
	} else {
		ctx, delayTestFrame := context.Background(), &pb.Payload{}

		_, err = client.Echo(ctx, delayTestFrame, callOptions...)
		if err == nil {
			var total, n = time.Duration(0), 3
			for i := 0; i < n; i++ {
				s := time.Now()
				client.Echo(ctx, delayTestFrame)
				total += time.Now().Sub(s)
			}
			log.Infof("conn to server time delay: %s", total/time.Duration(n))
		} else {
			log.Errorf("WARN, %s", err)
		}
	}

	defer func() {
		if client != nil {
			client.Close()
		}
	}()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf(err.Error())
	}

	log.Infof("starting server tcp at %v ...", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Errorf("accept err: %v", err)
			continue
		}

		conn.(*net.TCPConn).SetKeepAlive(true)

		// close the Nagle algorythm, for long fat pipe if necessary
		// conn.(*net.TCPConn).SetNoDelay(false)

		// XXX DONOT set linger -1, cause after close write each incomming packet will be rejected and post back a RST packet.
		// If so, another side of the tcp connection will return connection reset by peer error.
		//conn.(*net.TCPConn).SetLinger(-1)

		go handleConnection(conn)
	}
}

type Client struct {
	*grpc.ClientConn
	pb.ProxyServiceClient
}

var client *Client

// using conn/client pool is better, but not necessary now
func gRPCClient() (*Client, error) {
	if client == nil {
		resolver.Register(&etcdResolver{})

		conn, err := grpc.Dial("proxy:///"+remoteAddr, grpc.WithBalancerName("round_robin"), grpc.WithTransportCredentials(lib.ClientTLS()))
		if err != nil {
			return nil, err
		}

		client = &Client{ClientConn: conn, ProxyServiceClient: pb.NewProxyServiceClient(conn)}
	}

	return client, nil
}
