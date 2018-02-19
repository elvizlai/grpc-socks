package main

import (
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"

	"github.com/elvizlai/grpc-socks/lib"
	"github.com/elvizlai/grpc-socks/log"
	"github.com/elvizlai/grpc-socks/pb"
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
		var list, dropList []resolver.Address

		for k, v := range acm {
			if v != nil {
				if delay, info, err := measure(v); err == nil {
					if delay <= maxTolerant {
						log.Debugf("service %s, time delay: %s, %q", k, delay, info)
						list = append(list, resolver.Address{Addr: k})
					} else {
						log.Warnf("service %s, time delay %s too high, drop, %q", k, delay, info)
						dropList = append(dropList, resolver.Address{Addr: k})
					}
				} else {
					log.Errorf("conn to service %s failed, err: %s", k, err)
				}
			} else {
				log.Errorf("conn to service %s failed", k)
			}
		}

		if len(list) == 0 {
			log.Errorf("no available services, try using drop list: %v", dropList)
			list = dropList
		}

		r.cc.NewAddress(list)

		<-timer.C

		if pt == 0 {
			break
		}

		timer.Reset(pt)
	}

}

func measure(c pb.ProxyServiceClient) (dur time.Duration, info string, err error) {
	defer func(cur time.Time) {
		dur = time.Now().Sub(cur)
	}(time.Now())

	var errChan = make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func() {
			ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
			defer cancelFunc()
			resp, e := c.Echo(ctx, &pb.Payload{})
			if resp != nil && info == "" {
				info = string(resp.Data)
			}
			errChan <- e
		}()
	}

	rc := 0

L:
	for {
		select {
		case err = <-errChan:
			if err != nil {
				return
			}
			rc++
			if rc == 3 {
				break L
			}
		}
	}

	return
}
