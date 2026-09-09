package utils

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nullt3r/udpx/pkg/probes"
	"github.com/nullt3r/udpx/pkg/version"
)

type Options struct {
	Arg_t      string
	Arg_tf     string
	Arg_o      string
	Arg_c      int
	Arg_nr     bool
	Arg_st     int
	Arg_sp     bool
	Arg_s      string
	Arg_q           bool
	Arg_version     bool  // Version flag
	Arg_src_ip      string
	Arg_exclude     string
	Arg_excludefile string
	Arg_b           int   // Response buffer size (default 32)
	Arg_retries     int   // Extra retry attempts per probe on no-response (default 0)
	PositionalArgs  []string  // Positional arguments (targets) after all flags
}

func ParseOptions() *Options {
	opts := &Options{}
	flag.StringVar(&opts.Arg_t, "t", "", "IPs/hostnames/CIDRs/octet-ranges to scan (nmap target specification syntax)")
	flag.StringVar(&opts.Arg_tf, "tf", "", "File containing IPs/hostnames/CIDRs/octet-ranges to scan")
	flag.StringVar(&opts.Arg_o, "o", "", "Output file to write results")
	flag.StringVar(&opts.Arg_s, "s", "", fmt.Sprintf("Comma-separated list of services to scan, e.g. dns,ntp,snmp (available: %s)", probes.GetProbeNames()))
	flag.IntVar(&opts.Arg_c, "c", 32, "Maximum number of concurrent connections")
	flag.BoolVar(&opts.Arg_nr, "nr", false, "Do not randomize addresses")
	flag.IntVar(&opts.Arg_st, "w", 500, "Maximum time to wait for a response (socket timeout) in ms")
	flag.BoolVar(&opts.Arg_sp, "sp", false, "Show received packets (only first 32 bytes)")
	flag.IntVar(&opts.Arg_b, "b", 32, "Response buffer size in bytes (recommended 512 for complete response_data, default 32, max 4096)")
	flag.BoolVar(&opts.Arg_q, "q", false, "Quiet mode: suppress banner and progress log lines (results still emitted)")
	flag.IntVar(&opts.Arg_retries, "retries", 0, "Extra retry attempts per probe when no response is received within -w (0 = no retries, single attempt only, default). Each retry re-sends the SAME probe payload and waits -w ms again; it does not change -w itself. Added because udpx is a single-packet scanner over lossy UDP: a lost probe or a lost reply is otherwise a silent, permanent miss for that run, with no way to distinguish 'not listening' from 'packet lost'.")
	// Version flag: register both -v and -version as aliases pointing to the same variable
	flag.BoolVar(&opts.Arg_version, "v", false, "Print version and exit")
	flag.BoolVar(&opts.Arg_version, "version", false, "Print version and exit")
	flag.StringVar(&opts.Arg_src_ip, "src-ip", "", "Source IP to bind probes to (must be assigned to a local interface; overrides the kernel's default route-table pick)")
	flag.StringVar(&opts.Arg_exclude, "exclude", "", "Comma-separated list of targets to exclude from the scan (IPs/hostnames/CIDRs/octet-ranges)")
	flag.StringVar(&opts.Arg_excludefile, "excludefile", "", "Path to a file with targets to exclude, separated by newlines, spaces, or tabs; '#' starts an end-of-line comment")

	// Custom usage: long flags (name length > 2) are shown with a leading '--',
	// short flags keep '-'. Both prefixes are accepted at parse time — this is
	// purely a documentation convenience so '--src-ip', '--exclude' and
	// '--excludefile' read with a double-dash long-option style.
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		
		// Print banner with version
		fmt.Fprintf(out, `%s
        __  ______  ____ _  __
       / / / / __ \/ __ \ |/ /
      / / / / / / / /_/ /   / 
     / /_/ / /_/ / ____/   |  
     \____/_____/_/   /_/|_|  
         %s

%s`, "\033[36m", version.VersionLong(), "\033[0m")
		
		fmt.Fprintf(out, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(out, "  %s [options] [targets...]\n\n", os.Args[0])
		fmt.Fprintf(out, "Options:\n")
		
		// Track which flags we've already printed (for aliases)
		printed := make(map[string]bool)
		
		flag.VisitAll(func(f *flag.Flag) {
			// Skip if we already printed this flag (e.g., both -v and -version)
			if printed[f.Name] {
				return
			}
			
			// Special handling for -v and -version aliases
			if f.Name == "version" {
				printed["v"] = true
				printed["version"] = true
				fmt.Fprintf(out, "  -v | --version\n")
				_, usage := flag.UnquoteUsage(f)
				desc := strings.ReplaceAll(usage, "\n", "\n    \t")
				fmt.Fprintf(out, "    \t%s\n", desc)
				return
			}
			if f.Name == "v" {
				return // Skip, already handled with version
			}
			
			printed[f.Name] = true
			
			prefix := "-"
			if len(f.Name) > 2 {
				prefix = "--"
			}
			
			// Custom argument names matching nmap style
			var argName string
			switch f.Name {
			case "t":
				argName = "host1[ host2[ ...]]"
			case "tf":
				argName = "filename"
			case "o":
				argName = "filename"
			case "s":
				argName = "service1[,service2[,...]]"
			case "c":
				argName = "num"
			case "w":
				argName = "ms"
			case "b":
				argName = "bufsize"
			case "retries":
				argName = "num"
			case "src-ip":
				argName = "ip-address"
			case "exclude":
				argName = "host1[,host2[,...]]"
			case "excludefile":
				argName = "filename"
			default:
				name, _ := flag.UnquoteUsage(f)
				argName = name
			}
			
			line := fmt.Sprintf("  %s%s", prefix, f.Name)
			if argName != "" {
				line += " " + argName
			}
			fmt.Fprintln(out, line)
			
			_, usage := flag.UnquoteUsage(f)
			desc := strings.ReplaceAll(usage, "\n", "\n    \t")
			
			// Special handling for -b flag - don't append default since it's already in the description
			if f.Name == "b" {
				// Do nothing - description already contains the full text
			} else if !isZeroValue(f, f.DefValue) {
				// Append "(default ...)" for non-zero defaults, mirroring Go's
				// stdlib PrintDefaults. Strings are quoted, everything else bare.
				if g, ok := f.Value.(flag.Getter); ok {
					if _, isStr := g.Get().(string); isStr {
						desc += fmt.Sprintf(" (default %q)", f.DefValue)
					} else {
						desc += fmt.Sprintf(" (default %s)", f.DefValue)
					}
				} else {
					desc += fmt.Sprintf(" (default %s)", f.DefValue)
				}
			}
			fmt.Fprintln(out, "    \t"+desc)
		})
		fmt.Fprintf(out, "\nTargets:\n")
		fmt.Fprintf(out, "  Targets can be specified as positional arguments (like nmap) OR via -t/-tf flags.\n")
		fmt.Fprintf(out, "  Examples:\n")
		fmt.Fprintf(out, "    %s 192.168.1.1 192.168.2.0/24\n", os.Args[0])
		fmt.Fprintf(out, "    %s -t 192.168.1.1 192.168.2.0/24\n", os.Args[0])
		fmt.Fprintf(out, "    %s -tf targets.txt\n", os.Args[0])
		fmt.Fprintf(out, "    %s -q -o - -s dns,ntp,snmp,mssql -b 512 192.168.1.0/24\n", os.Args[0])
	}

	flag.Parse()

	// Collect positional arguments (remaining after flag parsing)
	opts.PositionalArgs = flag.Args()

	return opts
}

// isZeroValue mirrors the stdlib helper of the same name in flag/flag.go:
// it returns true if `value` is the zero value for the flag's underlying type.
// Used by the custom Usage above so we only annotate genuinely defaulted flags.
func isZeroValue(f *flag.Flag, value string) bool {
	// Best-effort: most flag types' zero stringification matches one of these.
	switch value {
	case "", "0", "false":
		return true
	}
	return false
}
