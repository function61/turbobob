package powerline

import (
	"fmt"
)

type ColorPair struct {
	Text       Color
	Background Color
}

type Color interface {
	FgCode() string
	BgCode() string
}

type Shell interface {
	Foreground(code string) string
	Background(code string) string
	ForegroundEx(color Color) string
	BackgroundEx(color Color) string

	ResetColor() string
}

type BashShell struct{}

// returns \e[38;5;<CODE>m
func (b *BashShell) Foreground(code string) string {
	return b.escaped(fmt.Sprintf("38;5;%s", code))
}

// returns \e[48;5;<CODE>m
func (b *BashShell) Background(code string) string {
	return b.escaped(fmt.Sprintf("48;5;%s", code))
}

func (b *BashShell) ForegroundEx(color Color) string {
	return b.escaped(color.FgCode())
}

func (b *BashShell) BackgroundEx(color Color) string {
	return b.escaped(color.BgCode())
}

func (b *BashShell) escaped(msg string) string {
	// https://unix.stackexchange.com/a/389095
	// https://wiki.archlinux.org/index.php/Bash/Prompt_customization#Embedding_commands
	// \x01 is "start of heading", to tell Bash this is non-printable char (otherwise it messes up prompt length calc)
	// \x02 is "start of text", which undoes SOH

	// \x1b is \e (escape)

	// return fmt.Sprintf("\x1b[%sm", msg)
	return fmt.Sprintf("\x01\x1b[%sm\x02", msg)
	// return fmt.Sprintf(`\e[%sm`, msg)
}

// returns \e[0m
func (b *BashShell) ResetColor() string {
	return b.escaped("0")
}
