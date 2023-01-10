package scan

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
	"encoding/hex"

	"github.com/nullt3r/udpx/pkg/probes"
	"github.com/nullt3r/udpx/pkg/colors"
)

type Scanner struct {
	Target string
	Probes []probes.Probe
	Arg_st int
	Arg_sp bool
	Channel chan Message
}

type Message struct {
	Address      string     `json:"address"`
	Hostname     string     `json:"hostname"`
	Port         int        `json:"port"`
	Service      string     `json:"service"`
	ResponseData []byte     `json:"response_data"`
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
								s.Channel <- Message{Address: ip, Hostname: domain, Port: port, Service: probe.Name, ResponseData: recv_Data}
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
							s.Channel <- Message{Address: ip, Port: port, Service: probe.Name, ResponseData: recv_Data}
							return
						}
					}
				}()
			}
		}
	}
}
