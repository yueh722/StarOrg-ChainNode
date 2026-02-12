package security

import (
	"net"
	"net/http"
)

type FirewallPolicy struct {
	FWMode         FirewallMode
	AllowOnlyHosts []string //ip slice
}

type SecurityOption struct {
	//FWMode         FirewallMode
	//AllowOnlyHosts []string //ip slice

	FetchPolicy func() FirewallPolicy
	TokenSecret []byte
}

type FirewallMode string

const (
	AllowOnly FirewallMode = "allow_only"
	AllowAll  FirewallMode = "allow_all"
	LocalOnly FirewallMode = "local_only"
)

type Firewall struct {
	fetchPolicy func() FirewallPolicy
}

func NewFirewall(opt SecurityOption) *Firewall {
	fw := &Firewall{
		fetchPolicy: opt.FetchPolicy,
	}

	return fw
}

func (fw *Firewall) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if isLocalIP(clientIP(r.RemoteAddr)) {
			next.ServeHTTP(w, r)
			return
		}

		switch fw.fetchPolicy().FWMode {

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

func (fw *Firewall) getAllowIP() map[string]struct{} {

	var ips = make(map[string]struct{})

	for _, v := range fw.fetchPolicy().AllowOnlyHosts {
		ip := net.ParseIP(v)
		if ip == nil {
			continue
		}

		// 统一 IP 表示
		if ip4 := ip.To4(); ip4 != nil {
			ips[ip4.String()] = struct{}{}
		} else {
			ips[ip.To16().String()] = struct{}{}
		}
	}

	return ips
}

func (fw *Firewall) isAllowed(r *http.Request) bool {
	ip := clientIP(r.RemoteAddr)
	if ip == nil {
		return false
	}

	allowIP := fw.getAllowIP()
	if ip4 := ip.To4(); ip4 != nil {
		_, ok := allowIP[ip4.String()]
		return ok
	}

	_, ok := allowIP[ip.To16().String()]
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

func parseCIDR() {
	//ipv4addr, ipv4net := net.ParseCIDR("10.0.0.0/24")
}
