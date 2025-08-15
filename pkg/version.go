package pkg

import (
	"fmt"
	"strings"
	"time"

	"go.rtnl.ai/x/semver"
)

// Version component constants for the current build.
const (
	VersionMajor         = 0
	VersionMinor         = 2
	VersionPatch         = 0
	VersionReleaseLevel  = "alpha"
	VersionReleaseNumber = 2
)

// Set the GitVersion via -ldflags="-X 'go.rtnl.ai/quarterdeck/pkg.GitVersion=$(git rev-parse --short HEAD)'"
var GitVersion string

// Set the BuildDate via -ldflags="-X go.rtnl.ai/quarterdeck/pkg.BuildDate=YYYY-MM-DD"
var BuildDate string

// Version returns the semantic version of the current build.
func Version(short bool) string {
	vers := semver.Version{
		Major:      VersionMajor,
		Minor:      VersionMinor,
		Patch:      VersionPatch,
		PreRelease: PreRelease(),
		BuildMeta:  BuildMeta(),
	}

	if short {
		return vers.Short()
	}
	return vers.String()
}

func PreRelease() string {
	if VersionReleaseLevel != "" && VersionReleaseLevel != "final" {
		if VersionReleaseNumber > 0 {
			return fmt.Sprintf("%s.%d", VersionReleaseLevel, VersionReleaseNumber)
		}
		return VersionReleaseLevel
	}
	return ""
}

func ParseBuildDate() *time.Time {
	if BuildDate != "" {
		t, err := time.Parse("2006-01-02", BuildDate)
		if err != nil {
			return nil
		}
		return &t
	}
	return nil
}

func BuildMeta() string {
	parts := make([]string, 0, 2)

	if GitVersion != "" {
		parts = append(parts, GitVersion)
	}

	if bd := ParseBuildDate(); bd != nil {
		parts = append(parts, bd.Format("20060102"))
	}

	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		return strings.Join(parts, ".")
	}
}
