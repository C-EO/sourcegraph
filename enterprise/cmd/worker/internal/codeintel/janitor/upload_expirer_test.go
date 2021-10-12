package janitor

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/policies"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
)

func TestUploadExpirer(t *testing.T) {
	now := timeutil.Now()
	t1 := now.Add(-time.Hour)                 // 1 hour old
	t2 := now.Add(-time.Hour * 24 * 7)        // 1 week ago
	t3 := now.Add(-time.Hour * 24 * 30 * 5)   // 5 months ago
	t4 := now.Add(-time.Hour * 24 * 30 * 9)   // 9 months ago
	t5 := now.Add(-time.Hour * 24 * 30 * 18)  // 18 months ago
	t6 := now.Add(-time.Hour * 24 * 365 * 2)  // 3 years ago
	t8 := now.Add(-time.Hour * 24 * 365 * 15) // 15 years ago

	uploads := []dbstore.Upload{
		//
		// Repository 50

		// 1 week old
		// tip of develop (PROTECTED, younger than 3 months)
		{ID: 1, RepositoryID: 50, Commit: "deadbeef01", State: "completed", UploadedAt: t2},

		// 1 week old
		// on develop (UNPROTECTED, not tip)
		// tip of feat/blank (PROTECTED, younger than 3 months)
		{ID: 2, RepositoryID: 50, Commit: "deadbeef02", State: "completed", UploadedAt: t2},

		// 5 months old
		// on develop (UNPROTECTED, not tip)
		{ID: 3, RepositoryID: 50, Commit: "deadbeef03", State: "completed", UploadedAt: t3},

		// 5 months old
		// on develop (UNPROTECTED, not tip)
		// tag v1.2.3 (PROTECTED, younger than 6 months)
		{ID: 4, RepositoryID: 50, Commit: "deadbeef04", State: "completed", UploadedAt: t3},

		// 9 months old
		// on develop (UNPROTECTED, not tip)
		// tag v1.2.2 (UNPROTECTED, older than 6 months)
		{ID: 5, RepositoryID: 50, Commit: "deadbeef05", State: "completed", UploadedAt: t4},

		// 5 months old
		// tip of es/feature-z (UNPROTECTED, older than 3 months)
		{ID: 6, RepositoryID: 50, Commit: "deadbeef06", State: "completed", UploadedAt: t3},

		// 9 months old
		// tip of ef/feature-x (PROTECTED, younger than 2 years)
		{ID: 7, RepositoryID: 50, Commit: "deadbeef07", State: "completed", UploadedAt: t4},

		// 18 months old
		// on ef/feature-x (PROTECTED, younger than 2 years)
		{ID: 8, RepositoryID: 50, Commit: "deadbeef08", State: "completed", UploadedAt: t5},

		// 3 years old
		// tip of ef/feature-y (UNPROTECTED, older than 2 years)
		{ID: 9, RepositoryID: 50, Commit: "deadbeef09", State: "completed", UploadedAt: t6},

		//
		// Repository 51

		// 9 months old
		// tip of ef/feature-w (UNPROTECTED, policy does not apply to this repo)
		{ID: 10, RepositoryID: 51, Commit: "deadbeef10", State: "completed", UploadedAt: t4},

		//
		// Repository 52

		// 15 years old
		// tip of main (PROTECTED, no duration)
		{ID: 11, RepositoryID: 52, Commit: "deadbeef11", State: "completed", UploadedAt: t8},

		// 15 years old
		// on main (UNPROTECTED, not tip)
		{ID: 12, RepositoryID: 52, Commit: "deadbeef12", State: "completed", UploadedAt: t8},

		//
		// Repository 53

		// 1 hour old
		// covered by commit rule (PROTECTED, younger than 1 day)
		{ID: 13, RepositoryID: 53, Commit: "deadbeef13", State: "completed", UploadedAt: t1},
	}

	d1 := time.Hour * 24           // 1 day
	d2 := time.Hour * 24 * 90      // 3 months
	d3 := time.Hour * 24 * 180     // 6 months
	d4 := time.Hour * 24 * 365 * 2 // 2 years

	globalPolicies := []dbstore.ConfigurationPolicy{
		{
			ID:                0,
			Type:              "GIT_TREE",
			Pattern:           "main",
			RetentionEnabled:  true,
			RetentionDuration: nil, // indefinite
		},
		{
			ID:                2,
			Type:              "GIT_TREE",
			Pattern:           "*",
			RetentionEnabled:  true,
			RetentionDuration: &d2,
		},
		{
			ID:                3,
			Type:              "GIT_TAG",
			Pattern:           "*",
			RetentionEnabled:  true,
			RetentionDuration: &d3,
		},
	}

	policiesByRepositoryID := map[int][]dbstore.ConfigurationPolicy{
		50: {
			dbstore.ConfigurationPolicy{
				ID:                        4,
				Type:                      "GIT_TREE",
				Pattern:                   "ef/*",
				RetentionEnabled:          true,
				RetainIntermediateCommits: true,
				RetentionDuration:         &d4,
			},
		},
		53: {
			dbstore.ConfigurationPolicy{
				ID:                1,
				Type:              "GIT_COMMIT",
				Pattern:           "deadbeef13",
				RetentionEnabled:  true,
				RetentionDuration: &d1,
			},
		},
	}

	newMatch := func(name string, id int) policies.PolicyMatch {
		return policies.PolicyMatch{
			Name:           name,
			PolicyID:       &id,
			PolicyDuration: []*time.Duration{nil, &d1, &d2, &d3, &d4}[id],
		}
	}

	// Relevant return data from invocation of commitsDescribedByPolicy with
	// the configurationpolicies defined in this test.
	policyMatches := map[int]map[string][]policies.PolicyMatch{
		50: {
			"deadbeef01": {newMatch("develop", 2)},                                   // 3 months
			"deadbeef02": {newMatch("feat/blank", 2)},                                // 3 months
			"deadbeef04": {newMatch("v1.2.3", 3)},                                    // 6 months
			"deadbeef05": {newMatch("v1.2.2", 3)},                                    // 6 months
			"deadbeef06": {newMatch("es/feature-z", 2)},                              // 3 months
			"deadbeef07": {newMatch("ef/feature-x", 2), newMatch("ef/feature-x", 4)}, // 3 months; 2 years
			"deadbeef08": {newMatch("ef/feature-x", 4)},                              // 2 years
			"deadbeef09": {newMatch("ef/feature-y", 2), newMatch("ef/feature-y", 4)}, // 3 months; 2 years
		},
		51: {"deadbeef10": {newMatch("ef/feature-w", 2)}},              // 3 months
		52: {"deadbeef11": {newMatch("main", 0), newMatch("main", 2)}}, // forever; 3 months
		53: {"deadbeef13": {newMatch("deadbeef13", 1)}},                // 1 day
	}

	dbStore := testUploadExpirerMockDBStore(globalPolicies, policiesByRepositoryID, uploads)
	policyMatcher := NewMockPolicyMatcher()
	policyMatcher.CommitsDescribedByPolicyFunc.SetDefaultHook(func(ctx context.Context, repositoryID int, policies []dbstore.ConfigurationPolicy, now time.Time) (map[string][]policies.PolicyMatch, error) {
		return policyMatches[repositoryID], nil
	})

	uploadExpirer := &uploadExpirer{
		dbStore:                dbStore,
		policyMatcher:          policyMatcher,
		metrics:                newMetrics(&observation.TestContext),
		repositoryProcessDelay: 24 * time.Hour,
		repositoryBatchSize:    100,
		uploadProcessDelay:     24 * time.Hour,
		uploadBatchSize:        100,
		commitBatchSize:        100,
		branchesCacheMaxKeys:   10000,
	}

	if err := uploadExpirer.Handle(context.Background()); err != nil {
		t.Fatalf("unexpected error from handle: %s", err)
	}

	var protectedIDs []int
	for _, call := range dbStore.UpdateUploadRetentionFunc.History() {
		protectedIDs = append(protectedIDs, call.Arg1...)
	}
	sort.Ints(protectedIDs)

	var expiredIDs []int
	for _, call := range dbStore.UpdateUploadRetentionFunc.History() {
		expiredIDs = append(expiredIDs, call.Arg2...)
	}
	sort.Ints(expiredIDs)

	expectedProtectedIDs := []int{1, 2, 4, 7, 8, 11, 13}
	if diff := cmp.Diff(expectedProtectedIDs, protectedIDs); diff != "" {
		t.Errorf("unexpected protected upload identifiers (-want +got):\n%s", diff)
	}

	expectedExpiredIDs := []int{3, 5, 6, 9, 10, 12}
	if diff := cmp.Diff(expectedExpiredIDs, expiredIDs); diff != "" {
		t.Errorf("unexpected expired upload identifiers (-want +got):\n%s", diff)
	}

	calls := policyMatcher.CommitsDescribedByPolicyFunc.History()
	if len(calls) != 4 {
		t.Fatalf("unexpected number of calls to CommitsDescribedByPolicy. want=%d have=%d", 4, len(calls))
	}

	expectedPolicies := map[int][]dbstore.ConfigurationPolicy{
		50: append(globalPolicies, policiesByRepositoryID[50]...),
		51: globalPolicies,
		52: globalPolicies,
		53: append(globalPolicies, policiesByRepositoryID[53]...),
	}

	for _, call := range calls {
		if diff := cmp.Diff(expectedPolicies[call.Arg1], call.Arg2); diff != "" {
			t.Errorf("unexpected policies supplied to CommitsDescribedByPolicy (-want +got):\n%s", diff)
		}
	}
}
