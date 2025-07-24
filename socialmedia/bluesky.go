package socialmedia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type SessionResponse struct {
	AccessJwt string `json:"accessJwt"`
}

type ProfileResponse struct {
	Handle         string `json:"handle"`
	FollowersCount int    `json:"followersCount"`
}

func getBlueskyToken(identifier, appPassword string) (string, error) {
	payload := map[string]string{
		"identifier": identifier,
		"password":   appPassword,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post("https://bsky.social/xrpc/com.atproto.server.createSession", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth failed: %s", b)
	}

	var session SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return "", err
	}

	return session.AccessJwt, nil
}

func getBlueskyFollowers(handle, token string) (int, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://bsky.social/xrpc/app.bsky.actor.getProfile?actor=%s", handle), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("profile fetch failed: %s", b)
	}

	var profile ProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return 0, err
	}

	return profile.FollowersCount, nil
}
