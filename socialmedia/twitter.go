package socialmedia

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type TwitterResponse struct {
	Data struct {
		Username      string `json:"username"`
		PublicMetrics struct {
			FollowersCount int `json:"followers_count"`
		} `json:"public_metrics"`
	} `json:"data"`
}

func GetTwitterFollowers(username, bearerToken string) (int, error) {
	url := fmt.Sprintf("https://api.twitter.com/2/users/by/username/%s?user.fields=public_metrics", username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Twitter API error: %s", resp.Status)
	}

	var result TwitterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.Data.PublicMetrics.FollowersCount, nil
}
