//go:build !windows

package ui

// EnsureConsole is a no-op outside Windows: Unix launches always inherit
// (or can be checked for) a terminal by the caller.
func EnsureConsole() {}

// PauseBeforeExit is a no-op outside Windows: parent terminals persist.
func PauseBeforeExit() {}
