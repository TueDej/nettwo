//go:build !windows

package ui

// EnsureConsole is a no-op outside Windows: Unix launches always inherit a
// terminal from the calling shell.
func EnsureConsole() {}

// PauseBeforeExit is a no-op outside Windows: parent terminals persist on
// their own.
func PauseBeforeExit() {}

// LogStartupFailure is a no-op outside Windows; errors go to stderr.
func LogStartupFailure(err error) string { return "" }
