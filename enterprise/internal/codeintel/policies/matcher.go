package policies

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gobwas/glob"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

// TODO
type Matcher struct {
	gitserverClient           GitserverClient
	policies                  []dbstore.ConfigurationPolicy
	patterns                  map[string]glob.Glob
	extractor                 Extractor
	repositoryID              int
	includeTipOfDefaultBranch bool
	filterByCreatedDate       bool
}

func NewMatcher(
	gitserverClient GitserverClient,
	policies []dbstore.ConfigurationPolicy,
	extractor Extractor,
	repositoryID int,
	includeTipOfDefaultBranch bool,
	filterByCreatedDate bool,
) (*Matcher, error) {
	patterns, err := compilePatterns(policies)
	if err != nil {
		return nil, err
	}

	return &Matcher{
		gitserverClient:           gitserverClient,
		policies:                  policies,
		patterns:                  patterns,
		extractor:                 extractor,
		repositoryID:              repositoryID,
		includeTipOfDefaultBranch: includeTipOfDefaultBranch,
		filterByCreatedDate:       filterByCreatedDate,
	}, nil
}

// compilePatterns constructs a map from patterns in each given policy to a compiled glob object used
// to match to commits, branch names, and tag names. If there are multiple policies with the same pattern,
// the pattern is compiled only once.
func compilePatterns(policies []dbstore.ConfigurationPolicy) (map[string]glob.Glob, error) {
	patterns := make(map[string]glob.Glob, len(policies))
	for _, policy := range policies {
		if _, ok := patterns[policy.Pattern]; ok || policy.Type == dbstore.GitObjectTypeCommit {
			continue
		}

		pattern, err := glob.Compile(policy.Pattern)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to compile glob pattern `%s` in configuration policy %d", policy.Pattern, policy.ID))
		}

		patterns[policy.Pattern] = pattern
	}

	return patterns, nil
}

// TODO - document
type PolicyMatch struct {
	Name           string
	PolicyID       *int
	PolicyDuration *time.Duration
}

// TODO - document
type branchRequestMeta struct {
	isDefaultBranch     bool
	policyDurationByIDs map[int]*time.Duration
}

// TODO - document
func (m *Matcher) CommitsDescribedByPolicy(ctx context.Context, now time.Time) (map[string][]PolicyMatch, error) {
	if len(m.policies) == 0 && !m.includeTipOfDefaultBranch {
		return nil, nil
	}

	refDescriptions, err := m.gitserverClient.RefDescriptions(ctx, m.repositoryID)
	if err != nil {
		return nil, errors.Wrap(err, "gitserver.RefDescriptions")
	}

	commitMap := map[string][]PolicyMatch{}
	branchRequests := map[string]branchRequestMeta{}

	for commit, refDescriptions := range refDescriptions {
		for _, refDescription := range refDescriptions {
			switch refDescription.Type {
			case gitserver.RefTypeTag:
				// TODO - document
				m.resolveTagReference(commitMap, branchRequests, commit, refDescription, now)

			case gitserver.RefTypeBranch:
				// TODO - document
				m.resolveBranchReference(commitMap, branchRequests, commit, refDescription, now)
			}
		}
	}

	// TODO - document
	if err := m.resolveBranchMembership(ctx, commitMap, branchRequests, now); err != nil {
		return nil, err
	}

	// TODO - document
	if err := m.resolveCommitPolicies(ctx, commitMap, now); err != nil {
		return nil, err
	}

	return commitMap, nil
}

func (m *Matcher) resolveTagReference(
	commitMap map[string][]PolicyMatch,
	branchRequests map[string]branchRequestMeta,
	commit string,
	refDescription gitserver.RefDescription,
	now time.Time,
) {
	visitor := func(policy dbstore.ConfigurationPolicy) {
		policyDuration, _ := m.extractor(policy)

		commitMap[commit] = append(commitMap[commit], PolicyMatch{
			Name:           refDescription.Name,
			PolicyID:       &policy.ID,
			PolicyDuration: policyDuration,
		})
	}

	m.forEachMatchingPolicy(refDescription, dbstore.GitObjectTypeTag, visitor, now)
}

func (m *Matcher) resolveBranchReference(
	commitMap map[string][]PolicyMatch,
	branchRequests map[string]branchRequestMeta,
	commit string,
	refDescription gitserver.RefDescription,
	now time.Time,
) {
	// TODO - document
	if refDescription.IsDefaultBranch && m.includeTipOfDefaultBranch {
		commitMap[commit] = append(commitMap[commit], PolicyMatch{
			Name:           refDescription.Name,
			PolicyID:       nil,
			PolicyDuration: nil,
		})
	}

	visitor := func(policy dbstore.ConfigurationPolicy) {
		policyDuration, _ := m.extractor(policy)

		commitMap[commit] = append(commitMap[commit], PolicyMatch{
			Name:           refDescription.Name,
			PolicyID:       &policy.ID,
			PolicyDuration: policyDuration,
		})

		// TODO - document
		if policyDuration, includeIntermediateCommits := m.extractor(policy); includeIntermediateCommits {
			meta, ok := branchRequests[refDescription.Name]
			if !ok {
				meta.policyDurationByIDs = map[int]*time.Duration{}
			}

			meta.policyDurationByIDs[policy.ID] = policyDuration
			meta.isDefaultBranch = meta.isDefaultBranch || refDescription.IsDefaultBranch
			branchRequests[refDescription.Name] = meta
		}
	}

	m.forEachMatchingPolicy(refDescription, dbstore.GitObjectTypeTree, visitor, now)
}

func (m *Matcher) resolveBranchMembership(ctx context.Context, commitMap map[string][]PolicyMatch, branchRequests map[string]branchRequestMeta, now time.Time) error {
	for branchName, branchRequestMeta := range branchRequests {
		maxCommitAge := getMaxAge(branchRequestMeta.policyDurationByIDs, now)

		// TODO - document
		if !m.filterByCreatedDate {
			maxCommitAge = nil
		}

		commitDates, err := m.gitserverClient.CommitsUniqueToBranch(ctx, m.repositoryID, branchName, branchRequestMeta.isDefaultBranch, maxCommitAge)
		if err != nil {
			return errors.Wrap(err, "gitserver.CommitsUniqueToBranch")
		}

		for commit, commitDate := range commitDates {
		policies:
			for policyID, policyDuration := range branchRequestMeta.policyDurationByIDs {
				for _, match := range commitMap[commit] {
					if match.PolicyID != nil && *match.PolicyID == policyID {
						// TODO - document
						continue policies
					}
				}

				// TODO - need to re-check duration against policies
				if m.filterByCreatedDate && policyDuration != nil && now.Sub(commitDate) > *policyDuration {
					// TODO - ensure tested
					continue policies
				}

				// Don't capture loop variable pointer
				localPolicyID := policyID

				commitMap[commit] = append(commitMap[commit], PolicyMatch{
					Name:           branchName,
					PolicyID:       &localPolicyID,
					PolicyDuration: policyDuration,
				})
			}
		}
	}

	return nil
}

func (m *Matcher) resolveCommitPolicies(ctx context.Context, commitMap map[string][]PolicyMatch, now time.Time) error {
	for _, policy := range m.policies {
		if policy.Type == dbstore.GitObjectTypeCommit {
			commitDate, err := m.gitserverClient.CommitDate(ctx, m.repositoryID, policy.Pattern)
			if err != nil {
				if errcode.IsNotFound(err) {
					return nil
				}

				return errors.Wrap(err, "gitserver.ResolveRevision")
			}

			policyDuration, _ := m.extractor(policy)

			if m.filterByCreatedDate && policyDuration != nil && now.Sub(commitDate) > *policyDuration {
				continue
			}

			commitMap[policy.Pattern] = append(commitMap[policy.Pattern], PolicyMatch{
				Name:           policy.Pattern,
				PolicyID:       &policy.ID,
				PolicyDuration: policyDuration,
			})
		}
	}

	return nil
}

func (m *Matcher) forEachMatchingPolicy(refDescription gitserver.RefDescription, targetObjectType dbstore.GitObjectType, f func(policy dbstore.ConfigurationPolicy), now time.Time) {
	for _, policy := range m.policies {
		if policy.Type == targetObjectType && m.policyMatchesRefDescription(policy, refDescription, now) {
			f(policy)
		}
	}
}

func (m *Matcher) policyMatchesRefDescription(policy dbstore.ConfigurationPolicy, refDescription gitserver.RefDescription, now time.Time) bool {
	if !m.patterns[policy.Pattern].Match(refDescription.Name) {
		// Name doesn't match
		return false
	}

	if policyDuration, _ := m.extractor(policy); m.filterByCreatedDate && policyDuration != nil && now.Sub(refDescription.CreatedDate) > *policyDuration {
		// Too old
		return false
	}

	return true
}

func getMaxAge(policyDurationByIDs map[int]*time.Duration, now time.Time) *time.Time {
	var maxDuration *time.Duration
	for _, duration := range policyDurationByIDs {
		if duration == nil {
			// TODO - document
			return nil
		}
		if maxDuration == nil || *maxDuration < *duration {
			maxDuration = duration
		}
	}
	if maxDuration == nil {
		return nil
	}

	maxAge := now.Add(-*maxDuration)
	return &maxAge
}
