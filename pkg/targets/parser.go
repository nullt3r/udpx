package targets

import (
	"fmt"
	"net"
	"strings"
)

// ParseMultiple expands a space-separated list of targets into a combined list
// of IP strings. Each target follows the same syntax as Parse():
//   - IP addresses: "192.0.2.1"
//   - CIDR blocks: "192.0.2.0/24"
//   - Octet ranges: "192.0.2.1-3" (expands to .1, .2, .3)
//   - Hostnames: "example.com" (resolved via net.LookupIP)
//
// Whitespace (spaces, tabs) separates individual targets. Multiple consecutive
// whitespace characters are treated as a single separator. Returns aggregated
// list of all IPs from all targets, or first error encountered.
func ParseMultiple(targetString string) ([]string, error) {
	targetString = strings.TrimSpace(targetString)
	if targetString == "" {
		return nil, nil
	}

	// Split by whitespace, filtering empty entries
	targets := strings.Fields(targetString) // Fields splits on any whitespace and removes empty strings
	if len(targets) == 0 {
		return nil, nil
	}

	var result []string
	for _, target := range targets {
		ips, err := Parse(target)
		if err != nil {
			return nil, err
		}
		result = append(result, ips...)
	}

	return result, nil
}

// Parse expands a single target (IP, CIDR, octet range, or hostname) into a
// list of IP strings. Extended Target Syntax supports:
//   - IP addresses: "192.0.2.1"
//   - CIDR blocks: "192.0.2.0/24"
//   - Octet ranges: "192.0.2.1-3" (expands to .1, .2, .3)
//   - Hostnames: "example.com" (resolved via net.LookupIP)
//
// Returns []string of IPs or a clear error. Empty input returns empty slice.
func Parse(target string) ([]string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, nil
	}

	// Try octet range first (e.g. "192.0.2.1-3")
	if strings.Contains(target, "-") && !strings.Contains(target, "/") {
		ips, err := parseOctetRange(target)
		if err == nil {
			return ips, nil
		}
		// Fall through to other parsers if octet range fails
	}

	// Try CIDR (e.g. "192.0.2.0/24")
	if strings.Contains(target, "/") {
		ips, err := parseCIDR(target)
		if err == nil {
			return ips, nil
		}
		// Fall through if CIDR fails
	}

	// Try as a bare IP (e.g. "192.0.2.1")
	if ip := net.ParseIP(target); ip != nil {
		return []string{ip.String()}, nil
	}

	// Try as a hostname (e.g. "example.com")
	ips, err := parseHostname(target)
	if err == nil && len(ips) > 0 {
		return ips, nil
	}

	// Nothing worked
	return nil, fmt.Errorf("not a valid IP, CIDR, octet range, or hostname: %q", target)
}

// parseOctetRange expands "192.0.2.1-3" to ["192.0.2.1", "192.0.2.2", "192.0.2.3"].
// Supports ranges in any octet position: "192.0-2.2" → ["192.0.2.2", "192.1.2.2", "192.2.2.2"].
func parseOctetRange(s string) ([]string, error) {
	octets := strings.Split(s, ".")
	if len(octets) != 4 {
		return nil, fmt.Errorf("invalid octet range format: %q", s)
	}

	// Parse each octet, which may be a bare number or a range "start-end"
	expanded := make([][]int, 4)
	for i, octet := range octets {
		if strings.Contains(octet, "-") {
			parts := strings.Split(octet, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid octet range in position %d: %q", i, octet)
			}
			var start, end int
			if _, err := fmt.Sscanf(parts[0], "%d", &start); err != nil {
				return nil, fmt.Errorf("invalid octet range start %q in position %d: %v", parts[0], i, err)
			}
			if _, err := fmt.Sscanf(parts[1], "%d", &end); err != nil {
				return nil, fmt.Errorf("invalid octet range end %q in position %d: %v", parts[1], i, err)
			}
			if start < 0 || start > 255 || end < 0 || end > 255 {
				return nil, fmt.Errorf("octet out of range [0-255] in position %d: %q", i, octet)
			}
			if start > end {
				start, end = end, start // Allow "3-1" → "1-3"
			}
			for j := start; j <= end; j++ {
				expanded[i] = append(expanded[i], j)
			}
		} else {
			var val int
			if _, err := fmt.Sscanf(octet, "%d", &val); err != nil {
				return nil, fmt.Errorf("invalid octet %q in position %d: %v", octet, i, err)
			}
			if val < 0 || val > 255 {
				return nil, fmt.Errorf("octet out of range [0-255] in position %d: %q", i, octet)
			}
			expanded[i] = []int{val}
		}
	}

	// Cartesian product: build all combinations
	var result []string
	for _, o0 := range expanded[0] {
		for _, o1 := range expanded[1] {
			for _, o2 := range expanded[2] {
				for _, o3 := range expanded[3] {
					result = append(result, fmt.Sprintf("%d.%d.%d.%d", o0, o1, o2, o3))
				}
			}
		}
	}
	return result, nil
}

// parseCIDR expands a CIDR block like "192.0.2.0/24" to all IPs in the range.
func parseCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	var result []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		result = append(result, ip.String())
	}
	return result, nil
}

// parseHostname resolves a hostname to one or more IPs via net.LookupIP.
func parseHostname(hostname string) ([]string, error) {
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return nil, fmt.Errorf("hostname lookup failed: %w", err)
	}
	var result []string
	for _, ip := range ips {
		result = append(result, ip.String())
	}
	return result, nil
}

// incrementIP increments an IP address in-place for CIDR iteration.
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
