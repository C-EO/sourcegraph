package indexing

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/policies"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

type IndexScheduler struct {
	dbStore                DBStore
	gitserverClient        GitserverClient
	indexEnqueuer          IndexEnqueuer
	repositoryProcessDelay time.Duration
	repositoryBatchSize    int
	operations             *schedulerOperations
}

var (
	_ goroutine.Handler      = &IndexScheduler{}
	_ goroutine.ErrorHandler = &IndexScheduler{}
)

func NewIndexScheduler(
	dbStore DBStore,
	gitserverClient GitserverClient,
	indexEnqueuer IndexEnqueuer,
	repositoryProcessDelay time.Duration,
	repositoryBatchSize int,
	interval time.Duration,
	observationContext *observation.Context,
) goroutine.BackgroundRoutine {
	scheduler := &IndexScheduler{
		dbStore:                dbStore,
		gitserverClient:        gitserverClient,
		indexEnqueuer:          indexEnqueuer,
		repositoryProcessDelay: repositoryProcessDelay,
		repositoryBatchSize:    repositoryBatchSize,
		operations:             newOperations(observationContext),
	}

	return goroutine.NewPeriodicGoroutineWithMetrics(
		context.Background(),
		interval,
		scheduler,
		scheduler.operations.HandleIndexScheduler,
	)
}

// For mocking in tests
var autoIndexingEnabled = conf.CodeIntelAutoIndexingEnabled

func (s *IndexScheduler) Handle(ctx context.Context) error {
	if !autoIndexingEnabled() {
		return nil
	}

	// Get the batch of repositories that we'll handle in this invocation of the periodic goroutine. This
	// set should contain repositories that have yet to be updated, or that have been updated least recently.
	// This allows us to update every repository reliably, even if it takes a long time to process through
	// the backlog.
	repositories, err := s.dbStore.SelectRepositoriesForIndexScan(ctx, s.repositoryProcessDelay, s.repositoryBatchSize)
	if err != nil {
		return errors.Wrap(err, "dbstore.SelectRepositoriesForIndexScan")
	}
	if len(repositories) == 0 {
		// All repositories updated recently enough
		return nil
	}

	// Get shared policies that apply over all repositories
	globalPolicies, err := s.dbStore.GetConfigurationPolicies(ctx, dbstore.GetConfigurationPoliciesOptions{
		ForIndexing: true,
	})
	if err != nil {
		return errors.Wrap(err, "dbstore.GetConfigurationPolicies")
	}

	var queueErr error
	for _, repositoryID := range repositories {
		// Get repository-specific policies that apply only to this repository
		repositoryPolicies, err := s.dbStore.GetConfigurationPolicies(ctx, dbstore.GetConfigurationPoliciesOptions{
			RepositoryID: repositoryID,
			ForIndexing:  true,
		})
		if err != nil {
			return errors.Wrap(err, "dbstore.GetConfigurationPolicies")
		}

		combinedPolicies := append(globalPolicies, repositoryPolicies...)

		//
		// TODO - mock this out
		//

		// Determine the set of commits that should be reliably indexed for this repository
		matcher, err := policies.NewMatcher(s.gitserverClient, policies.IndexingExtractor, false, true)
		if err != nil {
			return errors.Wrap(err, "policies.NewMatcher")
		}
		commitMap, err := matcher.CommitsDescribedByPolicy(ctx, repositoryID, combinedPolicies, time.Now())
		if err != nil {
			return errors.Wrap(err, "policies.CommitsDescribedByPolicy")
		}

		for commit := range commitMap {
			if _, err := s.indexEnqueuer.QueueIndexes(ctx, repositoryID, commit, "", false); err != nil {
				if errors.HasType(err, &gitserver.RevisionNotFoundError{}) {
					continue
				}

				if queueErr == nil {
					queueErr = err
				} else {
					queueErr = multierror.Append(queueErr, err)
				}
			}
		}
	}
	if queueErr != nil {
		return queueErr
	}

	return nil
}

func (s *IndexScheduler) HandleError(err error) {
	log15.Error("Failed to schedule index jobs", "err", err)
}
