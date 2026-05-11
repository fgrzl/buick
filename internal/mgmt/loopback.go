package mgmt

import (
	"net"
	"strings"
)

// IsLoopbackClient reports whether a TCP client address string is loopback-only.
func IsLoopbackClient(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	if host == "" {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	// IPv4-mapped IPv6 loopback
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 127
	}
	return false
}

// IsMgmtPath reports whether the request path is reserved for management.
func IsMgmtPath(path string) bool {
	return strings.HasPrefix(path, "/_buick/")
}
