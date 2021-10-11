package resolvers

import (
	"context"
	"sync"

	"github.com/graph-gophers/graphql-go/relay"

	"github.com/cockroachdb/errors"

	"github.com/graph-gophers/graphql-go"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/types"

	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/store"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
)

var _ graphqlbackend.InsightsDashboardConnectionResolver = &dashboardConnectionResolver{}
var _ graphqlbackend.InsightsDashboardResolver = &insightDashboardResolver{}
var _ graphqlbackend.InsightViewConnectionResolver = &stubDashboardInsightViewConnectionResolver{}
var _ graphqlbackend.InsightViewResolver = &stubInsightViewResolver{}
var _ graphqlbackend.InsightDashboardPayloadResolver = &insightsDashboardPayloadResolver{}

type dashboardConnectionResolver struct {
	insightsDatabase dbutil.DB
	dashboardStore   store.DashboardStore
	args             *graphqlbackend.InsightDashboardsArgs

	// Cache results because they are used by multiple fields
	once       sync.Once
	dashboards []*types.Dashboard
	next       int64
	err        error
}

func (d *dashboardConnectionResolver) compute(ctx context.Context) ([]*types.Dashboard, int64, error) {
	d.once.Do(func() {
		args := store.DashboardQueryArgs{}
		if d.args.After != nil {
			afterID, err := unmarshalDashboardID(graphql.ID(*d.args.After))
			if err != nil {
				d.err = errors.Wrap(err, "unmarshalID")
				return
			}
			args.After = int(afterID.Arg)
		}
		if d.args.First != nil {
			args.Limit = int(*d.args.First)
		}
		dashboards, err := d.dashboardStore.GetDashboards(ctx, args)
		if err != nil {
			d.err = err
			return
		}
		d.dashboards = dashboards
		for _, dashboard := range dashboards {
			if int64(dashboard.ID) > d.next {
				d.next = int64(dashboard.ID)
			}
		}
	})
	return d.dashboards, d.next, d.err
}

func (d *dashboardConnectionResolver) Nodes(ctx context.Context) ([]graphqlbackend.InsightsDashboardResolver, error) {
	dashboards, _, err := d.compute(ctx)
	if err != nil {
		return nil, err
	}
	resolvers := make([]graphqlbackend.InsightsDashboardResolver, 0, len(dashboards))
	for _, dashboard := range dashboards {
		id := newRealDashboardID(int64(dashboard.ID))
		resolvers = append(resolvers, &insightDashboardResolver{dashboard: dashboard, id: &id})
	}
	return resolvers, nil
}

func (d *dashboardConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	_, _, err := d.compute(ctx)
	if err != nil {
		return nil, err
	}
	if d.next != 0 {
		return graphqlutil.NextPageCursor(string(newRealDashboardID(d.next).marshal())), nil
	}
	return graphqlutil.HasNextPage(false), nil
}

type insightDashboardResolver struct {
	dashboard *types.Dashboard
	id        *dashboardID
}

func (i *insightDashboardResolver) Title() string {
	return i.dashboard.Title
}

func (i *insightDashboardResolver) ID() graphql.ID {
	return i.id.marshal()
}

func (i *insightDashboardResolver) Views() graphqlbackend.InsightViewConnectionResolver {
	return &stubDashboardInsightViewConnectionResolver{ids: i.dashboard.InsightIDs}
}

type stubDashboardInsightViewConnectionResolver struct {
	ids []string
}

func (d *stubDashboardInsightViewConnectionResolver) Nodes(ctx context.Context) ([]graphqlbackend.InsightViewResolver, error) {
	resolvers := make([]graphqlbackend.InsightViewResolver, 0, len(d.ids))
	for _, id := range d.ids {
		resolvers = append(resolvers, &stubInsightViewResolver{id: id})
	}
	return resolvers, nil
}

func (d *stubDashboardInsightViewConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	return graphqlutil.HasNextPage(false), nil
}

func (r *Resolver) DeleteInsightsDashboard(ctx context.Context, args *graphqlbackend.DeleteInsightsDashboardArgs) (*graphqlbackend.EmptyResponse, error) {
	emptyResponse := &graphqlbackend.EmptyResponse{}

	dashboardID, err := unmarshalDashboardID(args.Id)
	if err != nil {
		return emptyResponse, err
	}
	if dashboardID.isVirtualized() {
		return emptyResponse, nil
	}

	err = r.dashboardStore.DeleteDashboard(ctx, dashboardID.Arg)
	if err != nil {
		return emptyResponse, err
	}
	return emptyResponse, nil
}

type stubInsightViewResolver struct {
	id string
}

func (s *stubInsightViewResolver) ID() graphql.ID {
	return relay.MarshalID("insight_view", s.id)
}

func (s *stubInsightViewResolver) VeryUniqueResolver() bool {
	return true
}

func (r *Resolver) AddInsightViewToDashboard(ctx context.Context, args *graphqlbackend.AddInsightViewToDashboardArgs) (graphqlbackend.InsightDashboardPayloadResolver, error) {
	var viewID string
	err := relay.UnmarshalSpec(args.Input.InsightViewID, &viewID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal insight view id")
	}
	dashboardID, err := unmarshalDashboardID(args.Input.DashboardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal dashboard id")
	}

	err = r.dashboardStore.AssociateViewsByViewIds(ctx, int(dashboardID.Arg), []string{viewID})
	if err != nil {
		return nil, errors.Wrap(err, "AddInsightViewToDashboard")
	}
	dashboards, err := r.dashboardStore.GetDashboards(ctx, store.DashboardQueryArgs{ID: int(dashboardID.Arg)})
	if err != nil || len(dashboards) < 1 {
		return nil, errors.Wrap(err, "GetDashboards")
	}
	return &insightsDashboardPayloadResolver{dashboard: dashboards[0]}, nil
}

func (r *Resolver) RemoveInsightViewFromDashboard(ctx context.Context, args *graphqlbackend.RemoveInsightViewFromDashboardArgs) (graphqlbackend.InsightDashboardPayloadResolver, error) {
	var viewID string
	err := relay.UnmarshalSpec(args.Input.InsightViewID, &viewID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal insight view id")
	}
	dashboardID, err := unmarshalDashboardID(args.Input.DashboardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal dashboard id")
	}

	err = r.dashboardStore.RemoveViewsFromDashboard(ctx, int(dashboardID.Arg), []string{viewID})
	if err != nil {
		return nil, errors.Wrap(err, "RemoveViewsFromDashboard")
	}
	dashboards, err := r.dashboardStore.GetDashboards(ctx, store.DashboardQueryArgs{ID: int(dashboardID.Arg)})
	if err != nil || len(dashboards) < 1 {
		return nil, errors.Wrap(err, "GetDashboards")
	}
	return &insightsDashboardPayloadResolver{dashboard: dashboards[0]}, nil
}

type insightsDashboardPayloadResolver struct {
	dashboard *types.Dashboard
}

func (i *insightsDashboardPayloadResolver) Dashboard(ctx context.Context) (graphqlbackend.InsightsDashboardResolver, error) {
	id := newRealDashboardID(int64(i.dashboard.ID))
	return &insightDashboardResolver{dashboard: i.dashboard, id: &id}, nil
}
