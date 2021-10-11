package janitor

import (
	"context"
	"time"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
)

// testUploadExpirerMockGitserverClient returns a mock GitserverClient instance that
// has default behaviors useful for testing the upload expirer.
func testUploadExpirerMockGitserverClient(branchMap map[string]map[string]string, tagMap map[string][]string) *MockGitserverClient {
	gitserverClient := NewMockGitserverClient()

	gitserverClient.CommitDateFunc.SetDefaultHook(func(ctx context.Context, repositoryID int, commit string) (time.Time, error) {
		return time.Time{}, nil
	})

	gitserverClient.RefDescriptionsFunc.SetDefaultHook(func(ctx context.Context, repositoryID int) (map[string][]gitserver.RefDescription, error) {
		refDescriptions := map[string][]gitserver.RefDescription{}
		for commit, branches := range branchMap {
			for branch, tip := range branches {
				if tip != commit {
					continue
				}

				refDescriptions[commit] = append(refDescriptions[commit], gitserver.RefDescription{
					Name:            branch,
					Type:            gitserver.RefTypeBranch,
					IsDefaultBranch: branch == "main",
				})
			}
		}

		for commit, tags := range tagMap {
			for _, tag := range tags {
				refDescriptions[commit] = append(refDescriptions[commit], gitserver.RefDescription{
					Name: tag,
					Type: gitserver.RefTypeTag,
				})
			}
		}

		return refDescriptions, nil
	})

	gitserverClient.CommitsUniqueToBranchFunc.SetDefaultHook(func(ctx context.Context, repositoryID int, branchName string, isDefaultBranch bool, maxAge *time.Time) (map[string]time.Time, error) {
		branches := map[string]time.Time{}
		for commit, branchMap := range branchMap {
			if _, ok := branchMap[branchName]; ok {
				branches[commit] = time.Time{}
			}
		}

		return branches, nil
	})

	return gitserverClient
}
