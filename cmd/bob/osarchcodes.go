package main

import (
	"github.com/function61/turbobob/pkg/bobfile"
)

// looks like "linux-amd64" (JSON keys in OsArchesSpec). need to add to
// - OsArchesSpec
// - AsBuildEnvVariables()
// - osArchCodeToOsArchesSpec()
// - osArchesIntersects()
type OsArchCode string

// TODO: a test that guarantees that this is in sync with OsArchesSpec?
func osArchCodeToOsArchesSpec(code OsArchCode) bobfile.OsArchesSpec {
	switch code {
	case "neutral":
		return bobfile.OsArchesSpec{Neutral: true}
	case "linux-neutral":
		return bobfile.OsArchesSpec{LinuxNeutral: true}
	case "linux-amd64":
		return bobfile.OsArchesSpec{LinuxAmd64: true}
	case "linux-arm":
		return bobfile.OsArchesSpec{LinuxArm: true}
	case "linux-arm64":
		return bobfile.OsArchesSpec{LinuxArm64: true}
	case "windows-neutral":
		return bobfile.OsArchesSpec{WindowsNeutral: true}
	case "windows-amd64":
		return bobfile.OsArchesSpec{WindowsAmd64: true}
	case "darwin-amd64":
		return bobfile.OsArchesSpec{DarwinAmd64: true}
	default:
		return bobfile.OsArchesSpec{}
	}
}

// TODO: a test that guarantees that this is in sync with OsArchesSpec?
func osArchesIntersects(a bobfile.OsArchesSpec, b bobfile.OsArchesSpec) bool {
	return a.Neutral && b.Neutral ||
		a.LinuxNeutral && b.LinuxNeutral ||
		a.LinuxAmd64 && b.LinuxAmd64 ||
		a.LinuxArm && b.LinuxArm ||
		a.LinuxArm64 && b.LinuxArm64 ||
		a.WindowsNeutral && b.WindowsNeutral ||
		a.WindowsAmd64 && b.WindowsAmd64 ||
		a.DarwinAmd64 && b.DarwinAmd64
}
