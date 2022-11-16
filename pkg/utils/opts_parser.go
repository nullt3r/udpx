package utils

import (
	"flag"
	"fmt"

	"github.com/nullt3r/udpx/pkg/probes"
)

type Options struct {
	Arg_t  string
	Arg_tf string
	Arg_o  string
	Arg_c  int
	Arg_nr bool
	Arg_st int
	Arg_sp bool
	Arg_s  string
}

func ParseOptions() *Options {
	opts := &Options{}
	flag.StringVar(&opts.Arg_t, "t", "", "IP/CIDR to scan")
	flag.StringVar(&opts.Arg_tf, "tf", "", "File containing IPs/CIDRs to scan")
	flag.StringVar(&opts.Arg_o, "o", "", "Output file to write results")
	flag.StringVar(&opts.Arg_s, "s", "", fmt.Sprintf("Scan only for a specific service, one of: %s", probes.GetProbeNames()))
	flag.IntVar(&opts.Arg_c, "c", 32, "Maximum number of concurrent connections")
	flag.BoolVar(&opts.Arg_nr, "nr", false, "Do not randomize addresses")
	flag.IntVar(&opts.Arg_st, "w", 500, "Maximum time to wait for a response (socket timeout) in ms")
	flag.BoolVar(&opts.Arg_sp, "sp", false, "Show received packets (only first 32 bytes)")

	flag.Parse()

	return opts
}
