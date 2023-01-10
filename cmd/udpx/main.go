package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
	"os"

	"github.com/nullt3r/udpx/pkg/probes"
	"github.com/nullt3r/udpx/pkg/scan"
	"github.com/nullt3r/udpx/pkg/utils"
	"github.com/nullt3r/udpx/pkg/colors"
)

func main() {
	fmt.Printf(`%s
        __  ______  ____ _  __
       / / / / __ \/ __ \ |/ /
      / / / / / / / /_/ /   / 
     / /_/ / /_/ / ____/   |  
     \____/_____/_/   /_/|_|  
         v1.0.5, by @nullt3r

%s`, colors.SetColor().Cyan, colors.SetColor().Reset)

	opts := utils.ParseOptions()

	var targets []string
	var toscan []string

	if len(opts.Arg_t) == 0 && len(opts.Arg_tf) == 0 {
		log.Fatalf("%s[!]%s Error, argument -t or -tf is required\n", colors.SetColor().Red, colors.SetColor().Reset)
	}

	if len(opts.Arg_tf) != 0 {
		val, err := utils.ReadFile(opts.Arg_tf)
		if err != nil {
			log.Fatalf("%s[!]%s Error while loading targets from file: %s", colors.SetColor().Red, colors.SetColor().Reset, err)
		}
		targets = val
	} else if len(opts.Arg_t) != 0 {
		targets = []string{opts.Arg_t}
	}

	for _, target := range targets {
		if strings.Contains(target, "/") {
			val, err := utils.IpsFromCidr(target)

			if err != nil {
				log.Fatalf("%s[!]%s Error parsing IP range: %s", colors.SetColor().Red, colors.SetColor().Reset, err)
			}

			toscan = append(toscan, val...)
		} else {
			toscan = append(toscan, target)
		}
	}

	if len(opts.Arg_s) != 0 {
		for n, probe := range probes.Probes {
			if probe.Name == opts.Arg_s {
				probes.Probes = []probes.Probe{probe}
				break
			}
			if n == len(probes.Probes)-1 {
				log.Fatalf("%s[!]%s Service '%s' is not supported", colors.SetColor().Red, colors.SetColor().Reset, opts.Arg_s)
			}
		}
	}

	toscan = utils.Deduplicate(toscan)
	toscan_count := len(toscan)

	if !opts.Arg_nr {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(toscan_count, func(i, j int) { toscan[i], toscan[j] = toscan[j], toscan[i] })
	}

	goroutineLimit := opts.Arg_c
	guard := make(chan struct{}, goroutineLimit)

	var wg sync.WaitGroup

	log.Printf("[+] Starting UDP scan on %d target(s)", toscan_count)

	wg.Add(toscan_count)

	result := make(chan string)

	go func() {
		for _, t := range toscan {
			guard <- struct{}{}
			go func(t string) {
				defer wg.Done()
				scanner := scan.Scanner{Target: t, Probes: probes.Probes, Arg_st: opts.Arg_st, Arg_sp: opts.Arg_sp, Result: result}
				scanner.Run()
				<-guard
			}(t)
		}

	}()

	go func() {
		wg.Wait()
		close(result)
	}()

	if len(opts.Arg_o) != 0 {
		log.Printf("[+] Using output file '%s'", opts.Arg_o)

		f, err := os.Create(opts.Arg_o)

		if err != nil {
			log.Fatalf("%s[!]%s Can't create output file: %s", colors.SetColor().Red, colors.SetColor().Reset, err)
		}

		defer f.Close()

		for r := range result {
			f.WriteString(r + "\n")
		}

	}

	<-result

	log.Print("[+] Scan completed")
}
