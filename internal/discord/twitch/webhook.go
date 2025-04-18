package twitch

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"discord-bot-env/internal/config"
	"discord-bot-env/internal/discord"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

// WebhookServer handles Twitch EventSub webhooks
type WebhookServer struct {
	cfg           *config.Config
	logger        *logrus.Logger
	discordClient *discord.Client
	httpServer    *http.Server
	oauthToken    string
}

// NewServer instantiates the Twitch webhook server
func NewServer(cfg *config.Config, logger *logrus.Logger, discordClient *discord.Client) *WebhookServer {
	mux := http.NewServeMux()
	srv := &WebhookServer{
		cfg:           cfg,
		logger:        logger,
		discordClient: discordClient,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.Port),
			Handler: mux,
		},
	}
	mux.HandleFunc("/webhook", srv.handleWebhook)
	return srv
}

// Start obtains an OAuth token, subscribes to stream.online, and starts the HTTP server
func (s *WebhookServer) Start(ctx context.Context) error {
	// 1. Get OAuth token for Twitch API
	token, err := s.getOAuthToken()
	if err != nil {
		return fmt.Errorf("error getting OAuth token: %w", err)
	}
	s.oauthToken = token

	// 2. Subscribe to stream.online for each BROADCASTER_ID env var
	for _, broadcasterID := range s.cfg.TwitchBroadcasterIDs {
		if err := s.subscribeStreamOnline(broadcasterID); err != nil {
			s.logger.Errorf("Error subscribing to stream.online for %s: %v", broadcasterID, err)
		}
	}
	s.logger.Infof("Subscriptions created for broadcaster IDs: %s", s.cfg.TwitchBroadcasterIDs)

	// 3. Start HTTP server
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Webhook server error: %v", err)
		}
	}()

	s.logger.Infof("Twitch webhook listening on %s (public callback: %s/webhook)",
		s.httpServer.Addr,
		s.cfg.CallbackURL,
	)

	// Wait for shutdown
	<-ctx.Done()
	return s.httpServer.Shutdown(context.Background())
}

// getOAuthToken fetches an app access token from Twitch
func (s *WebhookServer) getOAuthToken() (string, error) {
	url := fmt.Sprintf("https://id.twitch.tv/oauth2/token?client_id=%s&client_secret=%s&grant_type=client_credentials",
		s.cfg.TwitchClientID, s.cfg.TwitchClientSecret)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var data struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.AccessToken, nil
}

// subscribeStreamOnline creates a Twitch EventSub subscription of type stream.online
func (s *WebhookServer) subscribeStreamOnline(broadcasterID string) error {
	client := http.DefaultClient
	baseURL := "https://api.twitch.tv/helix/eventsub/subscriptions"

	// Determine current desired callback URL
	callbackURL := fmt.Sprintf("%s/webhook", s.cfg.CallbackURL)

	// 1. List existing subscriptions for this broadcaster and type
	listURL := fmt.Sprintf(
		"%s?type=stream.online&condition[broadcaster_user_id]=%s", baseURL, broadcasterID,
	)
	reqList, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		return err
	}
	reqList.Header.Set("Client-ID", s.cfg.TwitchClientID)
	reqList.Header.Set("Authorization", "Bearer "+s.oauthToken)

	respList, err := client.Do(reqList)
	if err != nil {
		return err
	}
	defer respList.Body.Close()

	if respList.StatusCode == http.StatusTooManyRequests {
		s.logger.Warn("Twitch subscription rate limit reached, skipping subscription check")
		return nil
	}
	if respList.StatusCode/100 != 2 {
		data, _ := io.ReadAll(respList.Body)
		return fmt.Errorf("error listing subscriptions: %s", data)
	}

	// Decode existing subscriptions
	var listData struct {
		Data []struct {
			ID        string `json:"id"`
			Transport struct {
				Callback string `json:"callback"`
				Method   string `json:"method"`
			} `json:"transport"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respList.Body).Decode(&listData); err != nil {
		return err
	}

	// Check for existing subscription
	for _, sub := range listData.Data {
		if sub.Transport.Callback == callbackURL {
			s.logger.Infof("Valid subscription exists (ID=%s), no action needed", sub.ID)
			return nil
		}
		// Outdated callback, delete it
		delURL := fmt.Sprintf("%s?id=%s", baseURL, sub.ID)
		reqDel, _ := http.NewRequest("DELETE", delURL, nil)
		reqDel.Header.Set("Client-ID", s.cfg.TwitchClientID)
		reqDel.Header.Set("Authorization", "Bearer "+s.oauthToken)
		respDel, err := client.Do(reqDel)
		if err != nil {
			s.logger.Warnf("failed to delete old subscription %s: %v", sub.ID, err)
		} else {
			respDel.Body.Close()
			s.logger.Infof("Deleted outdated subscription (ID=%s)", sub.ID)
		}
		// continue to ensure no matching subscription remains
	}

	// 2. Create new subscription with correct callback
	s.logger.Infof("Creating new subscription for stream.online with callback %s", callbackURL)
	body := map[string]interface{}{
		"type":    "stream.online",
		"version": "1",
		"condition": map[string]string{
			"broadcaster_user_id": broadcasterID,
		},
		"transport": map[string]string{
			"method":   "webhook",
			"callback": callbackURL,
			"secret":   s.cfg.TwitchWebhookSecret,
		},
	}
	jsonData, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Client-ID", s.cfg.TwitchClientID)
	req.Header.Set("Authorization", "Bearer "+s.oauthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error creating subscription: %s", data)
	}
	s.logger.Info("Subscription created successfully")
	return nil
}

// handleWebhook processes Twitch EventSub callbacks
func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// 1) Logs & body brut
	s.logger.Infof("📥 Webhook reçu : %s %s", r.Method, r.URL.Path)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Errorf("Lecture du body échouée : %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	s.logger.Infof("📝 Payload brut : %s", string(body))

	// 2) Récupère les headers Twitch
	msgType := r.Header.Get("Twitch-Eventsub-Message-Type") // webhook_callback_verification | notification | revocation
	msgID := r.Header.Get("Twitch-Eventsub-Message-Id")
	timestamp := r.Header.Get("Twitch-Eventsub-Message-Timestamp")
	signature := r.Header.Get("Twitch-Eventsub-Message-Signature")

	// 3) Vérifie signature HMAC
	if !s.verifySignature(msgID+timestamp+string(body), signature) {
		s.logger.Warn("Signature invalide, on rejette")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	s.logger.Infof("Signature OK – type=%s", msgType)

	// 4) Route selon le header
	switch msgType {
	case "webhook_callback_verification":
		// Renvoie le challenge pour validation chez Twitch
		var challenge struct {
			Challenge string `json:"challenge"`
		}
		if err := json.Unmarshal(body, &challenge); err != nil {
			s.logger.Errorf("Parsing challenge échoué : %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.logger.Info("Répond au challenge")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(challenge.Challenge))
		return

	case "notification":
		s.logger.Info("Notification reçue, on parse l’événement")
		// On isole subscription.type et event
		var payload struct {
			Subscription struct {
				Type string `json:"type"`
			} `json:"subscription"`
			Event struct {
				BroadcasterUserID   string `json:"broadcaster_user_id"`
				BroadcasterUserName string `json:"broadcaster_user_name"`
				StartedAt           string `json:"started_at"`
			} `json:"event"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			s.logger.Errorf("Parsing notification échoué : %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Si c’est un stream.online
		if payload.Subscription.Type == "stream.online" {
			s.logger.Infof("📣 %s est en live !", payload.Event.BroadcasterUserName)
			stream, err := GetStreamInfo(payload.Event.BroadcasterUserID, s.cfg.TwitchClientID, s.oauthToken)
			if err != nil {
				s.logger.Errorf("Error fetching stream info: %v", err)
			} else if stream != nil {
				// build embed with stream.UserName, stream.Title, stream.GameName, stream.ViewerCount, stream.StartedAt
				embed := &discordgo.MessageEmbed{
					Title: fmt.Sprintf("🔴 %s est en live !", stream.UserName),
					URL:   fmt.Sprintf("https://twitch.tv/%s", stream.UserName),

					Color: 0x9146FF, // Twitch purple

					Author: &discordgo.MessageEmbedAuthor{
						Name:    stream.UserName,
						URL:     fmt.Sprintf("https://twitch.tv/%s", stream.UserName),
						IconURL: fmt.Sprintf("https://static-cdn.jtvnw.net/jtv_user_pictures/%s-profile_image-70x70.png", stream.UserID),
						// IconURL: "https://static-cdn.jtvnw.net/user-default-pictures-uv/ead5c8b2-a4c9-4724-b1dd-9f00b46cbd3d-profile_image-70x70.png",
					},

					Image: &discordgo.MessageEmbedImage{
						URL:    fmt.Sprintf("https://static-cdn.jtvnw.net/previews-ttv/live_user_%s-440x248.jpg", stream.UserName),
						Width:  440,
						Height: 248,
					},

					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "📝 Titre",
							Value:  stream.Title,
							Inline: false,
						},
						{
							Name:   "🎮 Jeu",
							Value:  stream.GameName,
							Inline: true,
						},
						{
							Name:   "👀 Spectateurs",
							Value:  fmt.Sprintf("%d", stream.ViewerCount),
							Inline: true,
						},
					},

					Timestamp: stream.StartedAt.Format(time.RFC3339), // RFC3339 string

					Footer: &discordgo.MessageEmbedFooter{
						Text:    "Suivez sur Twitch !",
						IconURL: "https://static.twitchcdn.net/assets/favicon-32-e29e246c157142c94346.png",
					},
				}
				if err := s.discordClient.SendEmbed(s.cfg.NotifyChannelID, embed); err != nil {
					s.logger.Errorf("Envoi Discord raté : %v", err)
				} else {
					s.logger.Info("Embed Discord envoyé ✅")
				}
			}
		}

		w.WriteHeader(http.StatusNoContent)
		return

	case "revocation":
		s.logger.Warn("Subscription révoquée par Twitch")
		w.WriteHeader(http.StatusNoContent)
		return

	default:
		s.logger.Infof("Type inattendu : %s", msgType)
		w.WriteHeader(http.StatusNoContent)
		return
	}
}

// verifySignature checks Twitch signature header against payload
func (s *WebhookServer) verifySignature(message, signature string) bool {
	h := hmac.New(sha256.New, []byte(s.cfg.TwitchWebhookSecret))
	h.Write([]byte(message))
	expected := "sha256=" + hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
