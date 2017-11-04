package main

import (
	"flag"
	"net"
	"os"
	"runtime"

	"../lib"
	"../log"
	"../pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

var addr = ":50051"
var debug = false
var showVersion bool

var version = "self-build"

func init() {
	flag.StringVar(&addr, "l", addr, "listen addr")
	flag.BoolVar(&debug, "d", debug, "debug mode")
	flag.BoolVar(&showVersion, "v", false, "show version then exit")

	flag.Parse()

	if showVersion {
		log.Infof("version: %s, using: %s", version, runtime.Version())
		os.Exit(0)
	}

	encoding.RegisterCompressor(lib.Snappy())

	if debug {
		log.SetDebugMode()
	}
}

func main() {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}
	defer lis.Close()

	log.Infof("starting proxy server at %q ...", addr)

	s := grpc.NewServer(grpc.Creds(lib.ServerTLS()), grpc.StreamInterceptor(interceptor))
	defer s.GracefulStop()

	pb.RegisterProxyServiceServer(s, &proxy{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}

func interceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return handler(srv, ss)
}
