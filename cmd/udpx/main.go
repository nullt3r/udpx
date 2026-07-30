package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nullt3r/udpx/pkg/colors"
	"github.com/nullt3r/udpx/pkg/probes"
	"github.com/nullt3r/udpx/pkg/scan"
	"github.com/nullt3r/udpx/pkg/targets"
	"github.com/nullt3r/udpx/pkg/utils"
	"github.com/nullt3r/udpx/pkg/version"
)

func main() {
	// Parse options FIRST so -q / -o '-' can influence where the banner goes
	// (or whether it's printed at all).
	opts := utils.ParseOptions()

	// Validate buffer size
	const MaxBufferSize = 4096
	if opts.Arg_b < 32 {
		log.Fatalf("%s[!]%s Buffer size must be at least 32 bytes (got %d)", colors.SetColor().Red, colors.SetColor().Reset, opts.Arg_b)
	}
	if opts.Arg_b > MaxBufferSize {
		log.Fatalf("%s[!]%s Buffer size cannot exceed %d bytes (got %d)", colors.SetColor().Red, colors.SetColor().Reset, MaxBufferSize, opts.Arg_b)
	}

	// Handle -v/--version flag
	if opts.Arg_version {
		fmt.Println(version.VersionLong())
		os.Exit(0)
	}

	// Output channels:
	//   - stdoutJsonl == true  => "-o -" mode: stdout is reserved for JSONL records,
	//                             so the banner goes to stderr and so do log lines.
	//   - opts.Arg_q  == true  => quiet mode: suppress banner and silence log.Printf.
	stdoutJsonl := opts.Arg_o == "-"
	bannerW := io.Writer(os.Stdout)
	if stdoutJsonl {
		bannerW = os.Stderr
	}
	if opts.Arg_q {
		// Drop banner entirely; route Go's default logger (which writes to stderr) to /dev/null.
		bannerW = io.Discard
		log.SetOutput(io.Discard)
	}

	fmt.Fprintf(bannerW, `%s
        __  ______  ____ _  __
       / / / / __ \/ __ \ |/ /
      / / / / / / / /_/ /   / 
     / /_/ / /_/ / ____/   |  
     \____/_____/_/   /_/|_|  
         %s

%s`, colors.SetColor().Cyan, version.VersionLong(), colors.SetColor().Reset)

	var toscan []string

	if len(opts.Arg_t) == 0 && len(opts.Arg_tf) == 0 && len(opts.PositionalArgs) == 0 {
		log.Fatalf("%s[!]%s Error, targets required via positional arguments, -t, or -tf\n", colors.SetColor().Red, colors.SetColor().Reset)
	}

	if len(opts.Arg_tf) != 0 {
		targetList, err := utils.ReadFile(opts.Arg_tf)
		if err != nil {
			log.Fatalf("%s[!]%s Error while loading targets from file: %s", colors.SetColor().Red, colors.SetColor().Reset, err)
		}
		for _, target := range targetList {
			ips, err := targets.Parse(target)
			if err != nil {
				log.Fatalf("%s[!]%s Error parsing target %q: %s", colors.SetColor().Red, colors.SetColor().Reset, target, err)
			}
			toscan = append(toscan, ips...)
		}
	} else if len(opts.Arg_t) != 0 || len(opts.PositionalArgs) > 0 {
		// Combine -t flag value with any positional arguments
		// This handles cases like: -t 192.168.0.1 192.168.0.2 (both get combined)
		var targetStr string
		if len(opts.Arg_t) > 0 {
			targetStr = opts.Arg_t
			if len(opts.PositionalArgs) > 0 {
				targetStr += " " + strings.Join(opts.PositionalArgs, " ")
			}
		} else {
			targetStr = strings.Join(opts.PositionalArgs, " ")
		}

		ips, err := targets.ParseMultiple(targetStr)
		if err != nil {
			log.Fatalf("%s[!]%s Error parsing targets %q: %s", colors.SetColor().Red, colors.SetColor().Reset, targetStr, err)
		}
		toscan = append(toscan, ips...)
	}

	if len(opts.Arg_s) != 0 {
		// Split comma-separated services
		requestedServices := strings.Split(opts.Arg_s, ",")
		
		// Trim whitespace from each service name
		for i := range requestedServices {
			requestedServices[i] = strings.TrimSpace(requestedServices[i])
		}
		
		// Build a map of available probe names for quick lookup
		availableProbes := make(map[string]probes.Probe)
		for _, probe := range probes.Probes {
			availableProbes[probe.Name] = probe
		}
		
		// Validate all requested services and build selected probes
		var selectedProbes []probes.Probe
		var notFound []string
		
		for _, service := range requestedServices {
			if probe, exists := availableProbes[service]; exists {
				selectedProbes = append(selectedProbes, probe)
			} else {
				notFound = append(notFound, service)
			}
		}
		
		// If any services were not found, fail with error
		if len(notFound) > 0 {
			log.Fatalf("%s[!]%s Service(s) not supported: %s\n     Available services: %s", 
				colors.SetColor().Red, colors.SetColor().Reset, 
				strings.Join(notFound, ", "), 
				probes.GetProbeNames())
		}
		
		// Replace global Probes with selected probes
		probes.Probes = selectedProbes
	}


	toscan = utils.Deduplicate(toscan)
	toscan_count := len(toscan)

	// Apply -exclude / -excludefile BEFORE shuffle and BEFORE any probes are
	// sent. Excluded targets must never receive a single packet.
	if len(opts.Arg_exclude) != 0 || len(opts.Arg_excludefile) != 0 {
		excl, err := utils.BuildExcludeSet(opts.Arg_exclude, opts.Arg_excludefile)
		if err != nil {
			log.Fatalf("%s[!]%s %s", colors.SetColor().Red, colors.SetColor().Reset, err)
		}
		filtered := toscan[:0]
		removed := 0
		for _, ip := range toscan {
			if _, skip := excl[ip]; skip {
				removed++
				continue
			}
			filtered = append(filtered, ip)
		}
		toscan = filtered
		toscan_count = len(toscan)
		log.Printf("[+] Excluded %d target(s) via -exclude/-excludefile", removed)
	}

	// Resolve and validate -src-ip, if given. Fail fast with a clear message
	// rather than letting every probe blow up with "cannot assign requested
	// address" at dial time.
	var srcIP net.IP
	if len(opts.Arg_src_ip) != 0 {
		ip, err := utils.ValidateLocalIP(opts.Arg_src_ip)
		if err != nil {
			log.Fatalf("%s[!]%s -src-ip: %s", colors.SetColor().Red, colors.SetColor().Reset, err)
		}
		srcIP = ip
		log.Printf("[+] Binding probes to source IP %s", srcIP)
	}

	if !opts.Arg_nr {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(toscan_count, func(i, j int) { toscan[i], toscan[j] = toscan[j], toscan[i] })
	}

	goroutineLimit := opts.Arg_c
	guard := make(chan struct{}, goroutineLimit)

	var wg sync.WaitGroup

	log.Printf("[+] Starting UDP scan on %d target(s)", toscan_count)

	wg.Add(toscan_count)

	comm := make(chan scan.Message, 10)

	go func() {
		for _, t := range toscan {
			guard <- struct{}{}
			go func(t string) {
				defer wg.Done()
				scanner := scan.Scanner{Target: t, Probes: probes.Probes, Arg_st: opts.Arg_st, Arg_sp: opts.Arg_sp, Arg_b: opts.Arg_b, SrcIP: srcIP, Channel: comm}
				scanner.Run()
				<-guard
			}(t)
		}

	}()

	go func() {
		wg.Wait()
		close(comm)
	}()

	// Output sink for JSONL records:
	//   - "-o -"      => os.Stdout (no file handle to manage)
	//   - "-o <path>" => append-mode file (opened once up front, closed at exit)
	//   - empty       => no JSONL emission
	var jsonlW io.Writer
	if stdoutJsonl {
		jsonlW = os.Stdout
	} else if len(opts.Arg_o) != 0 {
		// Note: original code re-opened the file for every record. We open once
		// here in append+create mode so writes are buffered by the OS and the
		// fd is closed cleanly at exit.
		f, err := os.OpenFile(opts.Arg_o, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			log.Fatalf("%s[!]%s Error creating output file: %s", colors.SetColor().Red, colors.SetColor().Reset, err)
		}
		defer f.Close()
		jsonlW = f
		log.Printf("[+] Results will be written to: %s", opts.Arg_o)
	}

	for message := range comm {
		log.Printf("%s[*]%s %s:%d (%s)", colors.SetColor().Cyan, colors.SetColor().Reset, message.Address, message.Port, message.Service)

		if opts.Arg_sp {
			log.Printf("[+] Received packet: %s%s%s...", colors.SetColor().Yellow, hex.EncodeToString(message.ResponseData), colors.SetColor().Reset)
		}

		if jsonlW != nil {
			b, err := json.Marshal(&message)
			if err != nil {
				log.Fatalf("%s[!]%s Error: %s", colors.SetColor().Red, colors.SetColor().Reset, err)
			}
			if _, err := jsonlW.Write(append(b, '\n')); err != nil {
				log.Fatalf("%s[!]%s Error writing output: %s", colors.SetColor().Red, colors.SetColor().Reset, err)
			}
		}
	}

	<-comm

	log.Print("[+] Scan completed")
}
