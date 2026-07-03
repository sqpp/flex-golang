package flex

import (
	"fmt"
	"strings"
)

var (
	Version    = "1.0.0"
	BuildTime  = "unknown"
	GitCommit  = "unknown"
	Author     = "unknown"
	ProjectURL = "unknown"
	BuildArch  = "unknown"
	BuildGoVer = "unknown"
)

// GetFullVersionInfo returns a formatted version string for the CLI
func GetFullVersionInfo() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("FLEX Decoder - v%s\n", Version))
	
	if BuildTime != "unknown" {
		sb.WriteString(fmt.Sprintf("Build Time: %s\n", BuildTime))
	}
	if GitCommit != "unknown" {
		sb.WriteString(fmt.Sprintf("Commit: %s\n", GitCommit))
	}
	if BuildArch != "unknown" {
		sb.WriteString(fmt.Sprintf("Arch: %s\n", BuildArch))
	}
	if BuildGoVer != "unknown" {
		sb.WriteString(fmt.Sprintf("Go Version: %s\n", BuildGoVer))
	}
	if Author != "unknown" {
		sb.WriteString(fmt.Sprintf("Author: %s\n", Author))
	}
	if ProjectURL != "unknown" {
		sb.WriteString(fmt.Sprintf("Project: %s\n", ProjectURL))
	}
	
	return strings.TrimSpace(sb.String())
}
