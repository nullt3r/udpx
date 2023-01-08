package scan

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/nullt3r/udpx/pkg/probes"
	"github.com/nullt3r/udpx/pkg/colors"
	"github.com/nullt3r/udpx/pkg/utils"
)

type Scanner struct {
	Target string
	Probes []probes.Probe
	Arg_st int
	Arg_sp bool
	Result chan string
}

func (s Scanner) Run() {
	socketTimeout := time.Duration(s.Arg_st) * time.Millisecond
	target := s.Target

	// Check if input is a domain
	if net.ParseIP(target) == nil {
		// Resolve domain to IP
		ips, err := net.LookupIP(target)
		if err != nil {
			log.Printf("%s[!]%s Error resolving domain '%s': %s", colors.SetColor().Red, colors.SetColor().Reset, target, err)
			return
		}
		domain := target

		// Dial for each IP of domain
		for _, ip := range ips {
			ip := ip.String()
			// If IP is IPv6
			if strings.Contains(ip, ":") {
				ip = "[" + ip + "]"
			}
			for _, probe := range probes.Probes {
				for _, port := range probe.Port {
					func() {

						for _, payload := range probe.Payloads {
							recv_Data := make([]byte, 32)

							c, err := net.Dial("udp", fmt.Sprint(ip, ":", port))

							if err != nil {
								log.Printf("%s[!]%s [%s] Error connecting to host '%s': %s", colors.SetColor().Red, colors.SetColor().Reset, probe.Name, ip, err)
								return
							}

							defer c.Close()

							Data, err := hex.DecodeString(payload)

							if err != nil {
								log.Fatalf("%s[!]%s Error in decoding payload. Problem probe: '%s'", colors.SetColor().Red, colors.SetColor().Reset, probe.Name)
							}

							_, err = c.Write([]byte(Data))

							if err != nil {
								return
							}

							c.SetReadDeadline(time.Now().Add(socketTimeout))

							recv_length, err := bufio.NewReader(c).Read(recv_Data)

							if err != nil {
								return
							}

							if recv_length != 0 {
								log.Printf("%s[*]%s %s:%d (%s)", colors.SetColor().Cyan, colors.SetColor().Reset, ip, port, probe.Name)
								if s.Arg_sp {
									log.Printf("[+] Received packet: %s%s%s...", colors.SetColor().Yellow, hex.EncodeToString(recv_Data), colors.SetColor().Reset)
								}
								
								s.Result <- fmt.Sprintf(`{"address": "%s", "hostname": "%s", "protocol": "udp", "portid": "%d", "port_state": "open", "service_name": "%s", "service_product": null, "service_version": null, "extrainfo": "%s"}`, ip, domain, port, probe.Name, utils.EscapeByteArray(recv_Data))
								return
							}
						}
					}()
				}
			}
		}
	} else {
		// Dial for a single IP
		ip := target
		// If IP is IPv6
		if strings.Contains(ip, ":") {
			ip = "[" + ip + "]"
		}
		for _, probe := range probes.Probes {
			for _, port := range probe.Port {
				func() {
					for _, payload := range probe.Payloads {
						recv_Data := make([]byte, 32)

						c, err := net.Dial("udp", fmt.Sprint(ip, ":", port))

						if err != nil {
							log.Printf("%s[!]%s [%s] Error connecting to host '%s': %s", colors.SetColor().Red, colors.SetColor().Reset, probe.Name, ip, err)
							return
						}

						defer c.Close()

						Data, err := hex.DecodeString(payload)

						if err != nil {
							log.Fatalf("%s[!]%s Error in decoding payload. Problem probe: '%s'", colors.SetColor().Red, colors.SetColor().Reset, probe.Name)
						}

						_, err = c.Write([]byte(Data))

						if err != nil {
							return
						}

						c.SetReadDeadline(time.Now().Add(socketTimeout))

						recv_length, err := bufio.NewReader(c).Read(recv_Data)

						if err != nil {
							return
						}

						if recv_length != 0 {
							log.Printf("%s[*]%s %s:%d (%s)", colors.SetColor().Cyan, colors.SetColor().Reset, ip, port, probe.Name)
							if s.Arg_sp {
								log.Printf("[+] Received packet: %s%s%s...", colors.SetColor().Yellow, hex.EncodeToString(recv_Data), colors.SetColor().Reset)
							}
							s.Result <- fmt.Sprintf(`{"address": "%s", "hostname": null, "protocol": "udp", "portid": %d, "port_state": "open", "service_name": "%s", "service_product": null, "service_version": null, "extrainfo": "%s"}`, ip, port, probe.Name, utils.EscapeByteArray(recv_Data))
							return
						}
					}
				}()
			}
		}
	}
}
