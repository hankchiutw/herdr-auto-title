package antigravity

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// titlePattern matches the title field of an annotation, which is protobuf
// text format holding one quoted string.
var titlePattern = regexp.MustCompile(`(?m)^\s*title\s*:\s*("(?:[^"\\\n]|\\.)*")`)

func (r *Reader) annotation(sessionID string) string {
	//nolint:gosec // the root is the user's home and herdr refuses an id that is not a UUID
	content, err := os.ReadFile(filepath.Join(r.root, "annotations", sessionID+".pbtxt"))
	if err != nil {
		return ""
	}

	match := titlePattern.FindSubmatch(content)
	if match == nil {
		return ""
	}

	title, err := strconv.Unquote(string(match[1]))
	if err != nil {
		return ""
	}

	return title
}
