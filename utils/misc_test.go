package pulse

import (
	"net"
	"testing"
)

func TestPublicIP(t *testing.T) {
	cases_public := []string{"8.8.8.8", "1.1.1.1", "45.45.45.45", "120.222.111.222", "2404:6800:4003:c01::64"}
	for _, ipstr := range cases_public {
		ip := net.ParseIP(ipstr)
		v := islocalip(ip)
		if v {
			t.Error("Should be false for " + ipstr)
		}
	}
}

func TestLocalIP(t *testing.T) {
	cases_private := []string{"127.0.0.1", "10.5.6.4", "192.168.5.99", "100.66.55.66", "fd07:a47c:3742:823e:3b02:76:982b:463"}
	for _, ipstr := range cases_private {
		ip := net.ParseIP(ipstr)
		v := islocalip(ip)
		if !v {
			t.Error("Should be true for " + ipstr)
		}
	}
}
