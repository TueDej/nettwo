package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

var (
	mu          sync.Mutex
	spinnerMsg  string
	spinnerRun  bool
	spinnerStop chan struct{}
	quietMode   bool
	verboseMode bool
	silenced    bool // When true, ALL output is suppressed (used by TUI)
)

// SetSilenced toggles silent mode that suppresses all terminal output.
// Used by the TUI to prevent legacy ANSI output from corrupting alt-screen.
func SetSilenced(s bool) {
	mu.Lock()
	silenced = s
	mu.Unlock()
}

// Init configures the UI output modes. It is called from the root command's
// PersistentPreRunE.
func Init(quiet, verbose bool) {
	quietMode = quiet
	verboseMode = verbose
}

func Success(msg string) {
	if quietMode || silenced {
		return
	}
	fmt.Fprintln(os.Stderr, "\033[32m✓\033[0m "+msg)
}

func Error(msg string) {
	if silenced {
		return
	}
	fmt.Fprintln(os.Stderr, "\033[31m✗\033[0m "+msg)
}

func Warn(msg string) {
	if quietMode || silenced {
		return
	}
	fmt.Fprintln(os.Stderr, "\033[33m⚠\033[0m "+msg)
}

func Info(msg string) {
	if quietMode || silenced {
		return
	}
	fmt.Fprintln(os.Stderr, "\033[34mℹ\033[0m "+msg)
}

func Debug(msg string) {
	if !verboseMode || silenced {
		return
	}
	fmt.Fprintln(os.Stderr, "\033[90m»\033[0m "+msg)
}

func StartSpinner(msg string) {
	if quietMode || silenced {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	// If a spinner is already running, stop it first so we don't leak goroutines.
	if spinnerRun && spinnerStop != nil {
		spinnerRun = false
		close(spinnerStop)
		spinnerStop = nil
	}

	spinnerMsg = msg
	stop := make(chan struct{})
	spinnerStop = stop
	spinnerRun = true

	go func(stop chan struct{}, msg string) {
		spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		fmt.Fprint(os.Stderr, "\r")
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				mu.Lock()
				if !spinnerRun || spinnerStop != stop {
					mu.Unlock()
					return
				}
				fmt.Fprintf(os.Stderr, "\r\033[36m%s\033[0m %s", spinners[i%len(spinners)], msg)
				i++
				mu.Unlock()
			}
		}
	}(stop, msg)
}

func StopSpinner() {
	mu.Lock()
	defer mu.Unlock()

	if spinnerStop != nil {
		spinnerRun = false
		close(spinnerStop)
		spinnerStop = nil
	}
	if silenced {
		return
	}

	// Clear the spinner line immediately
	fmt.Fprint(os.Stderr, "\r\033[K")
}

func Println(args ...interface{}) {
	if quietMode || silenced {
		return
	}
	fmt.Fprintln(os.Stderr, args...)
}

func Printf(format string, args ...interface{}) {
	if quietMode || silenced {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func Prompt(prompt string) string {
	if quietMode {
		return ""
	}
	fmt.Fprint(os.Stderr, prompt)
	var input string
	fmt.Fscanln(os.Stdin, &input)
	return strings.TrimSpace(input)
}

func PromptPassword(prompt string) string {
	if quietMode {
		return ""
	}
	fmt.Fprint(os.Stderr, prompt)

	// If stdin is a terminal, disable echo so the password is masked.
	if term.IsTerminal(int(os.Stdin.Fd())) {
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // move to next line after the hidden input
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(bytePassword))
	}

	// Non-interactive (piped) stdin: read a line without echo tricks.
	bytePassword, _ := readLine(os.Stdin)
	fmt.Fprintln(os.Stderr)
	return strings.TrimSpace(string(bytePassword))
}

// readLine reads stdin until a newline. Used when stdin is not a TTY
// (e.g. piped input), where term.ReadPassword would error.
func readLine(fd *os.File) ([]byte, error) {
	var line []byte
	for {
		var buf [1]byte
		n, err := fd.Read(buf[:])
		if err != nil || n == 0 {
			break
		}
		if buf[0] == '\n' || buf[0] == '\r' {
			break
		}
		line = append(line, buf[0])
	}
	return line, nil
}

func IsQuiet() bool {
	return quietMode
}

func Confirm(prompt string) (bool, error) {
	if quietMode {
		return false, nil
	}
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	var input string
	fmt.Fscanln(os.Stdin, &input)
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes", nil
}

func SelectAccount(accounts []string, volumes map[string]string) (string, error) {
	if len(accounts) == 0 {
		return "", fmt.Errorf("no accounts available")
	}
	if len(accounts) == 1 {
		return accounts[0], nil
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Available accounts:")
	fmt.Fprintln(os.Stderr, "")
	for i, acc := range accounts {
		vol := "unknown"
		if v, ok := volumes[acc]; ok && v != "" {
			vol = v
		}
		fmt.Fprintf(os.Stderr, "  [%d] %-25s %s\n", i+1, acc, vol)
	}
	fmt.Fprintf(os.Stderr, "\n  [%d] Add new account\n", len(accounts)+1)
	fmt.Fprintln(os.Stderr, "")

	var input string
	for {
		fmt.Fprint(os.Stderr, "Select account: ")
		os.Stderr.Sync()
		fmt.Scanln(&input)
		input = strings.TrimSpace(input)
		if input != "" {
			break
		}
	}

	var idx int
	_, err := fmt.Sscanf(input, "%d", &idx)
	if err != nil || idx < 1 || idx > len(accounts)+1 {
		return "", fmt.Errorf("invalid choice")
	}

	if idx == len(accounts)+1 {
		return "", nil
	}

	return accounts[idx-1], nil
}
