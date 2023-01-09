package utils

import (
	"bufio"
	"fmt"
	"net"
	"os"
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
