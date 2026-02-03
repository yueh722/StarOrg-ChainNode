package utils

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// checkIpOrDomain validates if the input address is a valid IP (IPv4/IPv6) or resolvable domain name.
// Parameters:
//
//	address - The string to be checked (IP address or domain name, may contain leading/trailing spaces)
//
// Returns:
//
//	nil if the address is a valid IP or resolvable domain; non-nil error with detailed message if invalid
func CheckIpOrDomain(address string) error {
	// Remove leading and trailing whitespace to avoid format issues
	cleanAddr := strings.TrimSpace(address)
	if cleanAddr == "" {
		return errors.New("address cannot be empty")
	}

	// Validate if the address is a legal IP (supports both IPv4 and IPv6)
	ip := net.ParseIP(cleanAddr)
	if ip != nil {
		// Valid IP address, return nil (success)
		return nil
	}

	// Not an IP address, attempt to resolve domain name to IP addresses (IPv4/IPv6)
	ips, err := net.LookupIP(cleanAddr)
	if err != nil {
		// Domain resolution failed, return combined error message
		return fmt.Errorf("domain %s resolution failed, %v", cleanAddr, err)
	}

	// Edge case: resolution succeeded but no IP addresses returned (extremely rare)
	if len(ips) == 0 {
		return fmt.Errorf("domain %s resolution succeeded but no IP addresses were returned", cleanAddr)
	}

	// Domain resolved successfully, return nil (success)
	return nil
}
