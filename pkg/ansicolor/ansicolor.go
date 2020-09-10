// ANSI color constants
package ansicolor

var (
	Black   = &Color{"30", "40"}
	Red     = &Color{"31", "41"}
	Green   = &Color{"32", "42"}
	Yellow  = &Color{"33", "43"}
	Blue    = &Color{"34", "44"}
	Magenta = &Color{"35", "45"}
	Cyan    = &Color{"36", "46"}
	White   = &Color{"37", "47"}

	BrightBlack   = &Color{"90", "100"}
	BrightRed     = &Color{"91", "101"}
	BrightGreen   = &Color{"92", "102"}
	BrightYellow  = &Color{"93", "103"}
	BrightBlue    = &Color{"94", "104"}
	BrightMagenta = &Color{"95", "105"}
	BrightCyan    = &Color{"96", "106"}
	BrightWhite   = &Color{"97", "107"}
)

type Color struct {
	fgCode string
	bgCode string
}

func (a *Color) FgCode() string {
	return a.fgCode
}

func (a *Color) BgCode() string {
	return a.bgCode
}
