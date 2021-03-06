package gitserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
)

// TODO(efritz) - move this declaration (MaxCommitsPerUpdate = MaxTraversalLimit * 1.5)
const MaxCommitsPerUpdate = 150

// Head determines the tip commit of the default branch for the given repository.
func Head(ctx context.Context, store store.Store, repositoryID int) (string, error) {
	return execGitCommand(ctx, store, repositoryID, "rev-parse", "HEAD")
}

// CommitsNear returns a map from a commit to parent commits. The commits populating the
// map are the MaxCommitsPerUpdate closest ancestors from the given commit.
func CommitsNear(ctx context.Context, store store.Store, repositoryID int, commit string) (map[string][]string, error) {
	out, err := execGitCommand(ctx, store, repositoryID, "log", "--pretty=%H %P", commit, fmt.Sprintf("-%d", MaxCommitsPerUpdate))
	if err != nil {
		return nil, err
	}

	return parseParents(strings.Split(out, "\n")), nil
}

// CommitGraph returns the commit graph for the given repository as a mapping from a commit
// to its parents.
func CommitGraph(ctx context.Context, store store.Store, repositoryID int) (map[string][]string, error) {
	out, err := execGitCommand(ctx, store, repositoryID, "log", "--all", "--pretty=%H %P")
	if err != nil {
		return nil, err
	}

	return parseParents(strings.Split(out, "\n")), nil
}

// parseParents converts the output of git log into a map from commits to parent commits.
// If a commit is listed but has no ancestors then its parent slice is empty but is still
// present in the map.
func parseParents(pair []string) map[string][]string {
	commits := map[string][]string{}

	for _, pair := range pair {
		line := strings.TrimSpace(pair)
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")
		commits[parts[0]] = parts[1:]
	}

	return commits
}
