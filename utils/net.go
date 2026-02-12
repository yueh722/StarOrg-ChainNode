package utils

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/slice"
)

// checkIpOrDomain validates if the input address is a valid IP (IPv4/IPv6) or resolvable domain name.
// Parameters:
//
//	address - The string to be checked (IP address or domain name, may contain leading/trailing spaces)
//
// Returns:
//
//	nil if the address is a valid IP or resolvable domain; non-nil error with detailed message if invalid
func CheckIpOrDomain(address string) (string, error) {
	// Remove leading and trailing whitespace to avoid format issues
	cleanAddr := strings.TrimSpace(address)
	if cleanAddr == "" {
		return "", errors.New("address cannot be empty")
	}

	// Validate if the address is a legal IP (supports both IPv4 and IPv6)
	ip := net.ParseIP(cleanAddr)
	if ip != nil {
		// Valid IP address, return nil (success)
		return ip.String(), nil
	}

	// Not an IP address, attempt to resolve domain name to IP addresses (IPv4/IPv6)
	ips, err := net.LookupIP(cleanAddr)
	if err != nil {
		// Domain resolution failed, return combined error message
		return "", fmt.Errorf("domain %s resolution failed, %v", cleanAddr, err)
	}

	// Edge case: resolution succeeded but no IP addresses returned (extremely rare)
	if len(ips) == 0 {
		return "", fmt.Errorf("domain %s resolution succeeded but no IP addresses were returned", cleanAddr)
	}

	ipstrings := slice.Map(ips, func(_ int, ip net.IP) string {
		return ip.String()
	})

	// Domain resolved successfully, return nil (success)
	return strings.Join(ipstrings, ","), nil
}

func CheckURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	if parsedURL.Host == "" {
		return errors.New("URL missing host address")
	}

	host, port, err := net.SplitHostPort(parsedURL.Host)
	if err != nil {
		return fmt.Errorf("failed to split host and port: %w", err)
	}

	if host == "" {
		return errors.New("URL host address is empty")
	}

	portNum, err := convertor.ToInt(port)
	if portNum <= 0 || portNum > 65535 {
		return errors.New("port number illegal")
	}

	return nil
}
