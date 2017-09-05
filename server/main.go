package main

import (
	"flag"
	"net"

	"../lib"
	"../pb"
	"../log"

	"google.golang.org/grpc"
)

var addr = ":50051"
var debug = false

func init() {
	flag.StringVar(&addr, "l", addr, "listen addr")
	flag.BoolVar(&debug, "d", debug, "debug mode")

	flag.Parse()

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
