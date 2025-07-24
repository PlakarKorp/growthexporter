package socialmedia

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func GetSubredditSubscribers(subreddit string) (int, error) {
	url := fmt.Sprintf("https://www.reddit.com/r/%s/about.json", subreddit)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Plakar/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data struct {
		Data struct {
			Subscribers int `json:"subscribers"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	return data.Data.Subscribers, nil
}
