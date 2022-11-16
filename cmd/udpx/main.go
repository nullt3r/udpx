package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/nullt3r/udpx/pkg/probes"
	"github.com/nullt3r/udpx/pkg/scan"
	"github.com/nullt3r/udpx/pkg/utils"
)

func main() {
	fmt.Printf(`%s
      __  ______  ____ _  __
     / / / / __ \/ __ \ |/ /
    / / / / / / / /_/ /   / 
   / /_/ / /_/ / ____/   |  
   \____/_____/_/   /_/|_|  
           			by @nullt3r

%s`, utils.ColorCyan, utils.ColorReset)

	opts := utils.ParseOptions()

	var targets []string
	var ips []string

	if len(opts.Arg_t) == 0 && len(opts.Arg_tf) == 0 {
		log.Fatalf("%s[!]%s Error, argument -t or -tf is required\n", utils.ColorRed, utils.ColorReset)
	}

	if len(opts.Arg_tf) != 0 {
		val, err := utils.ReadFile(opts.Arg_tf)
		if err != nil {
			log.Fatalf("%s[!]%s Error while loading targets from file: %s", utils.ColorRed, utils.ColorReset, err)
			return
		}
		targets = val
	} else if len(opts.Arg_t) != 0 {
		targets = []string{opts.Arg_t}
	}

	for _, target := range targets {
		if strings.Contains(target, "/") {
			val, err := utils.IpsFromCidr(target)

			if err != nil {
				log.Fatalf("%s[!]%s Error parsing IP range: %s", utils.ColorRed, utils.ColorReset, err)
				return
			}

			ips = append(ips, val...)
		} else {
			ips = append(ips, target)
		}
	}

	if len(opts.Arg_s) != 0 {
		for n, probe := range probes.Probes {
			if probe.Name == opts.Arg_s {
				probes.Probes = []probes.Probe{probe}
				break
			}
			if n == len(probes.Probes)-1 {
				log.Fatalf("%s[!]%s Service '%s' is not supported", utils.ColorRed, utils.ColorReset, opts.Arg_s)
			}
		}
	}

	ips = utils.Deduplicate(ips)
	ips_count := len(ips)
	probe_count := len(probes.Probes)

	if !opts.Arg_nr {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(ips_count, func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })
	}

	goroutineLimit := opts.Arg_c
	guard := make(chan struct{}, goroutineLimit)

	var wg sync.WaitGroup

	log.Printf("[+] Starting UDP scan on %d target(s)", ips_count)

	wg.Add(ips_count)

	result := make(chan string, ips_count*probe_count)

	for _, ip := range ips {
		guard <- struct{}{}
		go func(ip string) {
			defer wg.Done()
			scanner := scan.Scanner{Ip: ip, Probes: probes.Probes, Arg_st: opts.Arg_st, Arg_sp: opts.Arg_sp, Result: result}
			scanner.Run()
			<-guard
		}(ip)
	}

	wg.Wait()

	close(result)

	if len(opts.Arg_o) != 0 {
		log.Printf("[+] Writing results to '%s'", opts.Arg_o)
		utils.WriteChannel(result, opts.Arg_o)
	}

	log.Print("[+] Scan completed")

	<-result
}
