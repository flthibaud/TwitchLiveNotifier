package twitch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Stream represents the Twitch Helix /streams response for a live stream
type Stream struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	UserName     string    `json:"user_name"`
	GameID       string    `json:"game_id"`
	GameName     string    `json:"game_name"`
	Title        string    `json:"title"`
	ViewerCount  int       `json:"viewer_count"`
	StartedAt    time.Time `json:"started_at"`
	Language     string    `json:"language"`
	ThumbnailURL string    `json:"thumbnail_url"`
	IsMature     bool      `json:"is_mature"`
}

// streamsResponse wraps the JSON response from Twitch Helix
type streamsResponse struct {
	Data []Stream `json:"data"`
}

// GetStreamInfo fetches stream information for the given broadcaster ID.
// Returns a pointer to Stream if live, or nil if offline.
func GetStreamInfo(broadcasterID, clientID, oauthToken string) (*Stream, error) {
	url := fmt.Sprintf("https://api.twitch.tv/helix/streams?user_id=%s", broadcasterID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", "Bearer "+oauthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("twitch API error: %s", resp.Status)
	}

	var sr streamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	if len(sr.Data) == 0 {
		return nil, nil // offline
	}
	return &sr.Data[0], nil
}
