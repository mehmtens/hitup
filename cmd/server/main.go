package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"go-chat-app/internal/api"
	"go-chat-app/internal/broker"
	"go-chat-app/internal/chat"
	"go-chat-app/internal/database"
)

func main() {
	port := flag.String("port", "8081", "Sunucu portu")
	dbHost := flag.String("dbhost", "127.0.0.1:5433", "PostgreSQL Host veya URL")
	redisHost := flag.String("redishost", "127.0.0.1:6379", "Redis Host veya URL")
	flag.Parse()

	// Render / Bulut ortam değişkeni önceliği
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}
	if envDB := os.Getenv("DATABASE_URL"); envDB != "" {
		*dbHost = envDB
	}
	if envRedis := os.Getenv("REDIS_URL"); envRedis != "" {
		*redisHost = envRedis
	}

	ctx := context.Background()

	os.MkdirAll("./uploads", os.ModePerm)

	// PostgreSQL Bağlantı Dizesi
	connStr := *dbHost
	if !strings.HasPrefix(connStr, "postgres://") && !strings.HasPrefix(connStr, "postgresql://") {
		connStr = fmt.Sprintf("postgres://postgres:postgres@%s/chatdb?sslmode=disable", *dbHost)
	}

	db, err := database.Connect(connStr)
	if err != nil {
		log.Fatalf("Veritabanı başlatılamadı: %v", err)
	}

	// Redis Broker
	redisBroker, err := broker.NewRedisBroker(*redisHost)
	if err != nil {
		log.Fatalf("Redis başlatılamadı: %v", err)
	}

	// Hub
	hub := chat.NewHub(db, redisBroker)
	go hub.Run(ctx)

	apiHandler := api.NewAPI(db, redisBroker)

	// Auth & Profile
	http.HandleFunc("/api/register", apiHandler.Register)
	http.HandleFunc("/api/verify-email", apiHandler.VerifyEmail)
	http.HandleFunc("/api/resend-code", apiHandler.ResendVerificationCode)
	http.HandleFunc("/api/login", apiHandler.Login)
	http.HandleFunc("/api/auth/google", apiHandler.GoogleAuth)
	http.HandleFunc("/api/forgot-password", apiHandler.ForgotPassword)
	http.HandleFunc("/api/reset-password", apiHandler.ResetPassword)
	http.HandleFunc("/api/me", apiHandler.GetMe)
	http.HandleFunc("/api/profile", apiHandler.UpdateProfile)
	http.HandleFunc("/api/privacy", apiHandler.UpdatePrivacy)
	http.HandleFunc("/api/block", apiHandler.ToggleBlock)
	http.HandleFunc("/api/users", apiHandler.ListUsers)

	// Conversations
	http.HandleFunc("/api/conversations", apiHandler.GetConversations)
	http.HandleFunc("/api/conversations/direct", apiHandler.StartDirectConversation)
	http.HandleFunc("/api/conversations/group", apiHandler.CreateGroupConversation)
	http.HandleFunc("/api/conversations/join-invite", apiHandler.JoinGroupByInvite)
	http.HandleFunc("/api/conversations/add-member", apiHandler.AddGroupMember)
	http.HandleFunc("/api/conversations/remove-member", apiHandler.RemoveGroupMember)
	http.HandleFunc("/api/conversations/members", apiHandler.GetGroupMembers)
	http.HandleFunc("/api/conversations/pin", apiHandler.TogglePinConversation)
	http.HandleFunc("/api/conversations/mute", apiHandler.ToggleMuteConversation)
	http.HandleFunc("/api/conversations/archive", apiHandler.ToggleArchiveConversation)
	http.HandleFunc("/api/conversations/wallpaper", apiHandler.SetWallpaper)
	http.HandleFunc("/api/conversations/clear", apiHandler.ClearHistory)
	http.HandleFunc("/api/conversations/gallery", apiHandler.GetGallery)
	http.HandleFunc("/api/conversations/messages", apiHandler.GetMessages)
	http.HandleFunc("/api/conversations/search", apiHandler.SearchMessages)

	// Messages & Rich Media
	http.HandleFunc("/api/messages/star", apiHandler.ToggleStarMessage)
	http.HandleFunc("/api/messages/starred", apiHandler.GetStarredMessages)
	http.HandleFunc("/api/messages/poll-vote", apiHandler.VotePoll)
	http.HandleFunc("/api/preview-link", apiHandler.PreviewLink)
	http.HandleFunc("/api/upload", apiHandler.UploadFile)

	// Stories / Status
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apiHandler.CreateStatus(w, r)
		} else {
			apiHandler.GetStatuses(w, r)
		}
	})
	http.HandleFunc("/api/status/view", apiHandler.ViewStatus)

	// Calls
	http.HandleFunc("/api/calls", apiHandler.GetCalls)
	http.HandleFunc("/api/calls/log", apiHandler.LogCall)

	// Static Assets
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	http.Handle("/", http.FileServer(http.Dir("./web")))

	// WebSocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		chat.ServeWs(hub, w, r)
	})

	addr := fmt.Sprintf(":%s", *port)
	log.Printf("HitUp Sunucusu http://localhost:%s adresinde çalışıyor...", *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Sunucu hatası: ", err)
	}
}
