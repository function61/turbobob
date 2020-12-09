package powerline

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/function61/turbobob/pkg/ansicolor"
	"github.com/spf13/cobra"
)

type Segment struct {
	Label  string
	Colors ColorPair
}

func NewSegment(label string, colors ColorPair) Segment {
	return Segment{label, colors}
}

func Entrypoint() *cobra.Command {
	return &cobra.Command{
		Use:    "powerline [rc]",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			rc, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				panic(err)
			}

			whiteOnRed := ColorPair{ansicolor.BrightWhite, ansicolor.Red}
			blackOnWhite := ColorPair{ansicolor.BrightBlack, ansicolor.White}
			blackOnBlue := ColorPair{ansicolor.Black, ansicolor.Blue}

			wd, err := os.Getwd()
			if err != nil {
				panic(err)
			}

			segments := []Segment{
				NewSegment("/", blackOnBlue),
			}

			for _, node := range tree(wd) {
				segments = append(segments, NewSegment(node, blackOnWhite))
			}

			dollar := func() Segment {
				if rc != 0 {
					return NewSegment("$", whiteOnRed)
				} else {
					return NewSegment("$", blackOnWhite)
				}
			}()

			segments = append(segments, dollar)
			// line:=Generate("…", "workspace")
			line := Generate(segments...)
			fmt.Println(line)
			// fmt.Println(ShellEscape(line))
		},
	}
}

func Generate(segments ...Segment) string {
	k := &powerlineContext{&BashShell{}}
	var buf bytes.Buffer

	write := func(x string) {
		buf.WriteString(x)
	}

	// TODO: use change detectors, so we don't have to repeatedly set same color
	fgColor := func(color Color) {
		write(k.shell.ForegroundEx(color))
	}
	bgColor := func(color Color) {
		write(k.shell.BackgroundEx(color))
	}

	lastIdx := len(segments) - 1
	for idx, seg := range segments {
		next := func() *Segment {
			if idx < lastIdx {
				return &segments[idx+1]
			} else {
				return nil
			}
		}()

		bgColor(seg.Colors.Background)
		fgColor(seg.Colors.Text)

		write(space(seg.Label))

		nextSegBg := func() Color {
			if idx == lastIdx {
				return ansicolor.Transparent
			} else {
				return next.Colors.Background
			}
		}()

		// transitions to differnt bg color need thick separator
		thickSeparator := seg.Colors.Background != nextSegBg

		bgColor(nextSegBg)

		if thickSeparator {
			// thick uses:
			// - fg color to match what was to left of it
			// - bg color to match what is to right of it (already set above)
			fgColor(seg.Colors.Background)
			write(fo.Separator)
		} else {
			write(fo.SeparatorThin)
		}
	}

	write(" ")

	write(k.shell.ResetColor())

	return buf.String()
}

type powerlineContext struct {
	shell Shell
}

func space(str string) string {
	return " " + str + " "
}

func ShellEscape(line string) string {
	return strings.ReplaceAll(line, "\x1b", `\e`)
	// return `"` + strings.ReplaceAll(line, "\x1b", `\e`) + `"`
}

var (
	fo = struct {
		Lock                 string
		Network              string
		NetworkAlternate     string
		Separator            string
		SeparatorThin        string
		SeparatorReverse     string
		SeparatorReverseThin string
	}{
		Lock:                 "\uE0A2",
		Network:              "\u260E",
		NetworkAlternate:     "\uE0A2",
		Separator:            "\uE0B0",
		SeparatorThin:        "\uE0B1",
		SeparatorReverse:     "\uE0B2",
		SeparatorReverseThin: "\uE0B3",
	}
)

// "/home/joonas/stash" => ["home","joonas","stash"]
func tree(path string) []string {
	comps := []string{}
	for {
		base := filepath.Base(path)
		path = filepath.Dir(path)
		comps = append([]string{base}, comps...)
		if path == "." || path == "/" {
			break
		}
	}
	return comps
}
