package version

import "fmt"

// Semantic versioning
const (
	MajorVersion = 1
	MinorVersion = 0
	PatchVersion = 8
)

// Version returns the full version string
func Version() string {
	return fmt.Sprintf("v%d.%d.%d", MajorVersion, MinorVersion, PatchVersion)
}

// VersionLong returns a longer version string with build info
func VersionLong() string {
	return fmt.Sprintf("udpx %s, by @nullt3r", Version())
}
