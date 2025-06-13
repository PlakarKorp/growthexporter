package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/github"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"golang.org/x/oauth2"
)

func createRepoMetrics(repoLabel string) map[string]prometheus.Gauge {
	labels := make(map[string]prometheus.Gauge)

	labels["stars"] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "github_repo_stars_" + repoLabel,
	})
	labels["watchers"] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "github_repo_watchers_" + repoLabel,
	})
	labels["forks"] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "github_repo_forks_" + repoLabel,
	})
	labels["openIssues"] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "github_repo_open_issues_" + repoLabel,
	})
	labels["subscribers"] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "github_repo_subscribers_" + repoLabel,
	})
	labels["repoViews"] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "github_repo_views_" + repoLabel,
	})
	labels["repoClones"] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "github_repo_clones_" + repoLabel,
	})
	labels["pullRequestsOpen"] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "github_repo_pull_requests_open_" + repoLabel,
	})
	labels["contributorsTotal"] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "github_repo_contributors_total_" + repoLabel,
	})
	labels["lastCommitTimestamp"] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "github_repo_last_commit_timestamp_" + repoLabel,
	})

	for _, metric := range labels {
		prometheus.MustRegister(metric)
	}

	return labels
}

func fetchRepoStats(owner, repo string, metrics map[string]prometheus.Gauge, client *github.Client) {
	for {
		ctx := context.Background()
		repository, _, err := client.Repositories.Get(ctx, owner, repo)
		if err != nil {
			log.Printf("Error fetching GitHub data for %s/%s: %v", owner, repo, err)
			time.Sleep(1 * time.Minute)
			continue
		}

		var prNumber int
		if prs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{State: "open"}); err == nil {
			metrics["pullRequestsOpen"].Set(float64(len(prs)))
			prNumber = len(prs)
		}


		metrics["stars"].Set(float64(repository.GetStargazersCount()))
		metrics["watchers"].Set(float64(repository.GetWatchersCount()))
		metrics["forks"].Set(float64(repository.GetForksCount()))
		metrics["openIssues"].Set(float64(repository.GetOpenIssuesCount()) - float64(prNumber))
		metrics["subscribers"].Set(float64(repository.GetSubscribersCount()))

		if views, _, err := client.Repositories.ListTrafficViews(ctx, owner, repo, nil); err == nil {
			total := 0
			for _, v := range views.Views {
				if v.Count != nil {
					total += *v.Count
				}
			}
			metrics["repoViews"].Set(float64(total))
		}

		if clones, _, err := client.Repositories.ListTrafficClones(ctx, owner, repo, nil); err == nil {
			total := 0
			for _, c := range clones.Clones {
				if c.Count != nil {
					total += *c.Count
				}
			}
			metrics["repoClones"].Set(float64(total))
		}

		if contributors, _, err := client.Repositories.ListContributors(ctx, owner, repo, nil); err == nil {
			metrics["contributorsTotal"].Set(float64(len(contributors)))
		}

		branch := repository.GetDefaultBranch()
		if commits, _, err := client.Repositories.ListCommits(ctx, owner, repo, &github.CommitsListOptions{SHA: branch, ListOptions: github.ListOptions{PerPage: 1}}); err == nil && len(commits) > 0 {
			metrics["lastCommitTimestamp"].Set(float64(commits[0].Commit.Committer.GetDate().Unix()))
		}

		time.Sleep(1 * time.Minute)
	}
}

func main() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "<YOUR_GITHUB_ACCESS_TOKEN>"},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	repos := []struct {
		Owner string
		Name  string
	}{
		{"PlakarKorp", "plakar"},
		{"PlakarKorp", "kloset"},
	}

	for _, repo := range repos {
		label := strings.ToLower(strings.ReplaceAll(repo.Owner+"_"+repo.Name, "-", "_"))
		metrics := createRepoMetrics(label)
		go fetchRepoStats(repo.Owner, repo.Name, metrics, client)
	}

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Exporter listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
