package campaigns

import (
	"context"
	"database/sql"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/keegancsmith/sqlf"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

// maxWorkers defines the maximum number of changeset jobs to run in parallel.
var maxWorkers = env.Get("CAMPAIGNS_MAX_WORKERS", "8", "maximum number of repository jobs to run in parallel")

const defaultWorkerCount = 8

type GitserverClient interface {
	CreateCommitFromPatch(ctx context.Context, req protocol.CreateCommitFromPatchRequest) (string, error)
}

// RunWorkers should be executed in a background goroutine and is responsible
// for finding pending ChangesetJobs and executing them.
// ctx should be canceled to terminate the function.
func RunWorkers(ctx context.Context, s *Store, clock func() time.Time, gitClient GitserverClient, sourcer repos.Sourcer, backoffDuration time.Duration) {
	log15.Warn("RunWorkers")
	processor := &processor{
		gitserverClient: gitClient,
		sourcer:         sourcer,
	}

	handler := workerutil.HandlerFunc(func(ctx context.Context, tx workerutil.Store, record workerutil.Record) error {
		handle := tx.Handle()
		store := NewStore(handle.DB())
		return processor.Process(ctx, store, record.(*campaigns.Changeset))
	})

	options := workerutil.WorkerOptions{
		Handler:     handler,
		NumHandlers: 1,
		Interval:    backoffDuration,

		Metrics: workerutil.WorkerMetrics{HandleOperation: newObservationOperation()},
	}

	workerStore := workerutil.NewStore(s.Handle(), workerutil.StoreOptions{
		TableName: "changesets",
		ViewName:  "changesets_with_changeset_spec_spec c",
		AlternateColumnNames: map[string]string{
			// TODO: can we set the `state` field (not `worker_state`) to
			// `publishing` when an entry is fetched?
			"state": "worker_state",
		},
		ColumnExpressions: []*sqlf.Query{
			sqlf.Sprintf("c.id"),
			sqlf.Sprintf("c.repo_id"),
			sqlf.Sprintf("c.created_at"),
			sqlf.Sprintf("c.updated_at"),
			sqlf.Sprintf("c.metadata"),
			sqlf.Sprintf("c.campaign_ids"),
			sqlf.Sprintf("c.external_id"),
			sqlf.Sprintf("c.external_service_type"),
			sqlf.Sprintf("c.external_branch"),
			sqlf.Sprintf("c.external_deleted_at"),
			sqlf.Sprintf("c.external_updated_at"),
			sqlf.Sprintf("c.external_state"),
			sqlf.Sprintf("c.external_review_state"),
			sqlf.Sprintf("c.external_check_state"),
			sqlf.Sprintf("c.created_by_campaign"),
			sqlf.Sprintf("c.added_to_campaign"),
			sqlf.Sprintf("c.diff_stat_added"),
			sqlf.Sprintf("c.diff_stat_changed"),
			sqlf.Sprintf("c.diff_stat_deleted"),
			sqlf.Sprintf("c.sync_state"),

			sqlf.Sprintf("c.changeset_spec_id"),
			sqlf.Sprintf("c.state"),
			sqlf.Sprintf("c.worker_state"),
			sqlf.Sprintf("c.failure_message"),
			sqlf.Sprintf("c.started_at"),
			sqlf.Sprintf("c.finished_at"),
			sqlf.Sprintf("c.process_after"),
			sqlf.Sprintf("c.num_resets"),

			// TODO: c.spec
		},
		Scan:              scanFirstChangesetRecord,
		OrderByExpression: sqlf.Sprintf("c.updated_at"),
		StalledMaxAge:     5 * time.Second, // TODO(mrnugget)
		MaxNumResets:      5,               // TODO(mrnugget)
	})

	worker := workerutil.NewWorker(ctx, workerStore, options)
	worker.Start()
}

type Processor interface {
	Process(ctx context.Context, ch *campaigns.Changeset) error
}

type processor struct {
	gitserverClient GitserverClient
	sourcer         repos.Sourcer
}

func (p *processor) Process(ctx context.Context, store *Store, ch *campaigns.Changeset) error {
	log15.Warn("Processing changeset", "changeset", ch.ID)
	time.Sleep(5 * time.Second)
	ch.WorkerState = "done"
	log15.Warn("Processing done. Updating!", "changeset", ch.ID)
	return store.UpdateChangeset(ctx, ch)
}

func scanFirstChangesetRecord(rows *sql.Rows, err error) (workerutil.Record, bool, error) {
	return scanFirstChangeset(rows, err)
}

func scanFirstChangeset(rows *sql.Rows, err error) (*campaigns.Changeset, bool, error) {
	changesets, err := scanChangesets(rows, err)
	if err != nil || len(changesets) == 0 {
		return &campaigns.Changeset{}, false, err
	}
	return changesets[0], true, nil
}

func scanChangesets(rows *sql.Rows, queryErr error) (_ []*campaigns.Changeset, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = closeRows(rows, err) }()

	var changesets []*campaigns.Changeset
	for rows.Next() {
		var c campaigns.Changeset

		if err := scanChangeset(&c, rows); err != nil {
			return nil, err
		}

		changesets = append(changesets, &c)
	}

	return changesets, nil
}

func closeRows(rows *sql.Rows, err error) error {
	return basestore.CloseRows(rows, err)
}

func newObservationOperation() *observation.Operation {
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	metrics := metrics.NewOperationMetrics(
		observationContext.Registerer,
		"reconciler_todo",
		metrics.WithLabels("op"),
		metrics.WithCountHelp("Total number of results returned"),
	)

	return observationContext.Operation(observation.Op{
		Name:         "ReconcilerTODO.Process",
		MetricLabels: []string{"process"},
		Metrics:      metrics,
	})
}
