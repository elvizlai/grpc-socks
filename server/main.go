package main

import (
	"flag"
	"log"
	"net"

	"../lib"
	"../pb"

	"google.golang.org/grpc"
)

var addr = ":50051"
var debug = false

func init() {
	flag.StringVar(&addr, "l", addr, "listen addr")
	flag.BoolVar(&debug, "d", debug, "debug mode")

	flag.Parse()
}

func main() {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}
	defer lis.Close()

	log.Printf("starting proxy server at %q ...\n", addr)

	s := grpc.NewServer(grpc.Creds(lib.ServerTLS()))
	defer s.GracefulStop()

	pb.RegisterProxyServiceServer(s, &proxy{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
