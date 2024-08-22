// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package dotcom

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

// SnippetAttributionResponse is returned by SnippetAttribution on success.
type SnippetAttributionResponse struct {
	// EXPERIMENTAL: Searches the instances indexed code for code matching snippet.
	SnippetAttribution SnippetAttributionSnippetAttributionSnippetAttributionConnection `json:"snippetAttribution"`
}

// GetSnippetAttribution returns SnippetAttributionResponse.SnippetAttribution, and is useful for accessing the field via an interface.
func (v *SnippetAttributionResponse) GetSnippetAttribution() SnippetAttributionSnippetAttributionSnippetAttributionConnection {
	return v.SnippetAttribution
}

// SnippetAttributionSnippetAttributionSnippetAttributionConnection includes the requested fields of the GraphQL type SnippetAttributionConnection.
// The GraphQL type's documentation follows.
//
// EXPERIMENTAL: A list of snippet attributions.
type SnippetAttributionSnippetAttributionSnippetAttributionConnection struct {
	// totalCount is the total number of repository attributions we found before
	// stopping the search.
	//
	// Note: if we didn't finish searching the full corpus then limitHit will be
	// true. For filtering use case this means if limitHit is true you need to be
	// conservative with TotalCount and assume it could be higher.
	TotalCount int `json:"totalCount"`
	// limitHit is true if we stopped searching before looking into the full
	// corpus. If limitHit is true then it is possible there are more than
	// totalCount attributions.
	LimitHit bool `json:"limitHit"`
	// The page set of SnippetAttribution entries in this connection.
	Nodes []SnippetAttributionSnippetAttributionSnippetAttributionConnectionNodesSnippetAttribution `json:"nodes"`
}

// GetTotalCount returns SnippetAttributionSnippetAttributionSnippetAttributionConnection.TotalCount, and is useful for accessing the field via an interface.
func (v *SnippetAttributionSnippetAttributionSnippetAttributionConnection) GetTotalCount() int {
	return v.TotalCount
}

// GetLimitHit returns SnippetAttributionSnippetAttributionSnippetAttributionConnection.LimitHit, and is useful for accessing the field via an interface.
func (v *SnippetAttributionSnippetAttributionSnippetAttributionConnection) GetLimitHit() bool {
	return v.LimitHit
}

// GetNodes returns SnippetAttributionSnippetAttributionSnippetAttributionConnection.Nodes, and is useful for accessing the field via an interface.
func (v *SnippetAttributionSnippetAttributionSnippetAttributionConnection) GetNodes() []SnippetAttributionSnippetAttributionSnippetAttributionConnectionNodesSnippetAttribution {
	return v.Nodes
}

// SnippetAttributionSnippetAttributionSnippetAttributionConnectionNodesSnippetAttribution includes the requested fields of the GraphQL type SnippetAttribution.
// The GraphQL type's documentation follows.
//
// EXPERIMENTAL: Attribution result from snippetAttribution.
type SnippetAttributionSnippetAttributionSnippetAttributionConnectionNodesSnippetAttribution struct {
	// The name of the repository containing the snippet.
	//
	// Note: we do not return a type Repository since repositoryName may
	// represent a repository not on this instance. eg a match from the
	// sourcegraph.com open source corpus.
	RepositoryName string `json:"repositoryName"`
}

// GetRepositoryName returns SnippetAttributionSnippetAttributionSnippetAttributionConnectionNodesSnippetAttribution.RepositoryName, and is useful for accessing the field via an interface.
func (v *SnippetAttributionSnippetAttributionSnippetAttributionConnectionNodesSnippetAttribution) GetRepositoryName() string {
	return v.RepositoryName
}

// __SnippetAttributionInput is used internally by genqlient
type __SnippetAttributionInput struct {
	Snippet string `json:"snippet"`
	First   int    `json:"first"`
}

// GetSnippet returns __SnippetAttributionInput.Snippet, and is useful for accessing the field via an interface.
func (v *__SnippetAttributionInput) GetSnippet() string { return v.Snippet }

// GetFirst returns __SnippetAttributionInput.First, and is useful for accessing the field via an interface.
func (v *__SnippetAttributionInput) GetFirst() int { return v.First }

// Searches the instances indexed code for code matching snippet.
func SnippetAttribution(
	ctx context.Context,
	client graphql.Client,
	snippet string,
	first int,
) (*SnippetAttributionResponse, error) {
	req := &graphql.Request{
		OpName: "SnippetAttribution",
		Query: `
query SnippetAttribution ($snippet: String!, $first: Int!) {
	snippetAttribution(snippet: $snippet, first: $first) {
		totalCount
		limitHit
		nodes {
			repositoryName
		}
	}
}
`,
		Variables: &__SnippetAttributionInput{
			Snippet: snippet,
			First:   first,
		},
	}
	var err error

	var data SnippetAttributionResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}
