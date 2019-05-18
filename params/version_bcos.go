package params

import (
	"fmt"
)

const (
	BCOSwinVersionMajor = 0        // Major version component of the current release
	BCOSVersionMinor    = 0        // Minor version component of the current release
	BCOSVersionPatch    = 1        // Patch version component of the current release
	BCOSVersionMeta     = "stable" // Version metadata to append to the version string
)

// BcosVersion holds the textual version string.
var BcosVersion = func() string {
	return fmt.Sprintf("%d.%d.%d", BCOSwinVersionMajor, BCOSVersionMinor, BCOSVersionPatch)
}()

// BcosVersionWithMeta holds the textual version string including the metadata.
var BcosVersionWithMeta = func() string {
	v := BcosVersion
	if BCOSVersionMeta != "" {
		v += "-" + BCOSVersionMeta
	}
	return v
}()

// BcosArchiveVersion holds the textual version string used for Geth archives.
// e.g. "1.8.11-dea1ce05" for stable releases, or
//      "1.8.13-unstable-21c059b6" for unstable releases
func BcosArchiveVersion(gitCommit string) string {
	vsn := BcosVersion
	if BCOSVersionMeta != "stable" {
		vsn += "-" + BCOSVersionMeta
	}
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	return vsn
}

func BCOSVersionWithCommit(gitCommit string) string {
	vsn := BcosVersionWithMeta
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	return vsn
}
