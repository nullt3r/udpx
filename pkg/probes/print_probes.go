package probes

func GetProbeNames() string {
	var avail_probes string

	for i := range Probes {
		avail_probes += Probes[i].Name
		if i != len(Probes)-1 {
			avail_probes += ", "
		}
	}

	return avail_probes
}
