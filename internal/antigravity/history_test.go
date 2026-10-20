package antigravity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kryptamine/herdr-auto-title/internal/transcript"
)

const otherSession = "135b224f-38c9-4c6e-9e23-56b955db5013"

func (h home) history(content string) string {
	h.t.Helper()

	path := filepath.Join(h.root, ".gemini", "antigravity-cli", "history.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		h.t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		h.t.Fatal(err)
	}

	return path
}

func appendHistory(t *testing.T, path, content string) {
	t.Helper()

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()

	if _, err := file.WriteString(content); err != nil {
		t.Fatal(err)
	}
}

func TestTheLatestPromptNamesOnlyItsOwnSession(t *testing.T) {
	t.Parallel()

	h := newHome(t)
	h.annotate(`title:"Opening topic"`)
	h.history(strings.Join([]string{
		`{"conversationId":"` + session + `","display":"First request"}`,
		`{"conversationId":"` + session + `","display":"Latest request"}`,
		`{"conversationId":"` + otherSession + `","display":"Another session"}`,
	}, "\n") + "\n")

	r := NewReader(h.root)
	if got := r.Title(session); got != "Latest request" {
		t.Errorf("title = %q, want latest prompt over annotation", got)
	}

	if got := r.Title(otherSession); got != "Another session" {
		t.Errorf("other title = %q, want its own prompt", got)
	}
}

func TestHistoryIgnoresRecordsThatAreNotSessionPrompts(t *testing.T) {
	t.Parallel()

	for name, line := range map[string]string{
		"malformed":       `{`,
		"missing id":      `{"display":"Unlinked request","workspace":"/same/directory"}`,
		"wrong id":        `{"conversationId":"not-a-session","display":"Unlinked request"}`,
		"blank":           `{"conversationId":"` + session + `","display":" \n\t"}`,
		"slash command":   `{"conversationId":"` + session + `","display":"Rename topic","type":"slash_command"}`,
		"untyped command": `{"conversationId":"` + session + `","display":"  /resume another"}`,
		"unknown type":    `{"conversationId":"` + session + `","display":"Assistant output","type":"assistant"}`,
		"wrong shape":     `{"conversationId":"` + session + `","display":["Request"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := newHome(t)
			h.history(
				`{"conversationId":"` + session + `","display":"Keep this topic"}` + "\n" + line + "\n",
			)

			if got := NewReader(h.root).Title(session); got != "Keep this topic" {
				t.Errorf("title = %q, want the last eligible prompt", got)
			}
		})
	}
}

func TestHistoryFollowsOnlyCompleteAppendedPrompts(t *testing.T) {
	t.Parallel()

	h := newHome(t)
	path := h.history(`{"conversationId":"` + session + `","display":"First request"}` + "\n")

	r := NewReader(h.root)
	if got := r.Title(session); got != "First request" {
		t.Fatalf("first title = %q, want first request", got)
	}

	appendHistory(t, path, `{"conversationId":"`+session+`","display":"Second request`)

	if got := r.Title(session); got != "First request" {
		t.Errorf("partial title = %q, want previous complete prompt", got)
	}

	appendHistory(t, path, `"}`+"\n")

	if got := r.Title(session); got != "Second request" {
		t.Errorf("updated title = %q, want appended prompt", got)
	}
}

func TestMissingOrTruncatedHistoryFallsBackToTheAnnotation(t *testing.T) {
	t.Parallel()

	h := newHome(t)
	h.annotate(`title:"Opening topic"`)

	r := NewReader(h.root)
	if got := r.Title(session); got != "Opening topic" {
		t.Errorf("missing-history title = %q, want annotation", got)
	}

	path := h.history(`{"conversationId":"` + session + `","display":"Current request"}` + "\n")
	if got := r.Title(session); got != "Current request" {
		t.Fatalf("title = %q, want history prompt", got)
	}

	h.history("{}\n")

	if got := r.Title(session); got != "Opening topic" {
		t.Errorf("truncated-history title = %q, want annotation", got)
	}

	appendHistory(t, path, `{"conversationId":"`+session+`","display":"Another request"}`+"\n")

	if got := r.Title(session); got != "Another request" {
		t.Fatalf("title = %q, want appended history prompt", got)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	if got := r.Title(session); got != "Opening topic" {
		t.Errorf("removed-history title = %q, want annotation", got)
	}
}

func TestAPromptContributesOnlyABoundedFirstLine(t *testing.T) {
	t.Parallel()

	h := newHome(t)
	h.history(
		`{"conversationId":"` + session + `","display":"  \n ` + strings.Repeat(
			"界",
			maxPrompt+1,
		) + `\nSecond line"}` + "\n",
	)

	if got := NewReader(h.root).Title(session); got != strings.Repeat("界", maxPrompt) {
		t.Errorf("title has %d runes, want %d from the first line", len([]rune(got)), maxPrompt)
	}
}

func TestLargeHistoryUsesOnlyTheRecentTail(t *testing.T) {
	t.Parallel()

	h := newHome(t)
	h.annotate(`title:"Opening topic"`)
	h.history(`{"conversationId":"` + session + `","display":"Old request"}` + "\n" +
		strings.Repeat("x", transcript.MaxScan) + "\n" +
		`{"conversationId":"` + otherSession + `","display":"Recent request"}` + "\n")

	r := NewReader(h.root)
	if got := r.Title(session); got != "Opening topic" {
		t.Errorf("title = %q, want annotation when prompt is outside the bounded tail", got)
	}

	if got := r.Title(otherSession); got != "Recent request" {
		t.Errorf("recent title = %q, want prompt from tail", got)
	}
}

func TestRetainForgetsClosedSessionHistory(t *testing.T) {
	t.Parallel()

	h := newHome(t)
	h.history(`{"conversationId":"` + session + `","display":"First request"}` + "\n")

	r := NewReader(h.root)
	if got := r.Title(session); got != "First request" {
		t.Fatalf("title = %q, want first request", got)
	}

	r.Retain(nil)
	h.history(`{"conversationId":"` + session + `","display":"Other request"}` + "\n")

	if got := r.Title(session); got != "Other request" {
		t.Errorf("reopened title = %q, want a fresh history read", got)
	}
}

func TestAnEmptyHomeReadsNoHistory(t *testing.T) {
	t.Parallel()

	if got := NewReader("").Title(session); got != "" {
		t.Errorf("title = %q, want no topic without a home", got)
	}
}
