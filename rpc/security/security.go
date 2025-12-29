package security

import (
	"net"
	"net/http"
)

type SecurityOption struct {
	FWMode         FirewallMode
	AllowOnlyHosts []string // 这里只放 IP
}

type FirewallMode string

const (
	AllowOnly FirewallMode = "allow_only"
	AllowAll  FirewallMode = "allow_all"
	LocalOnly FirewallMode = "local_only"
)

type Firewall struct {
	mode    FirewallMode
	allowIP map[string]struct{}
}

func NewFirewall(opt SecurityOption) *Firewall {
	fw := &Firewall{
		mode:    opt.FWMode,
		allowIP: make(map[string]struct{}),
	}

	for _, v := range opt.AllowOnlyHosts {
		ip := net.ParseIP(v)
		if ip == nil {
			continue
		}

		// 统一 IP 表示
		if ip4 := ip.To4(); ip4 != nil {
			fw.allowIP[ip4.String()] = struct{}{}
		} else {
			fw.allowIP[ip.To16().String()] = struct{}{}
		}
	}

	return fw
}

func (fw *Firewall) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if isLocalIP(clientIP(r.RemoteAddr)) {
			next.ServeHTTP(w, r)
			return
		}
		
		switch fw.mode {

		case AllowAll:
			next.ServeHTTP(w, r)
			return

		case LocalOnly:
			// 仅允许 localhost / 127.0.0.1 / ::1
			if isLocalIP(clientIP(r.RemoteAddr)) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
			return

		case AllowOnly:
			if fw.isAllowed(r) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
			return

		default:
			next.ServeHTTP(w, r)
			//http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	})
}

func (fw *Firewall) isAllowed(r *http.Request) bool {
	ip := clientIP(r.RemoteAddr)
	if ip == nil {
		return false
	}

	if ip4 := ip.To4(); ip4 != nil {
		_, ok := fw.allowIP[ip4.String()]
		return ok
	}

	_, ok := fw.allowIP[ip.To16().String()]
	return ok
}

func clientIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return net.ParseIP(remoteAddr)
	}
	return net.ParseIP(host)
}

func isLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	return false
}
