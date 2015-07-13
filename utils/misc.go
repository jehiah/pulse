package pulse

import (
	"errors"
	"net"
)

var (
	localipv4                = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "127.0.0.0/8", "100.64.0.0/10"}
	localipv6                = []string{"fc00::/7", "::1/128"}
	securityerr              = errors.New("Security error: Not allowed to connect to local IP")
	tlsHandshakeTimeoutError = errors.New("net/http: TLS handshake timeout")
)

func islocalip(ip net.IP) bool {
	ipv4 := ip.To4()
	if ipv4 != nil {
		for _, cidr := range localipv4 {
			_, inet, _ := net.ParseCIDR(cidr)
			if inet.Contains(ipv4) {
				return true
			}
		}
	}
	ipv6 := ip.To16()
	if ipv6 != nil {
		for _, cidr := range localipv6 {
			_, inet, _ := net.ParseCIDR(cidr)
			if inet.Contains(ipv6) {
				return true
			}
		}
	}
	return false
}
