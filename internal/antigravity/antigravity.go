// Package antigravity reads what an Antigravity CLI session is about from its
// prompt history, falling back to the agent's generated annotation. Herdr
// identifies the session; a directory alone cannot distinguish sessions.
package antigravity

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kryptamine/herdr-auto-title/internal/transcript"
)

// Agent is the label Herdr gives Antigravity CLI.
const Agent = "agy"

const maxPrompt = 200

type Reader struct {
	mu       sync.Mutex
	root     string
	sessions map[string]*sessionHistory
}

type sessionHistory struct {
	follower transcript.Follower
	prompt   string
}

// NewReader uses the user's home because Antigravity CLI's state directory
// cannot be moved. An empty home reads nothing.
func NewReader(home string) *Reader {
	if home == "" {
		return &Reader{}
	}

	return &Reader{
		root:     filepath.Join(home, ".gemini", "antigravity-cli"),
		sessions: make(map[string]*sessionHistory),
	}
}

// Title prefers the latest prompt, then the generated annotation. The id
// becomes part of a path, so it must be one herdr.AgentSessionInfo.ID accepted.
func (r *Reader) Title(sessionID string) string {
	if r.root == "" {
		return ""
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	held, known := r.sessions[sessionID]
	if !known {
		held = &sessionHistory{}
		r.sessions[sessionID] = held
	}

	read := held.follower.Follow(filepath.Join(r.root, "history.jsonl"),
		func() { held.prompt = "" },
		func(line string) { held.absorb(line, sessionID) },
	)
	if !read {
		held.prompt = ""
	}

	if held.prompt != "" {
		return held.prompt
	}

	return r.annotation(sessionID)
}

// Retain forgets closed sessions so their offsets and prompts do not outlive
// the panes holding them.
func (r *Reader) Retain(ids []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	kept := make(map[string]*sessionHistory, len(ids))
	for _, id := range ids {
		if held, known := r.sessions[id]; known {
			kept[id] = held
		}
	}

	r.sessions = kept
}

func (s *sessionHistory) absorb(line, sessionID string) {
	var entry struct {
		ConversationID string `json:"conversationId"`
		Display        string `json:"display"`
		Type           string `json:"type"`
	}
	if json.Unmarshal([]byte(line), &entry) != nil || entry.ConversationID != sessionID ||
		entry.Type != "" {
		return
	}

	text := strings.TrimSpace(entry.Display)
	if text == "" || strings.HasPrefix(text, "/") {
		return
	}

	first, _, _ := strings.Cut(text, "\n")
	s.prompt = transcript.Truncate(strings.TrimSpace(first), maxPrompt)
}
