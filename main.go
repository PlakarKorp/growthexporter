package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	CreatedAt time.Time
	Source    string
	Key       string
	Event     string
	Value     int64
}

func newEvent(ts time.Time, source, key, event string, value int64) Event {
	return Event{
		CreatedAt: ts,
		Source:    source,
		Key:       key,
		Event:     event,
		Value:     value,
	}
}

func dbEventsLayer(pool *pgxpool.Pool, eventsChannel chan Event) {
	for event := range eventsChannel {
		fmt.Printf("Event: %+v\n", event)
		_, err := pool.Exec(context.Background(),
			`INSERT INTO event_logs (ts, source, key, event, value) VALUES ($1, $2, $3, $4, $5)`,
			event.CreatedAt, event.Source, event.Key, event.Event, event.Value)
		if err != nil {
			log.Printf("Error inserting event into database: %v", err)
		}
	}
}

func dbNginxLayer(pool *pgxpool.Pool, nginxlogsChannel chan Event) {
	for event := range nginxlogsChannel {
		fmt.Printf("Log: %+v\n", event)
		/*_, err := pool.Exec(context.Background(),
			`INSERT INTO event_logs (ts, source, key, event, value) VALUES ($1, $2, $3, $4, $5)`,
			event.CreatedAt, event.Source, event.Key, event.Event, event.Value)
		if err != nil {
			log.Printf("Error inserting event into database: %v", err)
		}
		*/
	}
}

func main() {
	nginxDir := os.Getenv("NGINX_LOG_DIR")
	if nginxDir == "" {
		log.Println("NGINX_LOG_DIR environment variable is not set, exiting.")
		os.Exit(1)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Println("DATABASE_URL environment variable is not set, exiting.")
		os.Exit(1)
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Println("GITHUB_TOKEN environment variable is not set, exiting.")
		os.Exit(1)
	}

	discordToken := os.Getenv("DISCORD_TOKEN")
	if discordToken == "" {
		log.Println("DISCORD_TOKEN environment variable is not set, exiting.")
		os.Exit(1)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("set DATABASE_URL")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db pool: %v", err)
	}
	defer pool.Close()

	eventsChannel := make(chan Event, 100)
	_ = eventsChannel
	//go dbEventsLayer(pool, eventsChannel)
	//go fetchGithubStats(ctx, githubToken, eventsChannel, time.Hour)
	//go fetchDiscordStats(ctx, discordToken, eventsChannel, time.Hour)

	nginxLogDir, err := filepath.Abs(nginxDir)
	if err != nil {
		log.Fatalf("invalid NGINX_LOG_DIR: %v", err)
	}

	go trackNginxLogs(ctx, databaseURL, nginxLogDir, time.Minute)

	<-make(chan struct{})
}
