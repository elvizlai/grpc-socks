package main

import (
	"flag"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"

	"grpc-socks/lib"
	"grpc-socks/log"
	"grpc-socks/pb"
)

var addr = ":50051"
var debug = false
var showVersion bool

var version = "self-build"
var buildAt = ""

func init() {
	flag.StringVar(&addr, "l", addr, "listen addr")
	flag.BoolVar(&debug, "d", debug, "debug mode")
	flag.BoolVar(&showVersion, "v", false, "show version then exit")

	flag.Parse()

	if showVersion {
		log.Infof("version:%s, build at %q using %s", version, buildAt, runtime.Version())
		os.Exit(0)
	}

	if debug {
		log.SetDebugMode()
	}

	encoding.RegisterCompressor(lib.Snappy())
}

func main() {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}
	defer ln.Close()

	log.Infof("starting proxy server at %q ...", addr)

	m := cmux.New(ln)

	httpL := m.Match(cmux.HTTP1Fast())
	httpS := &http.Server{
		Handler: nil,
	}
	go httpS.Serve(httpL)

	grpcL := m.Match(cmux.Any())
	grpcS := grpc.NewServer(grpc.Creds(lib.ServerTLS()), grpc.StreamInterceptor(interceptor))
	defer grpcS.GracefulStop()
	pb.RegisterProxyServiceServer(grpcS, &proxy{serverToken: append([]byte(version), append([]byte("@"), []byte(buildAt)...)...)})
	go func() {
		err := grpcS.Serve(grpcL)
		if err != nil {
			log.Fatalf("failed to serve grpc: %s", err.Error())
		}
	}()

	if err := m.Serve(); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}

func interceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return handler(srv, ss)
}
