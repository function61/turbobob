package termutil

import (
	"fmt"
)

// returns `unsetTerminalTitle` func which can be used to unset the title
func SetTitle(title string) func() {
	fmt.Printf("\x1b]2;%s\a", title)
	return func() {
		fmt.Print("\x1b]2;-\a")
	}
}
