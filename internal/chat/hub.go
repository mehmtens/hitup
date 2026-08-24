package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"go-chat-app/internal/broker"
	"go-chat-app/internal/database"
)

type Message struct {
	ID              int                      `json:"id,omitempty"`
	Type            string                   `json:"type"` 
	ConversationID  int                      `json:"conversation_id,omitempty"`
	SenderID        int                      `json:"sender_id"`
	SenderUsername  string                   `json:"sender_username,omitempty"`
	SenderAvatar    string                   `json:"sender_avatar,omitempty"`
	TargetUserID    int                      `json:"target_user_id,omitempty"`
	Content         string                   `json:"content,omitempty"`
	MediaURL        string                   `json:"media_url,omitempty"`
	FileName        string                   `json:"file_name,omitempty"`
	FileSize        int                      `json:"file_size,omitempty"`
	LocationLat     float64                  `json:"location_lat,omitempty"`
	LocationLng     float64                  `json:"location_lng,omitempty"`
	LinkPreview     *database.LinkPreview    `json:"link_preview,omitempty"`
	Poll            *database.PollDTO        `json:"poll,omitempty"`
	PollQuestion    string                   `json:"poll_question,omitempty"`
	PollOptions     []string                 `json:"poll_options,omitempty"`
	PollMultiple    bool                     `json:"poll_multiple,omitempty"`
	PollID          int                      `json:"poll_id,omitempty"`
	PollOptionID    int                      `json:"poll_option_id,omitempty"`
	ReplyToID       *int                     `json:"reply_to_id,omitempty"`
	ReplyToContent  string                   `json:"reply_to_content,omitempty"`
	ReplyToSender   string                   `json:"reply_to_sender,omitempty"`
	ForwardedFromID *int                     `json:"forwarded_from_id,omitempty"`
	Emoji           string                   `json:"emoji,omitempty"`
	Reactions       []database.ReactionCount `json:"reactions,omitempty"`
	Status          string                   `json:"status,omitempty"`
	IsDeleted       bool                     `json:"is_deleted,omitempty"`
	IsEdited        bool                     `json:"is_edited,omitempty"`
	DurationSec     int                      `json:"duration_sec,omitempty"`
	SDP             string                   `json:"sdp,omitempty"`
	Candidate       json.RawMessage          `json:"candidate,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[int]map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	db         *database.DB
	redis      *broker.RedisBroker
}

func NewHub(db *database.DB, redisBroker *broker.RedisBroker) *Hub {
	return &Hub{
		clients:    make(map[int]map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		db:         db,
		redis:      redisBroker,
	}
}

func (h *Hub) Run(ctx context.Context) {
	if h.redis != nil {
		h.redis.SubscribeAll(ctx, func(channel string, payload []byte) {
			var msg Message
			if err := json.Unmarshal(payload, &msg); err == nil {
				h.broadcastLocal(msg, payload)
			}
		})
	}

	// Arka plan temizleyici: Süresi dolan kaybolan mesajlar
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if h.db != nil {
				deletedIDs, err := h.db.CleanupExpiredMessages()
				if err == nil && len(deletedIDs) > 0 {
					for _, dID := range deletedIDs {
						delMsg := Message{Type: "delete_message_for_everyone", ID: dID}
						data, _ := json.Marshal(delMsg)
						h.broadcastLocal(delMsg, data)
					}
				}
			}
		}
	}()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]bool)
			}
			h.clients[client.userID][client] = true
			isFirstConn := len(h.clients[client.userID]) == 1
			h.mu.Unlock()

			if isFirstConn && h.redis != nil {
				h.redis.SetUserOnline(ctx, client.userID)
				presenceMsg := Message{
					Type:     "presence",
					SenderID: client.userID,
					Content:  "online",
				}
				h.redis.Publish(ctx, "chat:presence", presenceMsg)
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if userClients, ok := h.clients[client.userID]; ok {
				if _, exists := userClients[client]; exists {
					delete(userClients, client)
					close(client.send)
					if len(userClients) == 0 {
						delete(h.clients, client.userID)
						if h.redis != nil {
							h.redis.SetUserOffline(ctx, client.userID)
						}
						if h.db != nil {
							h.db.UpdateLastSeen(client.userID)
						}
						presenceMsg := Message{
							Type:      "presence",
							SenderID:  client.userID,
							Content:   "offline",
							CreatedAt: time.Now(),
						}
						if h.redis != nil {
							h.redis.Publish(ctx, "chat:presence", presenceMsg)
						}
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			// WebRTC Çağrı & Ekran Paylaşımı Sinyalleşmesi
			if isCallSignal(msg.Type) {
				if h.redis != nil {
					h.redis.Publish(ctx, fmt.Sprintf("chat:call_%d", msg.TargetUserID), msg)
				} else {
					data, _ := json.Marshal(msg)
					h.sendToUser(msg.TargetUserID, data)
				}
				continue
			}

			// Mesaj Kaydı
			if (msg.Type == "chat" || msg.Type == "image" || msg.Type == "video" || msg.Type == "audio" || msg.Type == "file" || msg.Type == "location") && h.db != nil {
				savedRecord, err := h.db.SaveMessage(database.SaveMessageParams{
					ConversationID:  msg.ConversationID,
					SenderID:        msg.SenderID,
					Content:         msg.Content,
					Type:            msg.Type,
					MediaURL:        msg.MediaURL,
					FileName:        msg.FileName,
					FileSize:        msg.FileSize,
					LocationLat:     msg.LocationLat,
					LocationLng:     msg.LocationLng,
					LinkPreview:     msg.LinkPreview,
					ReplyToID:       msg.ReplyToID,
					ForwardedFromID: msg.ForwardedFromID,
					DurationSec:     msg.DurationSec,
				})
				if err != nil {
					log.Printf("Mesaj kaydedilemedi: %v", err)
					continue
				}
				msg.ID = savedRecord.ID
				msg.CreatedAt = savedRecord.CreatedAt
			} else if msg.Type == "poll_create" && h.db != nil {
				// Anketli mesaj oluştur
				savedRecord, err := h.db.SaveMessage(database.SaveMessageParams{
					ConversationID: msg.ConversationID,
					SenderID:       msg.SenderID,
					Content:        msg.PollQuestion,
					Type:           "poll",
				})
				if err == nil {
					pollDTO, pErr := h.db.CreatePoll(savedRecord.ID, msg.PollQuestion, msg.PollMultiple, msg.PollOptions)
					if pErr == nil {
						msg.ID = savedRecord.ID
						msg.Type = "poll"
						msg.Poll = pollDTO
						msg.Content = msg.PollQuestion
						msg.CreatedAt = savedRecord.CreatedAt
					}
				}
			} else if msg.Type == "poll_vote" && h.db != nil {
				pollDTO, err := h.db.VotePoll(msg.PollID, msg.PollOptionID, msg.SenderID)
				if err == nil {
					msg.Poll = pollDTO
				}
			} else if msg.Type == "read" && h.db != nil {
				h.db.MarkMessagesAsRead(msg.ConversationID, msg.SenderID)
			} else if msg.Type == "delete_message_for_everyone" && h.db != nil {
				updated, err := h.db.DeleteMessageForEveryone(msg.ID, msg.SenderID)
				if err == nil {
					msg.Content = updated.Content
					msg.IsDeleted = true
					msg.MediaURL = ""
				}
			} else if msg.Type == "delete_message_for_me" && h.db != nil {
				h.db.DeleteMessageForMe(msg.ID, msg.SenderID)
				// Yalnızca kullanıcıya silindiği bildirilir
				data, _ := json.Marshal(msg)
				h.sendToUser(msg.SenderID, data)
				continue
			} else if msg.Type == "edit_message" && h.db != nil {
				updated, err := h.db.EditMessage(msg.ID, msg.SenderID, msg.Content)
				if err == nil {
					msg.IsEdited = true
					msg.Content = updated.Content
				}
			} else if msg.Type == "reaction" && h.db != nil {
				reactions, err := h.db.ToggleReaction(msg.ID, msg.SenderID, msg.Emoji)
				if err == nil {
					msg.Reactions = reactions
				}
			}

			if h.redis != nil {
				channel := fmt.Sprintf("chat:conv_%d", msg.ConversationID)
				h.redis.Publish(ctx, channel, msg)
			} else {
				data, _ := json.Marshal(msg)
				h.broadcastLocal(msg, data)
			}
		}
	}
}

func isCallSignal(t string) bool {
	return t == "call_offer" || t == "call_answer" || t == "ice_candidate" || t == "call_reject" || t == "call_end" || t == "screen_share_offer" || t == "screen_share_answer"
}

func (h *Hub) sendToUser(userID int, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if userClients, ok := h.clients[userID]; ok {
		for client := range userClients {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

func (h *Hub) broadcastLocal(msg Message, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if isCallSignal(msg.Type) && msg.TargetUserID > 0 {
		if userClients, ok := h.clients[msg.TargetUserID]; ok {
			for client := range userClients {
				select {
				case client.send <- data:
				default:
				}
			}
		}
		return
	}

	for _, userClients := range h.clients {
		for client := range userClients {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(userClients, client)
			}
		}
	}
}
