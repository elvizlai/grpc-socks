package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"grpc-socks/pb"
)

var (
	crt  = "server.crt"
	key  = "server.key"
	ca   = "ca.crt"
	addr = ":8080"
)

func main() {
	// Load the certificates from disk
	certificate, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		log.Fatalf("could not load server key pair: %s", err)
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(ca)
	if err != nil {
		log.Fatalf("could not read ca certificate: %s", err)
	}

	// Append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("failed to append client certs")
	}

	// Create the channel to listen on
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("could not list on %s: %s", addr, err)
	}

	// Create the TLS credentials
	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	})

	// Create the gRPC server with the credentials
	srv := grpc.NewServer(grpc.Creds(creds))

	// Register the handler object
	pb.RegisterProxyServiceServer(srv, &proxy{})

	// Serve and Listen
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("grpc serve error: %s", err)
	}
}

type proxy struct {
}

func (p *proxy) Echo(ctx context.Context, req *pb.Payload) (*pb.Payload, error) {
	pr, ok := peer.FromContext(ctx)

	if ok {
		fmt.Printf("%#v\n", pr.AuthInfo)
	}

	return req, nil
}

func (p *proxy) Pipeline(stream pb.ProxyService_PipelineServer) error {
	return nil
}

func (p *proxy) PipelineUDP(stream pb.ProxyService_PipelineUDPServer) error {
	return nil
}
