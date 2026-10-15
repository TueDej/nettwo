package ui

import "testing"

func TestLaunchStateNeedsOwnConsole(t *testing.T) {
	tests := []struct {
		name  string
		state launchState
		want  bool
	}{
		{
			name: "cmd/powershell launch stays put",
			state: launchState{
				ConsoleWindowPresent: true,
				StdinIsConsole:       true,
				StdoutIsConsole:      true,
			},
			want: false,
		},
		{
			name: "redirected stdout from a terminal stays put",
			state: launchState{
				ConsoleWindowPresent: true,
				StdinIsConsole:       true,
				StdoutIsConsole:      false,
			},
			// Output redirection is deliberate; a terminal is still driving us.
			want: false,
		},
		{
			name: "no console window gets its own even with arguments",
			state: launchState{
				ConsoleWindowPresent: false,
				StdinIsConsole:       false,
				StdoutIsConsole:      false,
				ArgCount:             2,
			},
			want: true,
		},
		{
			name: "explorer double-click gets its own window",
			state: launchState{
				ConsoleWindowPresent: true,
				StdinIsConsole:       true,
				StdoutIsConsole:      true,
				ParentIsExplorer:     true,
			},
			want: true,
		},
		{
			name: "explorer double-click with args stays put",
			state: launchState{
				ConsoleWindowPresent: true,
				StdinIsConsole:       true,
				StdoutIsConsole:      true,
				ParentIsExplorer:     true,
				ArgCount:             1,
			},
			want: false,
		},
		{
			name: "relaunched child never loops",
			state: launchState{
				ConsoleWindowPresent: true,
				StdinIsConsole:       true,
				StdoutIsConsole:      true,
				ParentIsExplorer:     true,
				AlreadyRelaunched:    true,
			},
			want: false,
		},
		{
			name: "relaunched child with broken console still never loops",
			state: launchState{
				AlreadyRelaunched: true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.needsOwnConsole(); got != tt.want {
				t.Fatalf("needsOwnConsole() = %v, want %v", got, tt.want)
			}
		})
	}
}
