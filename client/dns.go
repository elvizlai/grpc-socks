package main

import (
	"context"
	"net"

	"github.com/coocood/freecache"

	"github.com/elvizlai/grpc-socks/pb"
)

type DNSResolver struct {
	cache *freecache.Cache
}

var expireSeconds = 7200

var nameCtxKey = struct{}{}

// DNSResolver uses the remote DNS to resolve host names
func (d DNSResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	ctx = context.WithValue(ctx, nameCtxKey, name)

	if v, err := d.cache.Get([]byte(name)); err == nil {
		return ctx, v, nil
	}

	ipResp, err := proxyClient.ResolveIP(ctx, &pb.IPAddr{
		Address: name,
	})
	if err == nil {
		d.cache.Set([]byte(name), ipResp.Data, expireSeconds)
	}

	return ctx, ipResp.Data, err
}
