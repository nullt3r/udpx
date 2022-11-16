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
	"github.com/nullt3r/udpx/pkg/utils"
)

type Scanner struct {
	Ip     string
	Probes []probes.Probe
	Arg_st int
	Arg_sp bool
	Result chan string
}

func (s Scanner) Run() {
	socketTimeout := time.Duration(s.Arg_st) * time.Millisecond

	// If Ip is Ipv6
	if strings.Contains(s.Ip, ":") {
		s.Ip = "[" + s.Ip + "]"
	}

	for _, probe := range probes.Probes {
		func() {
			recv_Data := make([]byte, 32)

			c, err := net.Dial("udp", fmt.Sprint(s.Ip, ":", probe.Port))

			if err != nil {
				log.Printf("%s[!]%s [%s] Error connecting to host '%s': %s", utils.ColorRed, utils.ColorReset, probe.Name, s.Ip, err)
				return
			}

			defer c.Close()

			Data, err := hex.DecodeString(probe.Data)

			if err != nil {
				log.Fatalf("%s[!]%s Error in decoding probe data. Problem probe: '%s'", utils.ColorRed, utils.ColorReset, probe.Name)
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
				log.Printf("%s[*]%s %s:%d (%s)", utils.ColorCyan, utils.ColorReset, s.Ip, probe.Port, probe.Name)
				if s.Arg_sp {
					log.Printf("[+] Received packet: %s%s%s...", utils.ColorYellow, hex.EncodeToString(recv_Data), utils.ColorReset)
				}
				s.Result <- fmt.Sprintf("%s:%d	%s", s.Ip, probe.Port, probe.Name)
			}
		}()
	}
}
