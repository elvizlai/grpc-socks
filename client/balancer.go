package main

import (
	"../lib"
	"../pb"
	"../log"

	"strings"
	"time"

	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc"
	"golang.org/x/net/context"
)

type etcdResolver struct {
	rawAddr string
	cc      resolver.ClientConn
	hasInit bool
}

func (r *etcdResolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	r.cc = cc

	go r.watch(target.Endpoint)

	return r, nil
}

func (r etcdResolver) Scheme() string {
	return "proxy"
}

func (r etcdResolver) ResolveNow(rn resolver.ResolveNowOption) {
	log.Infof("ResolveNow") // TODO check
}

// Close closes the resolver.
func (r etcdResolver) Close() {
	log.Infof("Close")
}

func (r *etcdResolver) watch(addr string) {
	addrList := strings.Split(addr, ",")

	if len(addrList) == 1 {
		r.cc.NewAddress([]resolver.Address{{Addr: addrList[0]}})
		return
	}

	maxTolerant := time.Duration(time.Millisecond * time.Duration(tolerant))
	if maxTolerant <= 0 {
		var list []resolver.Address
		for i := range addrList {
			list = append(list, resolver.Address{Addr: addrList[i]})
		}
		r.cc.NewAddress(list)
		return
	}

	//delay test
	var acm = make(map[string]pb.ProxyServiceClient, 0)
	for i := range addrList {
		conn, err := grpc.Dial(addrList[i], grpc.WithTransportCredentials(lib.ClientTLS()))
		if err != nil {
			acm[addrList[i]] = nil
		} else {
			acm[addrList[i]] = pb.NewProxyServiceClient(conn)
		}
	}

	pt := time.Minute * time.Duration(period)
	timer := time.NewTimer(pt)

	for {
		var list []resolver.Address

		for k, v := range acm {
			if v != nil {
				if delay := measure(v); delay <= maxTolerant {
					log.Debugf("service %s, time delay: %s", k, delay)
					list = append(list, resolver.Address{Addr: k})
				} else {
					log.Warnf("service %s, time delay %s too high, drop", k, delay)
				}
			} else {
				log.Errorf("conn to service %s failed", k)
			}
		}

		// append all if list is empty. TODO optimize, better chosen reachable service
		if len(list) == 0 {
			for k := range acm {
				list = append(list, resolver.Address{Addr: k})
			}
		}

		r.cc.NewAddress(list)

		<-timer.C

		if pt == 0 {
			break
		}

		timer.Reset(pt)
	}

}

// TODO timeout handle
func measure(c pb.ProxyServiceClient) (dur time.Duration) {
	defer func(cur time.Time) {
		dur = time.Now().Sub(cur) / 3
	}(time.Now())

	for i := 0; i < 3; i++ {
		func() {
			ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
			defer cancelFunc()
			_, err := c.Echo(ctx, &pb.Payload{})
			if err == context.Canceled {
				log.Warnf("time out")
			}
		}()
	}

	return dur
}
