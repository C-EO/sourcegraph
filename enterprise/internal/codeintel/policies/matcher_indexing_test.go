package policies

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
)

func TestCommitsDescribedByPolicyForIndexing(t *testing.T) {
	mainGitserverClient := testUploadExpirerMockGitserverClient("main", testBranchHeads, testTagHeads, testBranchMembers, testCreatedAt)
	developGitserverClient := testUploadExpirerMockGitserverClient("develop", testBranchHeads, testTagHeads, testBranchMembers, testCreatedAt)

	runTest := func(t *testing.T, gitserverClient GitserverClient, policies []dbstore.ConfigurationPolicy, expectedPolicyMatches map[string][]PolicyMatch) {
		matcher, err := NewMatcher(gitserverClient, policies, IndexingExtractor, 50, false, true)
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
	}

	policyID := 42
	testDuration := time.Hour * 10

	t.Run("matches tag policies", func(t *testing.T) {
		policies := []dbstore.ConfigurationPolicy{
			{
				ID:                policyID,
				Type:              "GIT_TAG",
				Pattern:           "v1.*",
				IndexCommitMaxAge: &testDuration,
			},
		}

		runTest(t, mainGitserverClient, policies, map[string][]PolicyMatch{
			// N.B. tag v2.2.2 does not match filter
			// N.B. tag v1.2.2 does not fall within policy duration
			"deadbeef04": {PolicyMatch{Name: "v1.2.3", PolicyID: &policyID, PolicyDuration: &testDuration}},
		})
	})

	t.Run("matches branches tip policies", func(t *testing.T) {
		policies := []dbstore.ConfigurationPolicy{
			{
				ID:                policyID,
				Type:              "GIT_TREE",
				Pattern:           "ef/*",
				IndexCommitMaxAge: &testDuration,
			},
		}

		runTest(t, mainGitserverClient, policies, map[string][]PolicyMatch{
			// N.B. branch es/* does not match this filter
			// N.B. ef/feature-y does not fall within policy duration
			"deadbeef07": {PolicyMatch{Name: "ef/feature-x", PolicyID: &policyID, PolicyDuration: &testDuration}},
		})
	})

	t.Run("matches commits on branch policies", func(t *testing.T) {
		policies := []dbstore.ConfigurationPolicy{
			{
				ID:                       policyID,
				Type:                     "GIT_TREE",
				Pattern:                  "ef/*",
				IndexCommitMaxAge:        &testDuration,
				IndexIntermediateCommits: true,
			},
		}

		runTest(t, mainGitserverClient, policies, map[string][]PolicyMatch{
			// N.B. branch es/* does not match this filter
			// N.B. ef/feature-y does not fall within policy duration
			"deadbeef07": {PolicyMatch{Name: "ef/feature-x", PolicyID: &policyID, PolicyDuration: &testDuration}},
			"deadbeef08": {PolicyMatch{Name: "ef/feature-x", PolicyID: &policyID, PolicyDuration: &testDuration}},
		})
	})

	t.Run("return all matching policies for each commit", func(t *testing.T) {
		policyID1 := policyID
		policyID2 := policyID1 + 1
		policyID3 := policyID1 + 2

		testDuration1 := testDuration
		testDuration2 := time.Hour * 13
		testDuration3 := time.Hour * 20

		policies := []dbstore.ConfigurationPolicy{
			{
				ID:                       policyID,
				Type:                     "GIT_TREE",
				Pattern:                  "develop",
				IndexCommitMaxAge:        &testDuration1,
				IndexIntermediateCommits: true,
			},
			{
				ID:                       policyID2,
				Type:                     "GIT_TREE",
				Pattern:                  "*",
				IndexCommitMaxAge:        &testDuration2,
				IndexIntermediateCommits: true,
			},
			{
				ID:                       policyID3,
				Type:                     "GIT_TREE",
				Pattern:                  "feat/*",
				IndexCommitMaxAge:        &testDuration3,
				IndexIntermediateCommits: true,
			},
		}

		runTest(t, mainGitserverClient, policies, map[string][]PolicyMatch{
			"deadbeef01": {
				PolicyMatch{Name: "develop", PolicyID: &policyID1, PolicyDuration: &testDuration1},
				PolicyMatch{Name: "develop", PolicyID: &policyID2, PolicyDuration: &testDuration2},
			},
			"deadbeef02": {
				PolicyMatch{Name: "develop", PolicyID: &policyID1, PolicyDuration: &testDuration1},
				PolicyMatch{Name: "feat/blank", PolicyID: &policyID2, PolicyDuration: &testDuration2},
				PolicyMatch{Name: "feat/blank", PolicyID: &policyID3, PolicyDuration: &testDuration3},
			},
			"deadbeef03": {
				PolicyMatch{Name: "develop", PolicyID: &policyID1, PolicyDuration: &testDuration1},
				PolicyMatch{Name: "develop", PolicyID: &policyID2, PolicyDuration: &testDuration2},
			},
			"deadbeef04": {
				PolicyMatch{Name: "develop", PolicyID: &policyID1, PolicyDuration: &testDuration1},
				PolicyMatch{Name: "develop", PolicyID: &policyID2, PolicyDuration: &testDuration2},
			},

			// N.B. deadbeef05 too old to match policy 1
			// N.B. deadbeef06 and deadbeef09 are too old for any matching policy
			"deadbeef05": {PolicyMatch{Name: "develop", PolicyID: &policyID2, PolicyDuration: &testDuration2}},
			"deadbeef07": {PolicyMatch{Name: "ef/feature-x", PolicyID: &policyID2, PolicyDuration: &testDuration2}},
			"deadbeef08": {PolicyMatch{Name: "ef/feature-x", PolicyID: &policyID2, PolicyDuration: &testDuration2}},
		})
	})

	t.Run("matches commit policies", func(t *testing.T) {
		policies := []dbstore.ConfigurationPolicy{
			{
				ID:                policyID,
				Type:              "GIT_COMMIT",
				Pattern:           "deadbeef04",
				IndexCommitMaxAge: &testDuration,
			},
		}

		runTest(t, mainGitserverClient, policies, map[string][]PolicyMatch{
			"deadbeef04": {PolicyMatch{Name: "deadbeef04", PolicyID: &policyID, PolicyDuration: &testDuration}},
		})
	})

	t.Run("does not match commit policies outside of policy duration", func(t *testing.T) {
		policies := []dbstore.ConfigurationPolicy{
			{
				ID:                policyID,
				Type:              "GIT_COMMIT",
				Pattern:           "deadbeef05",
				IndexCommitMaxAge: &testDuration,
			},
		}

		runTest(t, mainGitserverClient, policies, map[string][]PolicyMatch{
			// N.B. deadbeef05 does not fall within policy duration
		})
	})

	t.Run("does not match a default policy", func(t *testing.T) {
		runTest(t, developGitserverClient, nil, nil)
	})
}
