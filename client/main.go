package main

import (
	"flag"

	socks5 "github.com/armon/go-socks5"
	"github.com/coocood/freecache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/resolver"

	"github.com/elvizlai/grpc-socks/lib"
	"github.com/elvizlai/grpc-socks/log"
	"github.com/elvizlai/grpc-socks/pb"
)

var (
	debug      = false
	compress   = false
	localAddr  = "127.0.0.1:50050"
	remoteAddr = "127.0.0.1:50051"

	proxyClient pb.ProxyClient
	tolerant    uint
	period      uint
)

func init() {
	flag.BoolVar(&debug, "d", debug, "debug mode")
	flag.StringVar(&localAddr, "l", localAddr, "local addr")
	flag.StringVar(&remoteAddr, "r", remoteAddr, "remote addr")
	flag.BoolVar(&compress, "cp", compress, "enable snappy compress")
	flag.Parse()

	if debug {
		log.SetDebugMode()
	}

	if compress {
		encoding.RegisterCompressor(lib.Snappy())
		callOptions = append(callOptions, grpc.UseCompressor("snappy"))
	}
}

func main() {
	resolver.Register(&etcdResolver{})

	conn, err := grpc.Dial("lb:///"+remoteAddr, grpc.WithBalancerName("round_robin"), grpc.WithTransportCredentials(lib.ClientTLS()))
	if err != nil {
		panic(err)
	}
	proxyClient = pb.NewProxyClient(conn)

	conf := &socks5.Config{
		Resolver: DNSResolver{cache: freecache.NewCache(100 * 1024 * 1024)},
		Dial:     DialFunc,
	}

	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	if err := server.ListenAndServe("tcp", localAddr); err != nil {
		panic(err)
	}
}
