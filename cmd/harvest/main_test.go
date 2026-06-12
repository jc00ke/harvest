package main

import "testing"

func TestWantsUI(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: []string{}, want: false},
		{name: "-ui flag", args: []string{"-ui"}, want: true},
		{name: "--ui flag", args: []string{"--ui"}, want: true},
		{name: "subcommand", args: []string{"entries", "list"}, want: false},
		{name: "-ui not first", args: []string{"entries", "-ui"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := wantsUI(tt.args), tt.want; got != want {
				t.Errorf("wantsUI(%v)=%t, want=%t", tt.args, got, want)
			}
		})
	}
}

func TestWantsDemo(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: []string{}, want: false},
		{name: "-ui only", args: []string{"-ui"}, want: false},
		{name: "-ui --demo", args: []string{"-ui", "--demo"}, want: true},
		{name: "-ui -demo", args: []string{"-ui", "-demo"}, want: true},
		{name: "--ui --demo", args: []string{"--ui", "--demo"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := wantsDemo(tt.args), tt.want; got != want {
				t.Errorf("wantsDemo(%v)=%t, want=%t", tt.args, got, want)
			}
		})
	}
}
