package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const discordAPIBase = "https://discord.com/api/v10"

type guildCounts struct {
	ID                       string `json:"id"`
	Name                     string `json:"name"`
	ApproximateMemberCount   int64  `json:"approximate_member_count"`
	ApproximatePresenceCount int64  `json:"approximate_presence_count"`
}

type discordChannel struct {
	ID   string `json:"id"`
	Type int    `json:"type"` // Discord channel type enum
}

type discordRole struct {
	ID string `json:"id"`
}

func fetchDiscordStats(ctx context.Context, token string, eventsChannel chan Event, interval time.Duration) {

	client := &http.Client{Timeout: 10 * time.Second}
	auth := "Bot " + token

	guildIDs := []string{
		"1281181647005421589", // Plakar guild ID(s)
	}

	for {
		for _, gid := range guildIDs {
			subject := gid // we’ll also append name when we have it

			// 1) Guild with approximate counts
			var g guildCounts
			if err := getJSON(ctx, client, auth, discordAPIBase+"/guilds/"+gid+"?with_counts=true", &g); err != nil {
				log.Printf("discord: guild %s: %v", gid, err)
				continue
			}
			if g.Name != "" {
				subject = fmt.Sprintf("%s/%s", g.Name, gid)
			}

			// 2) Channels
			var chs []discordChannel
			_ = getJSON(ctx, client, auth, discordAPIBase+"/guilds/"+gid+"/channels", &chs) // best effort

			// classify channel types (see Discord docs)
			var total, text, voice, thread, category int64
			for _, c := range chs {
				total++
				switch c.Type {
				case 0, 5, 15, 16: // text/news/forum/media
					text++
				case 2, 13: // voice/stage
					voice++
				case 10, 11, 12: // threads
					thread++
				case 4: // category
					category++
				}
			}

			// 3) Roles
			var roles []discordRole
			_ = getJSON(ctx, client, auth, discordAPIBase+"/guilds/"+gid+"/roles", &roles)
			roleCount := int64(len(roles))

			// Emit events (mirrors your GitHub style)
			source := "discord"
			now := time.Now().UTC()
			events := []Event{
				newEvent(now, source, subject, "members.approximate", g.ApproximateMemberCount),
				newEvent(now, source, subject, "presence.approximate", g.ApproximatePresenceCount),
				newEvent(now, source, subject, "channels.count", total),
				newEvent(now, source, subject, "channels.text.count", text),
				newEvent(now, source, subject, "channels.voice.count", voice),
				newEvent(now, source, subject, "channels.thread.count", thread),
				newEvent(now, source, subject, "channels.category.count", category),
				newEvent(now, source, subject, "roles.count", roleCount),
			}
			for _, e := range events {
				eventsChannel <- e
			}
		}

		time.Sleep(interval)
	}
}
func getJSON(ctx context.Context, client *http.Client, auth, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("User-Agent", "discord-stats-ingester/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		resetAfter := resp.Header.Get("X-RateLimit-Reset-After")
		return fmt.Errorf("rate limited; reset-after=%s url=%s", resetAfter, url)
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("forbidden (bot lacks permission) for %s", url)
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found: %s", url)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord api %s: %s", url, resp.Status)
	}

	dec := json.NewDecoder(resp.Body)
	// dec.DisallowUnknownFields()  // <-- remove this
	return dec.Decode(out)
}
