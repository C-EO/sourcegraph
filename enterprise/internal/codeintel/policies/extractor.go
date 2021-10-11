package policies

import (
	"time"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
)

// TODO - document
type Extractor func(policy dbstore.ConfigurationPolicy) (maxAge *time.Duration, includeIntermediateCommits bool)

func IndexingExtractor(policy dbstore.ConfigurationPolicy) (*time.Duration, bool) {
	return policy.IndexCommitMaxAge, policy.IndexIntermediateCommits
}

func RetentionExtractor(policy dbstore.ConfigurationPolicy) (*time.Duration, bool) {
	return policy.RetentionDuration, policy.RetainIntermediateCommits
}
