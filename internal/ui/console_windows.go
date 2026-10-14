//go:build windows

package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleWindow      = kernel32.NewProc("GetConsoleWindow")
	procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
)

// relaunchedEnv marks a child that was already placed in a fresh console,
// so a still-consoless child never relauches itself in a loop.
const relaunchedEnv = "NETTWO_NEW_CONSOLE"

func hasConsole() bool {
	ret, _, _ := procGetConsoleWindow.Call()
	return ret != 0
}

// ownsConsole reports whether this process is the only one attached to its
// console. That is the typical Explorer double-click case: the window dies
// with us, so errors must be kept on screen instead of flashing by.
func ownsConsole() bool {
	var list [4]uint32
	ret, _, _ := procGetConsoleProcessList.Call(
		uintptr(unsafe.Pointer(&list[0])),
		uintptr(len(list)),
	)
	return ret == 1
}

// EnsureConsole guarantees a usable console on Windows. When the program is
// started without one (Task Scheduler, GUI launchers, …), it re-opens itself
// in a fresh terminal window and exits the consoless parent. When double-
// clicked from Explorer, Windows already provides a console, so this is a
// no-op and the TUI runs in that window.
func EnsureConsole() {
	if hasConsole() {
		return
	}
	if os.Getenv(relaunchedEnv) != "" {
		return // already relaunched; surface the real error instead of looping
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	args := []string{"/c", "start", "", exe}
	for _, a := range os.Args[1:] {
		args = append(args, quoteArg(a))
	}
	cmd := exec.Command("cmd.exe", args...)
	cmd.Env = append(os.Environ(), relaunchedEnv+"=1")
	// Hide the transient launcher; `start` opens the visible console.
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return
	}
	os.Exit(0)
}

// PauseBeforeExit keeps an Explorer-owned console window open after a fatal
// error so the message can actually be read. It is a no-op everywhere else
// (a parent terminal stays open on its own).
func PauseBeforeExit() {
	if !ownsConsole() {
		return
	}
	fmt.Fprintln(os.Stderr, "Press Enter to exit...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func quoteArg(a string) string {
	if a == "" {
		return `""`
	}
	if !strings.ContainsAny(a, " \t\"") {
		return a
	}
	return `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
}
