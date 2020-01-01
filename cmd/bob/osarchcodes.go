package main

// "linux-amd64" (JSON keys in OsArchesSpec)
type OsArchCode string

// TODO: a test that guarantees that this is in sync with OsArchesSpec?
func osArchCodeToOsArchesSpec(code OsArchCode) OsArchesSpec {
	switch code {
	case "neutral":
		return OsArchesSpec{Neutral: true}
	case "linux-neutral":
		return OsArchesSpec{LinuxNeutral: true}
	case "linux-amd64":
		return OsArchesSpec{LinuxAmd64: true}
	case "linux-arm":
		return OsArchesSpec{LinuxArm: true}
	case "linux-arm64":
		return OsArchesSpec{LinuxArm64: true}
	case "windows-neutral":
		return OsArchesSpec{WindowsNeutral: true}
	case "windows-amd64":
		return OsArchesSpec{WindowsAmd64: true}
	case "darwin-amd64":
		return OsArchesSpec{DarwinAmd64: true}
	default:
		return OsArchesSpec{}
	}
}

// TODO: a test that guarantees that this is in sync with OsArchesSpec?
func osArchesIntersects(a OsArchesSpec, b OsArchesSpec) bool {
	return a.Neutral && b.Neutral ||
		a.LinuxNeutral && b.LinuxNeutral ||
		a.LinuxAmd64 && b.LinuxAmd64 ||
		a.LinuxArm && b.LinuxArm ||
		a.LinuxArm64 && b.LinuxArm64 ||
		a.WindowsNeutral && b.WindowsNeutral ||
		a.WindowsAmd64 && b.WindowsAmd64 ||
		a.DarwinAmd64 && b.DarwinAmd64
}
