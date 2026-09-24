package tools

import (
	"embed"
	"runtime/debug"
)

// releaseVersion is set for release archives. go install @version supplies the
// module version through build info instead, so both distribution paths agree.
var releaseVersion = ""

func Version() string {
	if releaseVersion != "" {
		return releaseVersion
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// SkillFiles contains complete, versioned skill directories shipped with zt.
//
//go:embed skills
var SkillFiles embed.FS
