package tools

import "embed"

const BundleVersion = "0.1.0-dev"

// SkillFiles contains complete, versioned skill directories shipped with zt.
//
//go:embed skills
var SkillFiles embed.FS
