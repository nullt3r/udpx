package utils

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/nullt3r/udpx/pkg/targets"
)

func EscapeByteArray(message []byte) []byte {
    var result []byte
    for _, b := range message {
        if b > 127 || b == '"' || b == '\n' || b == '\t' || (b <= ' ' && b >= 0) {
            result = append(result, []byte(fmt.Sprintf("\\x%02x", b))...)
        } else {
            result = append(result, b)
        }
    }
    return result
}

func IpsFromCidr(cidr string) ([]string, error) {
	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string

	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	// If mask is /32 or /31
	//if len(ips) <= 2 {
	//	return ips, nil
	//}

	// remove network address and broadcast address
	//return ips[1 : len(ips)-1], nil

	return ips, nil
}

func ReadFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func WriteChannel(lines chan string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}


func Deduplicate(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	// If the key(values of the slice) is not equal
	// to the already present value in new slice (list)
	// then we append it. else we jump on another element.
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// ValidateLocalIP parses s as an IP and verifies it is currently assigned to
// some local network interface. Returns the parsed net.IP on success.
//
// Used to back the -src-ip flag: failing fast here gives a clear error
// instead of a per-probe "bind: cannot assign requested address" later.
func ValidateLocalIP(s string) (net.IP, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("not a valid IP address: %q", s)
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("enumerating local interface addresses: %w", err)
	}
	for _, a := range addrs {
		var local net.IP
		switch v := a.(type) {
		case *net.IPNet:
			local = v.IP
		case *net.IPAddr:
			local = v.IP
		}
		if local != nil && local.Equal(ip) {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("IP %s is not assigned to any local interface", ip)
}

// BuildExcludeSet expands exclude targets (IPs, CIDRs, octet-ranges, hostnames)
// from both a comma-separated list (`list`) and a file (`file`) into a set of
// IP strings to remove from the scan set. Uses the Extended Target Syntax
// (pkg/targets.Parse) so both --exclude and --excludefile support the same
// target formats.
//
// File format: targets are separated by newlines, spaces, or tabs.
// A '#' marks an end-of-line comment — everything from '#' to
// end-of-line is ignored on each line.
//
// Malformed entries fail fast with a clear error (consistent with -src-ip).
func BuildExcludeSet(list, file string) (map[string]struct{}, error) {
	out := make(map[string]struct{})

	add := func(entry string) error {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			return nil
		}
		ips, err := targets.Parse(entry)
		if err != nil {
			return fmt.Errorf("exclude: %w", err)
		}
		for _, ip := range ips {
			out[ip] = struct{}{}
		}
		return nil
	}

	if list != "" {
		for _, e := range strings.Split(list, ",") {
			if err := add(e); err != nil {
				return nil, err
			}
		}
	}
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("excludefile: %w", err)
		}
		// Per-line: strip '#'-to-EOL comments first, then split the
		// remaining content on spaces/tabs/newlines. Doing the comment
		// strip line-by-line (rather than treating '#' as a token-prefix
		// marker) is necessary so an inline comment after a target on the
		// same line works: "192.0.2.1   # skip the gateway".
		var b strings.Builder
		for _, line := range strings.Split(string(data), "\n") {
			if i := strings.IndexByte(line, '#'); i >= 0 {
				line = line[:i]
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
		for _, e := range strings.FieldsFunc(b.String(), func(r rune) bool {
			return r == '\n' || r == '\r' || r == ' ' || r == '\t'
		}) {
			if err := add(e); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
