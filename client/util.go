package main

import (
	"net"
)

// 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, 100.64.0.0/10, 17.0.0.0/8
var localAddrList = []*net.IPNet{
	parseCIDR("192.168.0.0/16"),
	parseCIDR("10.0.0.0/8"),
	parseCIDR("172.16.0.0/12"),
	parseCIDR("100.64.0.0/10"),
	parseCIDR("17.0.0.00/8"),
}

func parseCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

func isLocal(addr *net.TCPAddr) bool {
	if addr.IP.String() == "127.0.0.1" {
		return true
	}

	for i := range localAddrList {
		if localAddrList[i].Contains(addr.IP) {
			return true
		}
	}

	return false
}
