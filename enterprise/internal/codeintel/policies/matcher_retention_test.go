package policies

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
)

func TestCommitsDescribedByPolicyForRetention(t *testing.T) {
	mainGitserverClient := testUploadExpirerMockGitserverClient("main", testBranchHeads, testTagHeads, testBranchMembers, testCreatedAt)
	developGitserverClient := testUploadExpirerMockGitserverClient("develop", testBranchHeads, testTagHeads, testBranchMembers, testCreatedAt)

	runTest := func(t *testing.T, gitserverClient *MockGitserverClient, policies []dbstore.ConfigurationPolicy, expectedPolicyMatches map[string][]PolicyMatch) {
		matcher, err := NewMatcher(gitserverClient, policies, RetentionExtractor, 50, true, false)
		if err != nil {
			t.Fatalf("unexpected error creating matcher: %s", err)
		}
		policyMatches, err := matcher.CommitsDescribedByPolicy(context.Background(), testNow)
		if err != nil {
			t.Fatalf("unexpected error finding matches: %s", err)
		}

		sortPolicyMatchesMap(policyMatches)
		sortPolicyMatchesMap(expectedPolicyMatches)

		if diff := cmp.Diff(expectedPolicyMatches, policyMatches); diff != "" {
			t.Errorf("unexpected policy matches (-want +got):\n%s", diff)
		}

		for i, call := range gitserverClient.CommitsUniqueToBranchFunc.History() {
			if call.Arg4 != nil {
				t.Errorf("unexpected restriction of git results by date: call #%d", i)
			}
		}
	}

	policyID := 42
	testDuration := time.Hour * 24

	t.Run("matches tag policies", func(t *testing.T) {
		policies := []dbstore.ConfigurationPolicy{
			{
				ID:                policyID,
				Type:              "GIT_TAG",
				Pattern:           "v1.*",
				RetentionDuration: &testDuration,
			},
		}

		runTest(t, mainGitserverClient, policies, map[string][]PolicyMatch{
			// N.B. tag v2.2.2 does not match filter
			"deadbeef04": {PolicyMatch{Name: "v1.2.3", PolicyID: &policyID, PolicyDuration: &testDuration}},
			"deadbeef05": {PolicyMatch{Name: "v1.2.2", PolicyID: &policyID, PolicyDuration: &testDuration}},
		})
	})

	t.Run("matches branches tip policies", func(t *testing.T) {
		policies := []dbstore.ConfigurationPolicy{
			{
				ID:                policyID,
				Type:              "GIT_TREE",
				Pattern:           "ef/*",
				RetentionDuration: &testDuration,
			},
		}

		runTest(t, mainGitserverClient, policies, map[string][]PolicyMatch{
			// N.B. branch es/* does not match this filter
			"deadbeef07": {PolicyMatch{Name: "ef/feature-x", PolicyID: &policyID, PolicyDuration: &testDuration}},
			"deadbeef09": {PolicyMatch{Name: "ef/feature-y", PolicyID: &policyID, PolicyDuration: &testDuration}},
		})
	})

	t.Run("matches commits on branch policies", func(t *testing.T) {
		policies := []dbstore.ConfigurationPolicy{
			{
				ID:                        policyID,
				Type:                      "GIT_TREE",
				Pattern:                   "ef/*",
				RetentionDuration:         &testDuration,
				RetainIntermediateCommits: true,
			},
		}

		runTest(t, mainGitserverClient, policies, map[string][]PolicyMatch{
			// N.B. branch es/* does not match this filter
			"deadbeef07": {PolicyMatch{Name: "ef/feature-x", PolicyID: &policyID, PolicyDuration: &testDuration}},
			"deadbeef08": {PolicyMatch{Name: "ef/feature-x", PolicyID: &policyID, PolicyDuration: &testDuration}},
			"deadbeef09": {PolicyMatch{Name: "ef/feature-y", PolicyID: &policyID, PolicyDuration: &testDuration}},
		})
	})

	t.Run("matches commit policies", func(t *testing.T) {
		policies := []dbstore.ConfigurationPolicy{
			{
				ID:                policyID,
				Type:              "GIT_COMMIT",
				Pattern:           "deadbeef04",
				RetentionDuration: &testDuration,
			},
		}

		runTest(t, mainGitserverClient, policies, map[string][]PolicyMatch{
			"deadbeef04": {PolicyMatch{Name: "deadbeef04", PolicyID: &policyID, PolicyDuration: &testDuration}},
		})
	})

	t.Run("matches implicit tip of default branch policy", func(t *testing.T) {
		runTest(t, developGitserverClient, nil, map[string][]PolicyMatch{
			"deadbeef01": {{Name: "develop", PolicyID: nil, PolicyDuration: nil}},
		})
	})
}
