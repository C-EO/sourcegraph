package policies

import (
	"context"
	"time"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
)

func testUploadExpirerMockGitserverClient(defaultBranchName string, branchHeads, tagHeads map[string]string, branchMembers map[string][]string, createdAt map[string]time.Time) *MockGitserverClient {
	gitserverClient := NewMockGitserverClient()

	gitserverClient.CommitDateFunc.SetDefaultHook(func(ctx context.Context, repositoryID int, commit string) (time.Time, error) {
		return createdAt[commit], nil
	})

	gitserverClient.RefDescriptionsFunc.SetDefaultHook(func(ctx context.Context, repositoryID int) (map[string][]gitserver.RefDescription, error) {
		refDescriptions := map[string][]gitserver.RefDescription{}

		for branch, commit := range branchHeads {
			refDescriptions[commit] = append(refDescriptions[commit], gitserver.RefDescription{
				Name:            branch,
				Type:            gitserver.RefTypeBranch,
				IsDefaultBranch: branch == defaultBranchName,
				CreatedDate:     createdAt[commit],
			})
		}

		for tag, commit := range tagHeads {
			refDescriptions[commit] = append(refDescriptions[commit], gitserver.RefDescription{
				Name:        tag,
				Type:        gitserver.RefTypeTag,
				CreatedDate: createdAt[commit],
			})
		}

		return refDescriptions, nil
	})

	gitserverClient.CommitsUniqueToBranchFunc.SetDefaultHook(func(ctx context.Context, repositoryID int, branchName string, isDefaultBranch bool, maxAge *time.Time) (map[string]time.Time, error) {
		branches := map[string]time.Time{}
		for _, commit := range branchMembers[branchName] {
			if maxAge == nil || !createdAt[commit].Before(*maxAge) {
				branches[commit] = createdAt[commit]
			}
		}

		return branches, nil
	})

	return gitserverClient
}
