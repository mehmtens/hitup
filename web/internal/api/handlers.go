package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-chat-app/internal/auth"
	"go-chat-app/internal/broker"
	"go-chat-app/internal/database"
	"go-chat-app/internal/email"
)

type API struct {
	db     *database.DB
	redis  *broker.RedisBroker
	mailer *email.Mailer
}

func NewAPI(db *database.DB, redisBroker *broker.RedisBroker) *API {
	return &API{
		db:     db,
		redis:  redisBroker,
		mailer: email.NewMailer(),
	}
}

type AuthRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string         `json:"token"`
	User  *database.User `json:"user"`
}

func (a *API) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Yalnızca POST desteklenir", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "Kullanıcı adı, e-posta ve şifre zorunludur", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Şifre işlenemedi", http.StatusInternalServerError)
		return
	}

	// Onaylanmamışsa güncelle, onaylıysa çakışma hatası dön
	user, err := a.db.CreateOrUpdateUnverifiedUser(req.Username, req.Email, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	code := email.GenerateOTP()
	if err := a.db.CreateVerificationCode(user.Email, code, "register", 15); err != nil {
		log.Printf("Doğrulama kodu kaydedilemedi: %v", err)
	}

	go func() {
		if err := a.mailer.SendVerificationEmail(user.Email, user.Username, code); err != nil {
			log.Printf("E-posta gönderilemedi: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "pending_verification",
		"message": "Doğrulama kodu e-posta adresinize gönderildi.",
		"email":   user.Email,
	})
}

func (a *API) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Yalnızca POST desteklenir", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Code == "" {
		http.Error(w, "E-posta ve doğrulama kodu zorunludur", http.StatusBadRequest)
		return
	}

	valid, err := a.db.VerifyCode(req.Email, req.Code, "register")
	if err != nil || !valid {
		http.Error(w, "Geçersiz veya süresi dolmuş doğrulama kodu", http.StatusBadRequest)
		return
	}

	user, err := a.db.SetUserVerified(req.Email)
	if err != nil {
		http.Error(w, "Kullanıcı doğrulanamadı", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		http.Error(w, "Token üretilemedi", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: token, User: user})
}

func (a *API) ResendVerificationCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Yalnızca POST desteklenir", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		http.Error(w, "E-posta zorunludur", http.StatusBadRequest)
		return
	}

	user, err := a.db.GetUserByUsernameOrEmail(req.Email)
	if err != nil {
		http.Error(w, "Kullanıcı bulunamadı", http.StatusNotFound)
		return
	}

	code := email.GenerateOTP()
	a.db.CreateVerificationCode(user.Email, code, "register", 15)
	go a.mailer.SendVerificationEmail(user.Email, user.Username, code)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Yeni doğrulama kodu gönderildi."})
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Yalnızca POST desteklenir", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	identifier := req.Username
	if identifier == "" {
		identifier = req.Email
	}

	user, err := a.db.GetUserByUsernameOrEmail(identifier)
	if err != nil || !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		http.Error(w, "Kullanıcı adı veya şifre hatalı", http.StatusUnauthorized)
		return
	}

	if !user.IsVerified {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "unverified",
			"message": "Hesabınız doğrulanmamış. Lütfen e-postanıza gönderilen kod ile onaylayın.",
			"email":   user.Email,
		})
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		http.Error(w, "Token üretilemedi", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: token, User: user})
}

func (a *API) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Yalnızca POST desteklenir", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IDToken == "" {
		http.Error(w, "id_token zorunludur", http.StatusBadRequest)
		return
	}

	googleInfo, err := auth.VerifyGoogleIDToken(req.IDToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Google kimlik doğrulama hatası: %v", err), http.StatusUnauthorized)
		return
	}

	user, err := a.db.GetOrCreateGoogleUser(googleInfo.Sub, googleInfo.Email, googleInfo.Name, googleInfo.Picture)
	if err != nil {
		http.Error(w, "Kullanıcı hesabı oluşturulamadı", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		http.Error(w, "Token üretilemedi", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: token, User: user})
}

func (a *API) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Yalnızca POST desteklenir", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		http.Error(w, "E-posta zorunludur", http.StatusBadRequest)
		return
	}

	user, err := a.db.GetUserByUsernameOrEmail(req.Email)
	if err != nil {
		http.Error(w, "Bu e-posta adresiyle kayıtlı bir hesap bulunamadı.", http.StatusNotFound)
		return
	}

	code := email.GenerateOTP()
	a.db.CreateVerificationCode(user.Email, code, "reset_password", 15)
	go a.mailer.SendPasswordResetEmail(user.Email, code)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Şifre sıfırlama kodu gönderildi.",
		"email":   user.Email,
	})
}

func (a *API) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Yalnızca POST desteklenir", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Code == "" || req.NewPassword == "" {
		http.Error(w, "Tüm alanlar zorunludur", http.StatusBadRequest)
		return
	}

	valid, err := a.db.VerifyCode(req.Email, req.Code, "reset_password")
	if err != nil || !valid {
		http.Error(w, "Geçersiz veya süresi dolmuş kod", http.StatusBadRequest)
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, "Şifre işlenemedi", http.StatusInternalServerError)
		return
	}

	if err := a.db.ResetPasswordByEmail(req.Email, newHash); err != nil {
		http.Error(w, "Şifre güncellenemedi", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Şifreniz başarıyla sıfırlandı. Şimdi giriş yapabilirsiniz."})
}

func (a *API) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	user, err := a.db.GetUserByID(claims.UserID)
	if err != nil {
		http.Error(w, "Kullanıcı bulunamadı", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (a *API) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		AvatarURL string `json:"avatar_url"`
		About     string `json:"about"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	if err := a.db.UpdateProfile(claims.UserID, req.AvatarURL, req.About); err != nil {
		http.Error(w, "Profil güncellenemedi", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *API) UpdatePrivacy(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		PrivacyLastSeen string `json:"privacy_last_seen"`
		PrivacyAvatar   string `json:"privacy_avatar"`
		PrivacyAbout    string `json:"privacy_about"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	if err := a.db.UpdatePrivacy(claims.UserID, req.PrivacyLastSeen, req.PrivacyAvatar, req.PrivacyAbout); err != nil {
		http.Error(w, "Gizlilik ayarları kaydedilemedi", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *API) ToggleBlock(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		BlockedID int `json:"blocked_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BlockedID <= 0 {
		http.Error(w, "Geçersiz blocked_id", http.StatusBadRequest)
		return
	}

	isBlocked, err := a.db.ToggleBlockUser(claims.UserID, req.BlockedID)
	if err != nil {
		http.Error(w, "İşlem başarısız", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"is_blocked": isBlocked})
}

func (a *API) ListUsers(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	users, err := a.db.ListOtherUsers(claims.UserID)
	if err != nil {
		http.Error(w, "Kullanıcılar getirilemedi", http.StatusInternalServerError)
		return
	}

	onlineMap := a.redis.GetOnlineUserIDs(context.Background())
	for i := range users {
		users[i].IsOnline = onlineMap[users[i].ID]
		if users[i].PrivacyAvatar == "nobody" {
			users[i].AvatarURL = ""
		}
		if users[i].PrivacyAbout == "nobody" {
			users[i].About = ""
		}
		if users[i].PrivacyLastSeen == "nobody" {
			users[i].LastSeenAt = nil
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (a *API) GetConversations(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	list, err := a.db.GetUserConversations(claims.UserID)
	if err != nil {
		http.Error(w, "Sohbetler getirilemedi", http.StatusInternalServerError)
		return
	}

	onlineMap := a.redis.GetOnlineUserIDs(context.Background())
	for i := range list {
		if list[i].TargetUser != nil {
			list[i].TargetUser.IsOnline = onlineMap[list[i].TargetUser.ID]
			if list[i].TargetUser.PrivacyAvatar == "nobody" {
				list[i].TargetUser.AvatarURL = ""
				list[i].AvatarURL = ""
			}
			if list[i].TargetUser.PrivacyLastSeen == "nobody" {
				list[i].TargetUser.LastSeenAt = nil
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (a *API) StartDirectConversation(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		RecipientID int `json:"recipient_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RecipientID <= 0 {
		http.Error(w, "Geçersiz recipient_id", http.StatusBadRequest)
		return
	}

	if a.db.IsBlocked(claims.UserID, req.RecipientID) {
		http.Error(w, "Bu kullanıcı ile mesajlaşamazsınız (Engellendi)", http.StatusForbidden)
		return
	}

	convID, err := a.db.GetOrCreateDirectConversation(claims.UserID, req.RecipientID)
	if err != nil {
		http.Error(w, "Sohbet oluşturulamadı", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"conversation_id": convID})
}

func (a *API) CreateGroupConversation(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		AvatarURL   string `json:"avatar_url"`
		MemberIDs   []int  `json:"member_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || len(req.MemberIDs) == 0 {
		http.Error(w, "Grup adı ve üyeler zorunludur", http.StatusBadRequest)
		return
	}

	convID, inviteCode, err := a.db.CreateGroupConversation(req.Name, req.Description, req.AvatarURL, claims.UserID, req.MemberIDs)
	if err != nil {
		http.Error(w, "Grup oluşturulamadı", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation_id": convID,
		"name":            req.Name,
		"invite_code":     inviteCode,
	})
}

func (a *API) JoinGroupByInvite(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		InviteCode string `json:"invite_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InviteCode == "" {
		http.Error(w, "Davet kodu zorunludur", http.StatusBadRequest)
		return
	}

	convID, err := a.db.JoinGroupByInvite(claims.UserID, req.InviteCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"conversation_id": convID})
}

func (a *API) AddGroupMember(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		ConversationID int `json:"conversation_id"`
		NewMemberID    int `json:"new_member_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	if err := a.db.AddGroupMember(claims.UserID, req.ConversationID, req.NewMemberID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		ConversationID int `json:"conversation_id"`
		MemberID       int `json:"member_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	if err := a.db.RemoveGroupMember(claims.UserID, req.ConversationID, req.MemberID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	_, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	convID, _ := strconv.Atoi(r.URL.Query().Get("conv_id"))
	members, err := a.db.GetGroupMembers(convID)
	if err != nil {
		http.Error(w, "Üyeler getirilemedi", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func (a *API) TogglePinConversation(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		ConversationID int `json:"conversation_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	isPinned, err := a.db.TogglePinConversation(claims.UserID, req.ConversationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"is_pinned": isPinned})
}

func (a *API) ToggleMuteConversation(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		ConversationID int `json:"conversation_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	isMuted, err := a.db.ToggleMuteConversation(claims.UserID, req.ConversationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"is_muted": isMuted})
}

func (a *API) ToggleArchiveConversation(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		ConversationID int `json:"conversation_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	isArchived, err := a.db.ToggleArchiveConversation(claims.UserID, req.ConversationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"is_archived": isArchived})
}

func (a *API) SetWallpaper(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		ConversationID int    `json:"conversation_id"`
		Wallpaper      string `json:"wallpaper"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	a.db.SetConversationWallpaper(claims.UserID, req.ConversationID, req.Wallpaper)
	w.WriteHeader(http.StatusOK)
}

func (a *API) ClearHistory(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		ConversationID int `json:"conversation_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	a.db.ClearConversationHistory(req.ConversationID, claims.UserID)
	w.WriteHeader(http.StatusOK)
}

func (a *API) GetGallery(w http.ResponseWriter, r *http.Request) {
	_, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	convID, _ := strconv.Atoi(r.URL.Query().Get("conv_id"))
	mediaList, err := a.db.GetConversationMediaGallery(convID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mediaList)
}

func (a *API) GetMessages(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	convID, _ := strconv.Atoi(r.URL.Query().Get("conv_id"))
	beforeID, _ := strconv.Atoi(r.URL.Query().Get("before_id"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	messages, err := a.db.GetConversationMessages(convID, claims.UserID, limit, beforeID)
	if err != nil {
		http.Error(w, "Mesajlar alınamadı", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (a *API) SearchMessages(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	convID, _ := strconv.Atoi(r.URL.Query().Get("conv_id"))
	query := r.URL.Query().Get("q")
	if convID <= 0 || query == "" {
		http.Error(w, "conv_id ve q parametreleri zorunludur", http.StatusBadRequest)
		return
	}

	messages, err := a.db.SearchMessages(convID, claims.UserID, query)
	if err != nil {
		http.Error(w, "Arama hatası", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (a *API) ToggleStarMessage(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		MessageID int `json:"message_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	isStarred, err := a.db.ToggleStarMessage(claims.UserID, req.MessageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"is_starred": isStarred})
}

func (a *API) GetStarredMessages(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	messages, err := a.db.GetStarredMessages(claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (a *API) VotePoll(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		PollID   int `json:"poll_id"`
		OptionID int `json:"option_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	poll, err := a.db.VotePoll(req.PollID, req.OptionID, claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(poll)
}

func (a *API) CreateStatus(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		Content  string `json:"content"`
		MediaURL string `json:"media_url"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	status, err := a.db.CreateStatus(claims.UserID, req.Content, req.MediaURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (a *API) GetStatuses(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	statuses, err := a.db.GetActiveStatuses(claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

func (a *API) ViewStatus(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		StatusID int `json:"status_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	a.db.ViewStatus(req.StatusID, claims.UserID)
	w.WriteHeader(http.StatusOK)
}

func (a *API) GetCalls(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	logs, err := a.db.GetCallLogs(claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (a *API) LogCall(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		ReceiverID int    `json:"receiver_id"`
		ConvID     int    `json:"conversation_id"`
		Type       string `json:"type"`
		Status     string `json:"status"`
		Duration   int    `json:"duration_seconds"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	a.db.LogCall(claims.UserID, req.ReceiverID, req.ConvID, req.Type, req.Status, req.Duration)
	w.WriteHeader(http.StatusOK)
}

func (a *API) PreviewLink(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, "url parametresi gerekli", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		http.Error(w, "URL açılamadı", http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	htmlContent := string(bodyBytes)

	preview := database.LinkPreview{URL: targetURL}

	titleRe := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	if match := titleRe.FindStringSubmatch(htmlContent); len(match) > 1 {
		preview.Title = match[1]
	}

	descRe := regexp.MustCompile(`(?i)<meta property="og:description" content="(.*?)"`)
	if match := descRe.FindStringSubmatch(htmlContent); len(match) > 1 {
		preview.Description = match[1]
	}

	imgRe := regexp.MustCompile(`(?i)<meta property="og:image" content="(.*?)"`)
	if match := imgRe.FindStringSubmatch(htmlContent); len(match) > 1 {
		preview.Image = match[1]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(preview)
}

func (a *API) UploadFile(w http.ResponseWriter, r *http.Request) {
	_, err := a.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	r.ParseMultipartForm(100 << 20)
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Dosya okunamadı", http.StatusBadRequest)
		return
	}
	defer file.Close()

	os.MkdirAll("./uploads", os.ModePerm)
	ext := filepath.Ext(header.Filename)
	uniqueFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), url.PathEscape(header.Filename))
	dstPath := filepath.Join("./uploads", uniqueFilename)

	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Dosya kaydedilemedi", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Dosya yazılamadı", http.StatusInternalServerError)
		return
	}

	msgType := "file"
	lowerExt := strings.ToLower(ext)
	if lowerExt == ".jpg" || lowerExt == ".jpeg" || lowerExt == ".png" || lowerExt == ".gif" || lowerExt == ".webp" {
		msgType = "image"
	} else if lowerExt == ".mp4" || lowerExt == ".mov" || lowerExt == ".webm" || lowerExt == ".mkv" {
		msgType = "video"
	} else if lowerExt == ".mp3" || lowerExt == ".ogg" || lowerExt == ".wav" || lowerExt == ".m4a" {
		msgType = "audio"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":       "/uploads/" + uniqueFilename,
		"filename":  header.Filename,
		"file_size": header.Size,
		"type":      msgType,
	})
}

func (a *API) authenticate(r *http.Request) (*auth.Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		authHeader = r.URL.Query().Get("token")
	} else {
		authHeader = strings.TrimPrefix(authHeader, "Bearer ")
	}

	return auth.ValidateToken(authHeader)
}
