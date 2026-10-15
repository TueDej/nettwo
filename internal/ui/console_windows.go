//go:build windows

package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Windows launch bootstrap.
//
// Double-clicking an .exe in Explorer, launching it from a shortcut, a GUI
// launcher or Task Scheduler gives the process a console (or worse, *some*
// console) that TUI programs cannot use: bubbletea then fails to open a TTY and
// all the user sees is a window that flashes an error and disappears.
//
// Instead of guessing, we detect the situation up front and re-open ourselves
// in a brand new console window, so the program always runs somewhere it can
// draw a TUI. Everything we learn about the launch is appended to a small log
// file so failures outside our control can still be diagnosed.

const (
	th32csSnapProcess  = 0x00000002
	fileTypeChar       = 0x0002
	invalidHandleValue = ^uintptr(0)
	createNewConsole   = 0x00000010

	// relaunchedEnv marks a child that already got its own console, so a
	// launch loop is impossible.
	relaunchedEnv = "NETTWO_NEW_CONSOLE"

	// consoleTitle names the window we open for ourselves.
	consoleTitle = "nettwo"

	logFileName = "launch.log"
	maxLogSize  = 512 * 1024
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

var (
	procGetConsoleWindow         = kernel32.NewProc("GetConsoleWindow")
	procGetConsoleProcessList    = kernel32.NewProc("GetConsoleProcessList")
	procGetCurrentProcessId      = kernel32.NewProc("GetCurrentProcessId")
	procGetFileType              = kernel32.NewProc("GetFileType")
	procGetConsoleMode           = kernel32.NewProc("GetConsoleMode")
	procSetConsoleTitleW         = kernel32.NewProc("SetConsoleTitleW")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = kernel32.NewProc("Process32FirstW")
	procProcess32NextW           = kernel32.NewProc("Process32NextW")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
)

// processEntry32 mirrors Windows' PROCESSENTRY32W.
type processEntry32 struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr // padding before this matches the Windows ABI
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [260]uint16
}

var (
	logMu       sync.Mutex
	sizeChecked bool
)

func appendLog(format string, args ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()

	path := logFilePath()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	if !sizeChecked {
		sizeChecked = true
		if info, err := os.Stat(path); err == nil && info.Size() > maxLogSize {
			_ = os.Rename(path, path+".old")
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

// EnsureConsole guarantees the program runs somewhere it can draw a TUI.
//
// It re-opens the process in a fresh console window when the current one is
// missing or unusable, and also when the program was double-clicked from
// Explorer. Every other launch (cmd, PowerShell, Windows Terminal, WSL, CI,
// pipes) is left alone. The decision itself lives in launchState so it can be
// reviewed and tested without a Windows box.
func EnsureConsole() {
	state := launchState{
		ArgCount:          len(os.Args) - 1,
		AlreadyRelaunched: os.Getenv(relaunchedEnv) != "",
	}

	if !state.AlreadyRelaunched {
		ret, _, _ := procGetConsoleWindow.Call()
		state.ConsoleWindowPresent = ret != 0
		state.StdinIsConsole = isConsoleHandle(os.Stdin)
		state.StdoutIsConsole = isConsoleHandle(os.Stdout)
		state.ParentIsExplorer = parentIsExplorer()
	}

	exe, _ := os.Executable()
	appendLog(
		"launch pid=%d exe=%s args=%q console_procs=%d %s",
		os.Getpid(), exe, os.Args[1:], consoleProcessCount(), state.describe(),
	)

	if state.AlreadyRelaunched {
		// We are the copy that owns a console: make the window identifiable.
		title, err := syscall.UTF16PtrFromString(consoleTitle)
		if err == nil {
			procSetConsoleTitleW.Call(uintptr(unsafe.Pointer(title)))
		}
		return
	}

	if !state.needsOwnConsole() {
		return
	}

	if relaunchInNewConsole(os.Args[1:]) {
		// The new window takes over; this one is redundant.
		os.Exit(0)
	}
	appendLog("relaunch failed; continuing in the current console")
}

// describe renders a launchState for the log file.
func (s launchState) describe() string {
	return fmt.Sprintf(
		"console_window=%t stdin_console=%t stdout_console=%t parent_explorer=%t args=%d relaunched=%t",
		s.ConsoleWindowPresent, s.StdinIsConsole, s.StdoutIsConsole,
		s.ParentIsExplorer, s.ArgCount, s.AlreadyRelaunched,
	)
}

func isConsoleHandle(f *os.File) bool {
	if f == nil {
		return false
	}
	h := syscall.Handle(f.Fd())
	ft, _, _ := procGetFileType.Call(uintptr(h))
	if ft != fileTypeChar {
		return false
	}
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	return r != 0
}

// parentIsExplorer reports whether the process was started by Explorer, i.e.
// the classic double-click / "Open" from File Explorer.
func parentIsExplorer() bool {
	parent := parentProcessName()
	if parent == "" {
		return false // could not tell; prefer not to change behavior
	}
	return strings.Contains(strings.ToLower(parent), "explorer")
}

// parentProcessName returns the image name of our parent process, or "" when
// it cannot be determined (the parent already exited, or the API failed).
func parentProcessName() string {
	snap, _, _ := procCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if snap == invalidHandleValue || snap == 0 {
		return ""
	}
	defer procCloseHandle.Call(snap)

	self, _, _ := procGetCurrentProcessId.Call()

	var entry processEntry32
	entry.dwSize = uint32(unsafe.Sizeof(entry))

	parentID := uint32(0)
	names := map[uint32]string{}

	if ret, _, _ := procProcess32FirstW.Call(snap, uintptr(unsafe.Pointer(&entry))); ret == 0 {
		return ""
	}
	for {
		name := syscall.UTF16ToString(entry.szExeFile[:])
		names[entry.th32ProcessID] = name
		if entry.th32ProcessID == uint32(self) {
			parentID = entry.th32ParentProcessID
		}
		ret, _, _ := procProcess32NextW.Call(snap, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}
	return names[parentID]
}

// consoleProcessCount returns how many processes share our console. A count of
// 1 means we own it: nothing else will keep the window alive after we exit.
func consoleProcessCount() int {
	var list [8]uint32
	ret, _, _ := procGetConsoleProcessList.Call(
		uintptr(unsafe.Pointer(&list[0])),
		uintptr(len(list)),
	)
	if ret == 0 {
		return 0
	}
	return int(ret)
}

// relaunchInNewConsole starts a copy of this process in a brand new console
// window and reports whether that succeeded.
func relaunchInNewConsole(args []string) bool {
	exe, err := os.Executable()
	if err != nil {
		appendLog("os.Executable failed: %v", err)
		return false
	}

	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), relaunchedEnv+"=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewConsole,
	}
	if err := cmd.Start(); err != nil {
		appendLog("relaunch failed: %v", err)
		return false
	}
	appendLog("relaunched %s in a new console (pid %d)", exe, cmd.Process.Pid)
	return true
}

// PauseBeforeExit keeps an Explorer-owned console window open after a fatal
// error so the message can actually be read. Elsewhere (a parent terminal,
// pipes, CI) it is a no-op.
func PauseBeforeExit() {
	if consoleProcessCount() != 1 {
		return // somebody else keeps the window alive
	}
	fmt.Fprintln(os.Stderr, "Press Enter to exit...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

// LogStartupFailure records a fatal startup error and returns the log path, or
// "" when no log could be written.
func LogStartupFailure(err error) string {
	appendLog("FATAL: %v", err)
	return logFilePath()
}

func logFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "nettwo", logFileName)
}
