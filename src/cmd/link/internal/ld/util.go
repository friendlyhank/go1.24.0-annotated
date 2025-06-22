package ld

import "os"

// Exit exits with code after executing all atExitFuncs.
func Exit(code int) {
	os.Exit(code)
}
