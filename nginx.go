package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	clog "github.com/PlakarKorp/go-cursorlog"
	"github.com/jackc/pgx/v5/pgxpool"
)

func listFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Stop walking if we can’t access something
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func trackNginxLogs(ctx context.Context, dsn string, directory string, interval time.Duration) {
	clog, err := clog.NewCursorLog(directory + "/cursor_state.json")
	if err != nil {
		fmt.Println("Error creating cursor log:", err)
		return
	}
	defer clog.Close()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	for {
		files, err := listFiles(directory)
		if err != nil {
			fmt.Println("Error listing files:", err)
		} else {
			wg := sync.WaitGroup{}
			wg.Add(len(files))
			for _, file := range files {
				go func(file string) {
					defer wg.Done()

					rd, err := clog.Open(file)
					if err != nil {
						fmt.Println("Error opening file:", err)
						return
					}
					defer rd.Close()

					processLogBatch(ctx, pool, file, rd)
				}(file)
			}
			wg.Wait()
			clog.Save()
		}
		// locate all files below /nginx/logs
		fmt.Println("Sleeping for", interval)
		time.Sleep(interval)
	}
}

func processLogBatch(ctx context.Context, pool *pgxpool.Pool, name string, rd io.Reader) {
	const (
		maxLineBytes = 4 << 20 // 4 MiB per log line
		batchSize    = 1000    // tune: 500–2000 works well
	)

	sc := bufio.NewScanner(rd)
	buf := make([]byte, 0, 64<<10)
	sc.Buffer(buf, maxLineBytes)

	q := `
		INSERT INTO nginx_logs (raw)
		SELECT x::jsonb
		FROM unnest($1::text[]) AS t(x)
		ON CONFLICT (fp) DO NOTHING;
	`

	batch := make([]string, 0, batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx2, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		tag, err := pool.Exec(ctx2, q, batch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: insert batch (%d) error: %v\n", name, len(batch), err)
		} else {
			fmt.Fprintf(os.Stderr, "%s: inserted=%d (batch=%d)\n", name, tag.RowsAffected(), len(batch))
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			// best-effort final flush on cancel
			flush()
			return
		default:
			// proceed to scan
		}

		if sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || !json.Valid([]byte(line)) {
				// skip silently or log if you prefer
				continue
			}
			batch = append(batch, line)

			// SIZE-BASED FLUSH ONLY
			if len(batch) >= batchSize {
				flush()
			}
			continue
		}

		// scanner ended this read loop
		if err := sc.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "scanner error: %v\n", err)
		}
		// final flush at EOF
		flush()
		return
	}
}
