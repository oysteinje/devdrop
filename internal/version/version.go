package version

// Version is set at build time via ldflags
var Version = "dev"

// GetVersion returns the current version
func GetVersion() string {
	if Version == "" {
		return "dev"
	}
	return Version
}