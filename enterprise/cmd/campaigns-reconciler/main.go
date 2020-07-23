package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/testing"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
)

func main() {
	ctx := context.Background()
	clock := func() time.Time { return time.Now().UTC() }

	dsn := dbutil.PostgresDSN("sourcegraph", os.Getenv)
	db, err := dbutil.NewDB(dsn, "campaigns-reconciler")
	if err != nil {
		log.Fatalf("failed to initialize db store: %v", err)
	}

	if _, err := db.ExecContext(ctx, "DELETE FROM changesets;"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM changeset_specs;"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM campaign_specs;"); err != nil {
		log.Fatal(err)
	}

	campaignsStore := ee.NewStore(db)

	spec, err := campaigns.NewCampaignSpecFromRaw(ct.TestRawCampaignSpec)
	if err != nil {
		log.Fatal(err)
	}
	spec.UserID = 1
	spec.NamespaceUserID = 1
	if err := campaignsStore.CreateCampaignSpec(ctx, spec); err != nil {
		log.Fatal(err)
	}

	specs := make([]*campaigns.ChangesetSpec, 0, 3)
	for i := 0; i < cap(specs); i++ {
		repoID := graphqlbackend.MarshalRepositoryID(1)
		s, err := campaigns.NewChangesetSpecFromRaw(ct.NewRawChangesetSpecGitBranch(repoID, "deadb33f"))
		if err != nil {
			log.Fatal(err)
		}
		s.CampaignSpecID = spec.ID
		s.UserID = 1
		s.RepoID = api.RepoID(1)

		if err := campaignsStore.CreateChangesetSpec(ctx, s); err != nil {
			log.Fatalf("failed to create changeset spec: %s", err)
		}
		specs = append(specs, s)
	}

	changesets := make(campaigns.Changesets, 0, 3)
	for i := 0; i < cap(specs); i++ {
		spec := specs[i]
		th := &campaigns.Changeset{
			RepoID:              spec.RepoID,
			Metadata:            newGitHubPR(),
			CampaignIDs:         []int64{int64(i) + 1},
			ExternalID:          fmt.Sprintf("%d", i),
			ExternalServiceType: extsvc.TypeGitHub,
			ExternalBranch:      "campaigns/test",
			ExternalUpdatedAt:   clock(),
			ExternalState:       campaigns.ChangesetExternalStateOpen,
			ExternalReviewState: campaigns.ChangesetReviewStateApproved,
			ExternalCheckState:  campaigns.ChangesetCheckStatePassed,

			ChangesetSpecID: spec.ID,
			State:           campaigns.ChangesetStateUnpublished,

			WorkerState: "queued",
		}

		changesets = append(changesets, th)
	}

	err = campaignsStore.CreateChangesets(ctx, changesets...)
	if err != nil {
		log.Fatalf("failed to create changesets: %s", err)
	}
}

func newGitHubPR() *github.PullRequest {
	githubActor := github.Actor{
		AvatarURL: "https://avatars2.githubusercontent.com/u/1185253",
		Login:     "mrnugget",
		URL:       "https://github.com/mrnugget",
	}

	return &github.PullRequest{
		ID:           "FOOBARID",
		Title:        "Fix a bunch of bugs",
		Body:         "This fixes a bunch of bugs",
		URL:          "https://github.com/sourcegraph/sourcegraph/pull/12345",
		Number:       12345,
		Author:       githubActor,
		Participants: []github.Actor{githubActor},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		HeadRefName:  "campaigns/test",
	}
}
