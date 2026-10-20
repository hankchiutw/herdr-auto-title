package resolver

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kryptamine/herdr-auto-title/internal/state"
)

func TestABranchNamedDirectoryUsesTheBranchWidth(t *testing.T) {
	t.Parallel()

	const (
		branch   = "bugfix-asatretdinov-cpanel-uapi-mc-13675"
		activity = "Fix OAuth redirect"
		wide     = 100
	)

	for _, test := range []struct {
		name, directory, branch, trunk, want string
		max                                  int
	}{
		{
			name: "issue key", directory: branch, branch: branch, trunk: "main",
			max: DefaultBranchMaxLength, want: "MC-13675 › " + activity,
		},
		{
			name: "case insensitive match", directory: strings.ToUpper(branch),
			branch: branch, trunk: "main",
			max: DefaultBranchMaxLength, want: "MC-13675 › " + activity,
		},
		{
			name: "recorded default branch", directory: branch, branch: branch, trunk: branch,
			max: DefaultBranchMaxLength, want: "MC-13675 › " + activity,
		},
		{
			name: "custom width", directory: "feature-dashboard-redesign",
			branch: "feature-dashboard-redesign", trunk: "main",
			max: len("feature-dashboard"), want: "feature-dashboard › " + activity,
		},
		{
			name: "short name", directory: "oauth-", branch: "oauth-", trunk: "main",
			max: DefaultBranchMaxLength, want: "oauth- › " + activity,
		},
		{
			name: "separator-only branch", directory: "______________",
			branch: "______________", trunk: "main",
			max: DefaultBranchMaxLength, want: "______________ › " + activity,
		},
		{
			name: "wide grapheme with narrow bound", directory: "\U0001F600",
			branch: "\U0001F600", trunk: "main",
			max: 1, want: "\U0001F600 › " + activity,
		},
		{
			name: "branches disabled", directory: branch, branch: branch, trunk: "main",
			max: 0, want: branch + " › " + activity,
		},
		{
			name: "issue directory without checkout", directory: branch,
			max: DefaultBranchMaxLength, want: "MC-13675 › " + activity,
		},
		{
			name: "issue directory with different branch", directory: branch,
			branch: "feat/oauth", trunk: "main",
			max: DefaultBranchMaxLength, want: "MC-13675 › feat/oauth › " + activity,
		},
		{
			name: "ordinary directory", directory: "stytrix-beta-e2e-ai-chat-etc",
			max: DefaultBranchMaxLength, want: "stytrix-beta-e2e-ai-chat-etc › " + activity,
		},
		{
			name: "project prefix without checkout", directory: "stytrix-beta.st-146-e2e-ai-chat-etc",
			max: DefaultBranchMaxLength, want: "ST-146 › " + activity,
		},
		{
			name:      "project prefix with matching branch",
			directory: "stytrix-beta.st-146-e2e-ai-chat-etc", branch: "st-146-e2e-ai-chat-etc",
			trunk: "main", max: DefaultBranchMaxLength, want: "ST-146 › " + activity,
		},
		{
			name:      "project prefix with different issue",
			directory: "stytrix-beta.st-146-e2e-ai-chat-etc", branch: "fix/OTHER-123-problem",
			trunk: "main", max: DefaultBranchMaxLength, want: "ST-146 › OTHER-123 › " + activity,
		},
		{
			name: "short issue directory", directory: "repo.ST-146",
			max: DefaultBranchMaxLength, want: "repo.ST-146 › " + activity,
		},
		{
			name: "issue key stays whole", directory: "stytrix-beta.st-146-e2e-ai-chat-etc",
			max: len("ST"), want: "ST-146 › " + activity,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pane := repoPane(test.branch, test.trunk)
			pane.Dir = filepath.Join(filepath.Dir(dashboard), test.directory)
			pane.TerminalTitle = activity

			chain := Default(Options{MaxLength: wide, BranchMax: test.max})
			if got := chain.Resolve(tabWithPane(pane)).Name; got != test.want {
				t.Errorf("title = %q, want %q", got, test.want)
			}
		})
	}
}

func TestABranchDirectoryIsComparedBeforeShortening(t *testing.T) {
	t.Parallel()

	const (
		directory = "stytrix-beta.st-146-e2e-ai-chat-etc"
		branch    = "st-146-e2e-ai-chat-etc"
		activity  = "Fix OAuth redirect"
		wide      = 100
	)

	for _, test := range []struct {
		name, workspace, title, handle, want string
		paneLabel                            bool
	}{
		{name: "workspace", workspace: directory, title: activity, want: activity},
		{name: "activity", title: directory, want: "ST-146"},
		{name: "tab above pane", title: activity, paneLabel: true, want: activity},
		{
			name: "handle", title: activity, handle: directory, paneLabel: true,
			want: directory + " › " + activity,
		},
		{
			name: "different full workspace", workspace: "ST-146", title: activity,
			want: "ST-146 › " + activity,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pane := repoPane(branch, "main")
			pane.Dir = filepath.Join(filepath.Dir(dashboard), directory)
			pane.TerminalTitle = test.title
			pane.Handle = test.handle
			tab := tabWithPane(pane)
			tab.WorkspaceName = test.workspace

			chain := Default(Options{
				MaxLength: wide, BranchMax: DefaultBranchMaxLength, ShowHandle: true,
			})

			got := chain.Resolve(tab).Name
			if test.paneLabel {
				got = chain.ResolvePanes(tab)[0].Name
			}

			if got != test.want {
				t.Errorf("title = %q, want %q", got, test.want)
			}
		})
	}
}

func TestARemotePaneDoesNotShortenItsLocalDirectory(t *testing.T) {
	t.Parallel()

	const branch = "bugfix-asatretdinov-cpanel-uapi-mc-13675"

	pane := repoPane(branch, "main")
	pane.Dir = filepath.Join(filepath.Dir(dashboard), branch)
	pane.Processes = []state.Process{{Name: "ssh", Args: []string{"ssh", "prod-01"}}}
	chain := New(Options{BranchMax: DefaultBranchMaxLength}, NewCWD(""))

	if got := chain.Resolve(tabWithPane(pane)).Name; got != branch {
		t.Errorf("title = %q, want %q", got, branch)
	}
}
