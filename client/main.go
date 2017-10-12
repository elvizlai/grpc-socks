package main

import (
	"flag"
	"net"
	"os"
	"runtime"
	"time"

	"../lib"
	"../log"
	"../pb"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var addr = ":50050"
var remoteAddr = ""
var debug bool
var showVersion bool

var version = "self-build"

func init() {
	flag.StringVar(&addr, "l", addr, "listen addr")
	flag.StringVar(&remoteAddr, "r", remoteAddr, "remote addr")
	flag.BoolVar(&debug, "d", false, "debug mode")
	flag.BoolVar(&showVersion, "v", false, "show version then exist")

	flag.Parse()

	if showVersion {
		log.Infof("version: %s, using: %s", version, runtime.Version())
		os.Exit(0)
	}

	if remoteAddr == "" {
		log.Fatalf("remote addr can not be empty")
	}

	if debug {
		log.SetDebugMode()
	}
}

func main() {
	// try to establish gRPC conn
	client, err := gRPCClient()
	if err != nil {
		log.Fatalf(err.Error())
	} else {
		ctx, delayTestFrame := context.Background(), &pb.Payload{Data: []byte{0x2e, 0xf6, 0xae, 0x1e, 0x83}}

		_, err = client.Echo(ctx, delayTestFrame)
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
		conn, err := grpc.Dial(remoteAddr, grpc.WithTransportCredentials(lib.ClientTLS()))
		if err != nil {
			return nil, err
		}

		client = &Client{ClientConn: conn, ProxyServiceClient: pb.NewProxyServiceClient(conn)}
	}

	return client, nil
}
