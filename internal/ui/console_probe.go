package ui

// launchState captures everything we can observe about how this process was
// started. It is deliberately data-only so the launch decision can be reasoned
// about (and tested) on any platform: the Windows bootstrap fills it in from
// the real console, nobody else ever calls it.
type launchState struct {
	// ConsoleWindowPresent is true when a console window is attached.
	ConsoleWindowPresent bool
	// StdinIsConsole/StdoutIsConsole are true when the corresponding stream is
	// a real console handle rather than a pipe, file or NUL.
	StdinIsConsole  bool
	StdoutIsConsole bool

	// ParentIsExplorer is true when the parent process is Explorer, i.e. the
	// program was double-clicked or "opened" from File Explorer.
	ParentIsExplorer bool

	// ArgCount is the number of command line arguments we were given.
	ArgCount int

	// AlreadyRelaunched is true when we are the copy of ourselves that was
	// started in its own console; such a process must never relaunch again.
	AlreadyRelaunched bool
}

// needsOwnConsole reports whether the program has to be re-opened in its own
// console window to be usable.
//
// Three tiers, in order:
//
//  1. A relaunched child never relaunches again — that is what makes the
//     bootstrap loop-free.
//  2. The current console cannot host a full-screen TUI: no console window at
//     all, or stdin is not a console (Task Scheduler, GUI launchers, detached
//     or detached-looking launches). Running a TUI there fails on TTY setup,
//     which is exactly the "opens a window and complains" report. This is
//     repaired regardless of arguments, because there is no terminal to honour
//     them in.
//  3. We were double-clicked from Explorer. That console is created by the
//     shell for a GUI launch, is sized by the legacy defaults and dies with
//     us, so a window of its own is both more reliable and what a user expects
//     from a double-click.
//
// Normal terminal launches (cmd, PowerShell, Windows Terminal, WSL, CI,
// pipes, explicit arguments) are left untouched. Redirecting stdout is a
// deliberate choice and does not trigger a new window on its own.
func (s launchState) needsOwnConsole() bool {
	if s.AlreadyRelaunched {
		return false
	}
	if !s.ConsoleWindowPresent || !s.StdinIsConsole {
		return true
	}
	if s.ArgCount > 0 {
		return false
	}
	return s.ParentIsExplorer
}
