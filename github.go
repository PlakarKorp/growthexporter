package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/go-github/github"

	"golang.org/x/oauth2"
)

func fetchGithubStats(ctx context.Context, token string, eventsChannel chan Event, interval time.Duration) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	repos := []struct {
		Owner string
		Name  string
	}{
		{"PlakarKorp", "plakar"},
		{"PlakarKorp", "kloset"},
		{"PlakarKorp", "kapsul"},
		{"PlakarKorp", "go-cdc-chunkers"},
		{"PlakarKorp", "go-kloset-sdk"},
		{"PlakarKorp", "integration-notion"},
		{"PlakarKorp", "integration-imap"},
		{"PlakarKorp", "integration-rclone"},
	}

	for {
		events := []Event{}
		source := "github"
		for _, repo := range repos {
			owner := repo.Owner
			repo := repo.Name
			now := time.Now().UTC()
			repository, _, err := client.Repositories.Get(ctx, owner, repo)
			if err != nil {
				log.Printf("Error fetching GitHub data for %s/%s: %v", owner, repo, err)
				time.Sleep(5 * time.Minute)
				continue
			}

			if prs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{State: "open"}); err == nil {
				events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "pr.open", int64(len(prs))))
			}

			events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "stars.count", int64(repository.GetStargazersCount())))
			events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "watchers.count", int64(repository.GetWatchersCount())))
			events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "forks.count", int64(repository.GetForksCount())))
			events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "issues.open.count", int64(repository.GetOpenIssuesCount())))
			events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "subscribers.count", int64(repository.GetSubscribersCount())))

			if views, _, err := client.Repositories.ListTrafficViews(ctx, owner, repo, nil); err == nil {
				events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "views.global.count", int64(views.GetCount())))
				events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "views.global.uniques", int64(views.GetUniques())))
				for _, point := range views.Views {
					events = append(events, newEvent(point.Timestamp.Time.UTC(), source, fmt.Sprintf("%s/%s", owner, repo), "views.point.count", int64(point.GetCount())))
					events = append(events, newEvent(point.Timestamp.Time.UTC(), source, fmt.Sprintf("%s/%s", owner, repo), "views.point.uniques", int64(point.GetUniques())))
				}
			}

			if referers, _, err := client.Repositories.ListTrafficReferrers(ctx, owner, repo); err == nil {
				for _, ref := range referers {
					events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s/referrer/%s", owner, repo, ref.GetReferrer()), "referrer.count", int64(ref.GetCount())))
					events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s/referrer/%s", owner, repo, ref.GetReferrer()), "referrer.uniques", int64(ref.GetUniques())))
				}
			}

			if clones, _, err := client.Repositories.ListTrafficClones(ctx, owner, repo, nil); err == nil {
				events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "clones.global.count", int64(clones.GetCount())))
				events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "clones.global.uniques", int64(clones.GetUniques())))
				for _, point := range clones.Clones {
					events = append(events, newEvent(point.Timestamp.Time.UTC(), source, fmt.Sprintf("%s/%s", owner, repo), "views.point.count", int64(point.GetCount())))
					events = append(events, newEvent(point.Timestamp.Time.UTC(), source, fmt.Sprintf("%s/%s", owner, repo), "views.point.uniques", int64(point.GetUniques())))
				}
			}

			if pop, _, err := client.Repositories.ListTrafficPaths(ctx, owner, repo); err == nil {
				for _, path := range pop {
					events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s/path/%s", owner, repo, path.GetPath()), "path.count", int64(path.GetCount())))
					events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s/path/%s", owner, repo, path.GetPath()), "path.uniques", int64(path.GetUniques())))
				}
			}

			if contributors, _, err := client.Repositories.ListContributors(ctx, owner, repo, nil); err == nil {
				events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "contributors.count", int64(len(contributors))))
			}

			branch := repository.GetDefaultBranch()
			if commits, _, err := client.Repositories.ListCommits(ctx, owner, repo, &github.CommitsListOptions{SHA: branch, ListOptions: github.ListOptions{PerPage: 1}}); err == nil && len(commits) > 0 {
				events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "commits.count", int64(len(commits))))
				events = append(events, newEvent(now, source, fmt.Sprintf("%s/%s", owner, repo), "commits.last.timestamp", commits[0].Commit.Committer.GetDate().Unix()))
			}

			for _, event := range events {
				eventsChannel <- event
			}
		}
		time.Sleep(interval)
	}
}
