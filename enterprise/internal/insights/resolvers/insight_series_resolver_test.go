package resolvers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hexops/autogold"
	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/log/logtest"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	edb "github.com/sourcegraph/sourcegraph/enterprise/internal/database"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/background/queryrunner"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/scheduler"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/store"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

// TestResolver_InsightSeries tests that the InsightSeries GraphQL resolver works.
func TestResolver_InsightSeries(t *testing.T) {
	testSetup := func(t *testing.T) (context.Context, [][]graphqlbackend.InsightSeriesResolver) {
		// Setup the GraphQL resolver.
		ctx := actor.WithInternalActor(context.Background())
		now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
		logger := logtest.Scoped(t)
		clock := func() time.Time { return now }
		insightsDB := edb.NewInsightsDB(dbtest.NewInsightsDB(logger, t))
		postgres := database.NewDB(logger, dbtest.NewDB(logger, t))
		resolver := newWithClock(insightsDB, postgres, clock)
		insightStore := store.NewInsightStore(insightsDB)

		view := types.InsightView{
			Title:            "title1",
			Description:      "desc1",
			PresentationType: types.Line,
		}
		insightSeries := types.InsightSeries{
			SeriesID:            "1234567",
			Query:               "query1",
			CreatedAt:           now,
			OldestHistoricalAt:  now,
			LastRecordedAt:      now,
			NextRecordingAfter:  now,
			SampleIntervalUnit:  string(types.Month),
			SampleIntervalValue: 1,
		}
		var err error
		view, err = insightStore.CreateView(ctx, view, []store.InsightViewGrant{store.GlobalGrant()})
		require.NoError(t, err)
		insightSeries, err = insightStore.CreateSeries(ctx, insightSeries)
		require.NoError(t, err)
		insightStore.AttachSeriesToView(ctx, insightSeries, view, types.InsightViewSeriesMetadata{
			Label:  "",
			Stroke: "",
		})

		insightMetadataStore := store.NewMockInsightMetadataStore()

		resolver.insightMetadataStore = insightMetadataStore

		// Create the insightview connection resolver and query series.
		conn, err := resolver.InsightViews(ctx, &graphqlbackend.InsightViewQueryArgs{})
		if err != nil {
			t.Fatal(err)
		}

		nodes, err := conn.Nodes(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var series [][]graphqlbackend.InsightSeriesResolver
		for _, node := range nodes {
			s, _ := node.DataSeries(ctx)
			series = append(series, s)
		}
		return ctx, series
	}

	t.Run("Points", func(t *testing.T) {
		ctx, insights := testSetup(t)
		autogold.Want("insights length", int(1)).Equal(t, len(insights))

		autogold.Want("insights[0].length", int(1)).Equal(t, len(insights[0]))

		// Issue a query against the actual DB.
		points, err := insights[0][0].Points(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		autogold.Want("insights[0][0].Points", []graphqlbackend.InsightsDataPointResolver{}).Equal(t, points)

	})
}

func fakeStatusGetter(status *queryrunner.JobsStatus, err error) GetSeriesQueueStatusFunc {
	return func(ctx context.Context, seriesID string) (*queryrunner.JobsStatus, error) {
		return status, err
	}
}

func fakeBackfillGetter(backfills []scheduler.SeriesBackfill, err error) GetSeriesBackfillsFunc {
	return func(ctx context.Context, seriesID int) ([]scheduler.SeriesBackfill, error) {
		return backfills, err
	}
}
func fakeIncompleteGetter() GetIncompleteDatapointsFunc {
	return func(ctx context.Context, seriesID int) ([]store.IncompleteDatapoint, error) {
		return nil, nil
	}
}

func TestInsightSeriesStatusResolver_IsLoadingData(t *testing.T) {

	type isLoadingTestCase struct {
		backfills    []scheduler.SeriesBackfill
		backfillsErr error
		queueStatus  queryrunner.JobsStatus
		queueErr     error
		series       types.InsightViewSeries
		want         autogold.Value
	}

	recentTime := time.Date(2020, time.April, 1, 1, 0, 0, 0, time.UTC)

	cases := []isLoadingTestCase{
		{
			backfills: []scheduler.SeriesBackfill{{State: scheduler.BackfillStateCompleted}},
			series:    types.InsightViewSeries{BackfillQueuedAt: &recentTime},
			want:      autogold.Want("completed backfillv2", "loading:false error:"),
		},
		{
			backfills: []scheduler.SeriesBackfill{},
			series:    types.InsightViewSeries{BackfillQueuedAt: &recentTime},
			want:      autogold.Want("completed backfillv1", "loading:false error:"),
		},
		{
			backfills: []scheduler.SeriesBackfill{{State: scheduler.BackfillStateNew}},
			series:    types.InsightViewSeries{BackfillQueuedAt: &recentTime},
			want:      autogold.Want("new backfillv2", "loading:true error:"),
		},
		{
			backfills: []scheduler.SeriesBackfill{{State: scheduler.BackfillStateProcessing}},
			series:    types.InsightViewSeries{BackfillQueuedAt: &recentTime},
			want:      autogold.Want("in process backfillv2", "loading:true error:"),
		},
		{
			backfills: []scheduler.SeriesBackfill{},
			queueStatus: queryrunner.JobsStatus{
				Queued:     10,
				Processing: 2,
				Errored:    1,
			},
			series: types.InsightViewSeries{BackfillQueuedAt: &recentTime},
			want:   autogold.Want("in progress backfillv1", "loading:true error:"),
		},
		{
			backfills: []scheduler.SeriesBackfill{{State: scheduler.BackfillStateFailed}},
			series:    types.InsightViewSeries{BackfillQueuedAt: &recentTime},
			want:      autogold.Want("failed backfillv2", "loading:false error:"),
		},
		{
			backfills: []scheduler.SeriesBackfill{},
			queueStatus: queryrunner.JobsStatus{
				Failed: 10,
			},
			series: types.InsightViewSeries{BackfillQueuedAt: &recentTime},
			want:   autogold.Want("failed backfillv1", "loading:false error:"),
		},
		{
			backfills: []scheduler.SeriesBackfill{{State: scheduler.BackfillStateCompleted}},
			queueStatus: queryrunner.JobsStatus{
				Queued: 1,
			},
			series: types.InsightViewSeries{BackfillQueuedAt: &recentTime},
			want:   autogold.Want("completed but snaphotting backfillv2", "loading:true error:"),
		},
		{
			backfills:    []scheduler.SeriesBackfill{},
			backfillsErr: errors.New("backfill error"),
			series:       types.InsightViewSeries{BackfillQueuedAt: &recentTime},
			want:         autogold.Want("error loading backfill", "loading:false error:LoadSeriesBackfills: backfill error"),
		},
		{
			backfills: []scheduler.SeriesBackfill{},
			queueErr:  errors.New("error loading queue status"),
			series:    types.InsightViewSeries{BackfillQueuedAt: &recentTime},
			want:      autogold.Want("error loading queue status", "loading:false error:QueryJobsStatus: error loading queue status"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.want.Name(), func(t *testing.T) {
			statusGetter := fakeStatusGetter(&tc.queueStatus, tc.queueErr)
			backfillGetter := fakeBackfillGetter(tc.backfills, tc.backfillsErr)
			statusResolver := newStatusResolver(statusGetter, backfillGetter, fakeIncompleteGetter(), tc.series)
			loading, err := statusResolver.IsLoadingData(context.Background())
			var loadingResult bool
			if loading != nil {
				loadingResult = *loading
			}
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}

			tc.want.Equal(t, fmt.Sprintf("loading:%t error:%s", loadingResult, errMsg))
		})
	}

}

func TestInsightStatusResolver_IncompleteDatapoints(t *testing.T) {
	// Setup the GraphQL resolver.
	ctx := actor.WithInternalActor(context.Background())
	now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	logger := logtest.Scoped(t)
	insightsDB := edb.NewInsightsDB(dbtest.NewInsightsDB(logger, t))
	postgres := database.NewDB(logger, dbtest.NewDB(logger, t))
	insightStore := store.NewInsightStore(insightsDB)
	tss := store.New(insightsDB, store.NewInsightPermissionStore(postgres))

	base := baseInsightResolver{
		insightStore:    insightStore,
		timeSeriesStore: tss,
		insightsDB:      insightsDB,
		postgresDB:      postgres,
	}

	series, err := insightStore.CreateSeries(ctx, types.InsightSeries{
		SeriesID:            "asdf",
		Query:               "asdf",
		SampleIntervalUnit:  string(types.Month),
		SampleIntervalValue: 1,
		GenerationMethod:    types.Search,
	})
	require.NoError(t, err)

	repo := 5
	addFakeIncomplete := func(in time.Time) {
		err = tss.AddIncompleteDatapoint(ctx, store.AddIncompleteDatapointInput{
			SeriesID: series.ID,
			RepoID:   &repo,
			Reason:   store.ReasonTimeout,
			Time:     in,
		})
		require.NoError(t, err)
	}

	resolver := NewStatusResolver(&base, types.InsightViewSeries{InsightSeriesID: series.ID})

	addFakeIncomplete(now)
	addFakeIncomplete(now)
	addFakeIncomplete(now.AddDate(0, 0, 1))

	stringify := func(input []graphqlbackend.IncompleteDatapointAlert) (res []string) {
		for _, in := range input {
			res = append(res, in.Time().String())
		}
		return res
	}

	t.Run("as timeout", func(t *testing.T) {
		got, err := resolver.IncompleteDatapoints(ctx)
		require.NoError(t, err)
		autogold.Want("as timeout", []string{"2020-01-01 00:00:00 +0000 UTC", "2020-01-02 00:00:00 +0000 UTC"}).Equal(t, stringify(got))
	})

}
