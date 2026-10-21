package resolver

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kryptamine/herdr-auto-title/internal/herdr/herdrtest"
	"github.com/kryptamine/herdr-auto-title/internal/state"
)

func tabWithPane(pane *state.PaneState) state.TabState {
	pane.ID = "wE:p1"
	_ = pane
	pane.Focused = true

	return tabOf([]*state.PaneState{pane})
}

func TestTerminalTitleBeatsTheWorkingDirectory(t *testing.T) {
	t.Parallel()

	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		Dir:           dashboard,
		TerminalTitle: "Fix OAuth redirect",
	}))

	if got.Name != "dashboard › Fix OAuth redirect" {
		t.Errorf("name = %q, want %q", got.Name, "dashboard › Fix OAuth redirect")
	}

	if got.Reason != "terminal_title" {
		t.Errorf("reason = %q, want terminal_title", got.Reason)
	}

	if got.Confidence != ConfidenceTerminalTitle {
		t.Errorf("confidence = %d, want %d", got.Confidence, ConfidenceTerminalTitle)
	}
}

func TestGenericTerminalTitleFallsThrough(t *testing.T) {
	t.Parallel()

	// Every one of these was observed on a live Herdr session.
	for _, title := range []string{"zsh", "Claude Code", "node", "~", "~/W/dashboard", ""} {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			got := defaultChain().Resolve(tabWithPane(&state.PaneState{
				Dir:           dashboard,
				TerminalTitle: title,
			}))

			if got.Name != "dashboard" {
				t.Errorf("name = %q, want %q", got.Name, "dashboard")
			}

			if got.Reason != "cwd" {
				t.Errorf("reason = %q, want cwd", got.Reason)
			}
		})
	}
}

func TestTerminalTitleFallsBackToTheRawField(t *testing.T) {
	t.Parallel()

	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		Dir:              dashboard,
		TerminalTitleRaw: "\x1b[32m✳ Fix OAuth redirect\x1b[0m",
	}))

	// The raw field carries escapes Herdr would normally have stripped.
	if got.Name != "dashboard › ✳ Fix OAuth redirect" {
		t.Errorf("name = %q, want %q", got.Name, "dashboard › ✳ Fix OAuth redirect")
	}
}

func TestStrippedTerminalTitleWinsOverTheRawOne(t *testing.T) {
	t.Parallel()

	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		Dir:              dashboard,
		TerminalTitle:    "Fix OAuth redirect",
		TerminalTitleRaw: "◐ Fix OAuth redirect",
	}))

	if got.Name != "dashboard › Fix OAuth redirect" {
		t.Errorf("name = %q, want %q", got.Name, "dashboard › Fix OAuth redirect")
	}
}

func TestAnAntigravityLaunchCommandFallsThroughToItsSessionTopic(t *testing.T) {
	t.Parallel()

	for _, title := range []string{
		"agy --sandbox --dang ~/work/dashboard",
		"agy --sandbox",
		"AGY\t--sandbox",
		"agy.exe --sandbox",
		filepath.Join(herdrtest.Root(), "bin", "agy") + " --sandbox",
		"\x1b[32magy --sandbox\x1b[0m",
	} {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			pane := &state.PaneState{
				Dir:              dashboard,
				Agent:            "agy",
				TerminalTitleRaw: title,
				AgentTopic:       "Fix OAuth redirect",
			}

			got := defaultChain().Resolve(tabWithPane(pane))
			if want := "dashboard › agy › Fix OAuth redirect"; got.Name != want {
				t.Errorf("name = %q, want %q", got.Name, want)
			}

			if got.Reason != "transcript" {
				t.Errorf("reason = %q, want transcript", got.Reason)
			}

			if got := topics().Topic(pane); got != "agy › Fix OAuth redirect" {
				t.Errorf("topic = %q, want the session topic", got)
			}
		})
	}
}

func TestAnAntigravityLaunchCommandWithoutASessionTopicAddsNoActivity(t *testing.T) {
	t.Parallel()

	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		Dir:           dashboard,
		Agent:         "agy",
		TerminalTitle: "agy --sandbox --dang ~/work/dashboard",
	}))
	if want := "dashboard › agy"; got.Name != want {
		t.Errorf("name = %q, want %q", got.Name, want)
	}
}

func TestAnAntigravityCommandCheckKeepsMeaningfulTitlesAndOtherAgents(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		agent string
		title string
		want  string
	}{
		{"agy", "Fix OAuth redirect", "dashboard › agy › Fix OAuth redirect"},
		{"agy", "agy integration design", "dashboard › agy › integration design"},
		{"agy", "agy - integration design", "dashboard › agy › integration design"},
		{"claude", "agy --sandbox", "dashboard › claude › agy --sandbox"},
		{"", "agy --sandbox", "dashboard › agy --sandbox"},
	} {
		t.Run(test.agent+"/"+test.title, func(t *testing.T) {
			t.Parallel()

			got := defaultChain().Resolve(tabWithPane(&state.PaneState{
				Dir:           dashboard,
				Agent:         test.agent,
				TerminalTitle: test.title,
				AgentTopic:    "Earlier session topic",
			}))
			if got.Name != test.want {
				t.Errorf("name = %q, want %q", got.Name, test.want)
			}

			if got.Reason != "terminal_title" {
				t.Errorf("reason = %q, want terminal_title", got.Reason)
			}
		})
	}
}

func TestTerminalTitleIsSanitized(t *testing.T) {
	t.Parallel()

	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		Dir:           dashboard,
		TerminalTitle: "\x1b[31mFix OAuth\nredirect\x1b[0m\t",
	}))

	if got.Name != "dashboard › Fix OAuth redirect" {
		t.Errorf("name = %q, want %q", got.Name, "dashboard › Fix OAuth redirect")
	}
}

func TestLongTerminalTitleIsTruncatedAsAWhole(t *testing.T) {
	t.Parallel()

	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		Dir:           dashboard,
		TerminalTitle: strings.Repeat("long ", 40),
	}))

	if runes := len([]rune(got.Name)); runes > DefaultMaxLength {
		t.Errorf("name is %d runes, want at most %d: %q", runes, DefaultMaxLength, got.Name)
	}

	if !strings.HasPrefix(got.Name, "dashboard › long") {
		t.Errorf("name = %q, want it to start with the context", got.Name)
	}
}

func TestTerminalTitleWithoutAWorkingDirectory(t *testing.T) {
	t.Parallel()

	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		TerminalTitle: "Fix OAuth redirect",
	}))

	// No context to pair it with, so no dangling separator.
	if got.Name != "Fix OAuth redirect" {
		t.Errorf("name = %q, want %q", got.Name, "Fix OAuth redirect")
	}
}

func TestTerminalTitleRepeatingTheContextIsDropped(t *testing.T) {
	t.Parallel()

	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		Dir:           dashboard,
		TerminalTitle: "Dashboard",
	}))

	if got.Name != "dashboard" {
		t.Errorf("name = %q, want %q", got.Name, "dashboard")
	}
}

func TestTerminalTitleWithNothingElse(t *testing.T) {
	t.Parallel()

	got := defaultChain().Resolve(tabWithPane(&state.PaneState{
		TerminalTitle: "zsh",
	}))

	if got.Name != GenericFallback {
		t.Errorf("name = %q, want %q", got.Name, GenericFallback)
	}
}

func TestEditorTitleKeepsTheFileAndDropsThePath(t *testing.T) {
	t.Parallel()

	// All three were observed on a live session.
	tests := []struct {
		title string
		want  string
	}{
		{
			"Makefile (~/Work/herdr-auto-title) - Nvim",
			"herdr-auto-title › Makefile - Nvim",
		},
		{
			"03-terminal-title-source.md (~/Work/herdr-auto-title/docs/issues) - Nvim",
			"herdr-auto-title › 03-terminal-title-source.md - Nvim",
		},
		{
			// A file browser buffer names no file, so only the editor is left.
			"- (oil:///Users/dev/Work/herdr-auto-title) - Nvim",
			"herdr-auto-title › Nvim",
		},
	}

	// The cap is not what this tests, and the longest of the three outgrows it.
	const wide = 80

	for _, tc := range tests {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			got := Default(
				Options{MaxLength: wide, BranchMax: DefaultBranchMaxLength},
			).Resolve(tabWithPane(&state.PaneState{
				Dir:           herdrtest.Dir("Work", "herdr-auto-title"),
				TerminalTitle: tc.title,
			}))

			if got.Name != tc.want {
				t.Errorf("name = %q, want %q", got.Name, tc.want)
			}
		})
	}
}
