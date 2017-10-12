package main

import (
	"crypto/x509"
	"io/ioutil"
	"crypto/tls"
	"log"

	"../../pb"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
	"golang.org/x/net/context"
)

var (
	crt  = "client.crt"
	key  = "client.key"
	ca   = "ca.crt"
	addr = ":8080"
)

func main() {
	// Load the client certificates from disk
	certificate, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		log.Fatalf("could not load client key pair: %s", err)
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(ca)
	if err != nil {
		log.Fatalf("could not read ca certificate: %s", err)
	}

	// Append the certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("failed to append ca certs")
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   "1024server",
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
		//InsecureSkipVerify: true,
	})

	// Create a connection with the TLS credentials
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("could not dial %s: %s", addr, err)
	}

	// Initialize the client and make the request
	client := pb.NewProxyServiceClient(conn)
	pong, err := client.Echo(context.Background(), &pb.Payload{Data: []byte{1, 0, 2, 4}})
	if err != nil {
		log.Fatalf("could not ping %s: %s", addr, err)
	}

	// Log the ping
	log.Printf("%s\n", pong.String())
}
