package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

type User struct {
	ID               int        `json:"id"`
	Username         string     `json:"username"`
	Email            string     `json:"email"`
	PasswordHash     string     `json:"-"`
	GoogleID         *string    `json:"google_id,omitempty"`
	IsVerified       bool       `json:"is_verified"`
	AvatarURL        string     `json:"avatar_url"`
	About            string     `json:"about"`
	PrivacyLastSeen  string     `json:"privacy_last_seen"`
	PrivacyAvatar    string     `json:"privacy_avatar"`
	PrivacyAbout     string     `json:"privacy_about"`
	LastSeenAt       *time.Time `json:"last_seen_at"`
	IsOnline         bool       `json:"is_online"`
	CreatedAt        time.Time  `json:"created_at"`
}

type UserStatus struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	MediaURL  string    `json:"media_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ConversationListItem struct {
	ID          int        `json:"id"`
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	AvatarURL   string     `json:"avatar_url"`
	Description string     `json:"description"`
	InviteCode  string     `json:"invite_code,omitempty"`
	LastMessage string     `json:"last_message"`
	LastTime    *time.Time `json:"last_time"`
	UnreadCount int        `json:"unread_count"`
	IsPinned    bool       `json:"is_pinned"`
	IsMuted     bool       `json:"is_muted"`
	IsArchived  bool       `json:"is_archived"`
	Wallpaper   string     `json:"wallpaper"`
	TargetUser  *User      `json:"target_user,omitempty"`
}

type ReactionCount struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
}

type LinkPreview struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	URL         string `json:"url,omitempty"`
}

type PollOptionDTO struct {
	ID        int      `json:"id"`
	Text      string   `json:"text"`
	VoteCount int      `json:"vote_count"`
	Voters    []string `json:"voters,omitempty"`
	HasVoted  bool     `json:"has_voted"`
}

type PollDTO struct {
	ID              int             `json:"id"`
	Question        string          `json:"question"`
	MultipleAnswers bool            `json:"multiple_answers"`
	Options         []PollOptionDTO `json:"options"`
	TotalVotes      int             `json:"total_votes"`
}

type MessageRecord struct {
	ID              int             `json:"id"`
	ConversationID  int             `json:"conversation_id"`
	SenderID        int             `json:"sender_id"`
	SenderUsername  string          `json:"sender_username"`
	SenderAvatar    string          `json:"sender_avatar"`
	Content         string          `json:"content"`
	Type            string          `json:"type"`
	MediaURL        string          `json:"media_url,omitempty"`
	FileName        string          `json:"file_name,omitempty"`
	FileSize        int             `json:"file_size,omitempty"`
	LocationLat     float64         `json:"location_lat,omitempty"`
	LocationLng     float64         `json:"location_lng,omitempty"`
	LinkPreview     *LinkPreview    `json:"link_preview,omitempty"`
	Poll            *PollDTO        `json:"poll,omitempty"`
	ReplyToID       *int            `json:"reply_to_id,omitempty"`
	ReplyToContent  string          `json:"reply_to_content,omitempty"`
	ReplyToSender   string          `json:"reply_to_sender,omitempty"`
	ForwardedFromID *int            `json:"forwarded_from_id,omitempty"`
	Reactions       []ReactionCount `json:"reactions,omitempty"`
	Status          string          `json:"status"`
	IsStarred       bool            `json:"is_starred"`
	IsDeleted       bool            `json:"is_deleted"`
	IsEdited        bool            `json:"is_edited"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

type UserStatusDTO struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatar_url"`
	Content   string    `json:"content"`
	MediaURL  string    `json:"media_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Views     []string  `json:"views,omitempty"`
}

type CallLogDTO struct {
	ID              int       `json:"id"`
	CallerID        int       `json:"caller_id"`
	CallerName      string    `json:"caller_name"`
	ReceiverID      int       `json:"receiver_id"`
	ReceiverName    string    `json:"receiver_name"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`
	DurationSeconds int       `json:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}

func Connect(connStr string) (*DB, error) {
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("bağlantı açılamadı: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping hatası: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("şema hatası: %w", err)
	}

	log.Println("PostgreSQL bağlantısı ve HitUp Enterprise şeması hazır.")
	return db, nil
}

func (d *DB) initSchema() error {
	queries := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		google_id VARCHAR(100) UNIQUE DEFAULT NULL,
		is_verified BOOLEAN NOT NULL DEFAULT FALSE,
		avatar_url TEXT DEFAULT '',
		about TEXT DEFAULT 'Hey there! I am using HitUp.',
		privacy_last_seen VARCHAR(20) DEFAULT 'everyone',
		privacy_avatar VARCHAR(20) DEFAULT 'everyone',
		privacy_about VARCHAR(20) DEFAULT 'everyone',
		last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(100) DEFAULT '';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS google_id VARCHAR(100) UNIQUE DEFAULT NULL;
	ALTER TABLE users ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT FALSE;
	ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT DEFAULT '';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS about TEXT DEFAULT 'Hey there! I am using HitUp.';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS privacy_last_seen VARCHAR(20) DEFAULT 'everyone';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS privacy_avatar VARCHAR(20) DEFAULT 'everyone';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS privacy_about VARCHAR(20) DEFAULT 'everyone';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

	CREATE TABLE IF NOT EXISTS email_verifications (
		id SERIAL PRIMARY KEY,
		email VARCHAR(100) NOT NULL,
		code VARCHAR(10) NOT NULL,
		type VARCHAR(20) NOT NULL,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_email_verif ON email_verifications(email, type);

	CREATE TABLE IF NOT EXISTS user_blocks (
		user_id INT REFERENCES users(id) ON DELETE CASCADE,
		blocked_id INT REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(user_id, blocked_id)
	);

	CREATE TABLE IF NOT EXISTS conversations (
		id SERIAL PRIMARY KEY,
		type VARCHAR(20) NOT NULL DEFAULT 'direct',
		name VARCHAR(100),
		avatar_url TEXT DEFAULT '',
		description TEXT DEFAULT '',
		invite_code VARCHAR(64) UNIQUE,
		created_by INT REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	ALTER TABLE conversations ADD COLUMN IF NOT EXISTS avatar_url TEXT DEFAULT '';
	ALTER TABLE conversations ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '';
	ALTER TABLE conversations ADD COLUMN IF NOT EXISTS invite_code VARCHAR(64) UNIQUE;
	ALTER TABLE conversations ADD COLUMN IF NOT EXISTS created_by INT REFERENCES users(id) ON DELETE SET NULL;

	CREATE TABLE IF NOT EXISTS conversation_members (
		conversation_id INT REFERENCES conversations(id) ON DELETE CASCADE,
		user_id INT REFERENCES users(id) ON DELETE CASCADE,
		role VARCHAR(20) NOT NULL DEFAULT 'member',
		is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
		is_muted BOOLEAN NOT NULL DEFAULT FALSE,
		is_archived BOOLEAN NOT NULL DEFAULT FALSE,
		wallpaper TEXT DEFAULT '',
		joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (conversation_id, user_id)
	);
	ALTER TABLE conversation_members ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'member';
	ALTER TABLE conversation_members ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN DEFAULT FALSE;
	ALTER TABLE conversation_members ADD COLUMN IF NOT EXISTS is_muted BOOLEAN DEFAULT FALSE;
	ALTER TABLE conversation_members ADD COLUMN IF NOT EXISTS is_archived BOOLEAN DEFAULT FALSE;
	ALTER TABLE conversation_members ADD COLUMN IF NOT EXISTS wallpaper TEXT DEFAULT '';

	CREATE TABLE IF NOT EXISTS messages (
		id SERIAL PRIMARY KEY,
		conversation_id INT REFERENCES conversations(id) ON DELETE CASCADE,
		sender_id INT REFERENCES users(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		type VARCHAR(20) NOT NULL DEFAULT 'text',
		media_url TEXT DEFAULT '',
		file_name TEXT DEFAULT '',
		file_size INT DEFAULT 0,
		location_lat FLOAT DEFAULT 0,
		location_lng FLOAT DEFAULT 0,
		link_preview TEXT DEFAULT '',
		reply_to_id INT REFERENCES messages(id) ON DELETE SET NULL,
		forwarded_from_id INT REFERENCES messages(id) ON DELETE SET NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'sent',
		is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
		is_edited BOOLEAN NOT NULL DEFAULT FALSE,
		expires_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS type VARCHAR(20) DEFAULT 'text';
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_url TEXT DEFAULT '';
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_name TEXT DEFAULT '';
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_size INT DEFAULT 0;
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS location_lat FLOAT DEFAULT 0;
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS location_lng FLOAT DEFAULT 0;
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS link_preview TEXT DEFAULT '';
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_id INT REFERENCES messages(id) ON DELETE SET NULL;
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS forwarded_from_id INT REFERENCES messages(id) ON DELETE SET NULL;
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT FALSE;
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_edited BOOLEAN DEFAULT FALSE;
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE;

	CREATE TABLE IF NOT EXISTS message_deleted_for_users (
		message_id INT REFERENCES messages(id) ON DELETE CASCADE,
		user_id INT REFERENCES users(id) ON DELETE CASCADE,
		PRIMARY KEY (message_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS message_reactions (
		message_id INT REFERENCES messages(id) ON DELETE CASCADE,
		user_id INT REFERENCES users(id) ON DELETE CASCADE,
		emoji VARCHAR(10) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (message_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS starred_messages (
		user_id INT REFERENCES users(id) ON DELETE CASCADE,
		message_id INT REFERENCES messages(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(user_id, message_id)
	);

	CREATE TABLE IF NOT EXISTS polls (
		id SERIAL PRIMARY KEY,
		message_id INT REFERENCES messages(id) ON DELETE CASCADE,
		question TEXT NOT NULL,
		multiple_answers BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS poll_options (
		id SERIAL PRIMARY KEY,
		poll_id INT REFERENCES polls(id) ON DELETE CASCADE,
		option_text TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS poll_votes (
		poll_id INT REFERENCES polls(id) ON DELETE CASCADE,
		option_id INT REFERENCES poll_options(id) ON DELETE CASCADE,
		user_id INT REFERENCES users(id) ON DELETE CASCADE,
		PRIMARY KEY(poll_id, option_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS user_statuses (
		id SERIAL PRIMARY KEY,
		user_id INT REFERENCES users(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		media_url TEXT DEFAULT '',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '24 hours')
	);

	CREATE TABLE IF NOT EXISTS status_views (
		status_id INT REFERENCES user_statuses(id) ON DELETE CASCADE,
		viewer_id INT REFERENCES users(id) ON DELETE CASCADE,
		viewed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(status_id, viewer_id)
	);

	CREATE TABLE IF NOT EXISTS call_logs (
		id SERIAL PRIMARY KEY,
		caller_id INT REFERENCES users(id) ON DELETE CASCADE,
		receiver_id INT REFERENCES users(id) ON DELETE CASCADE,
		conversation_id INT REFERENCES conversations(id) ON DELETE SET NULL,
		type VARCHAR(20) NOT NULL,
		status VARCHAR(20) NOT NULL,
		duration_seconds INT DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_messages_conv_created ON messages(conversation_id, created_at DESC);
	`
	_, err := d.conn.Exec(queries)
	return err
}

// User operations
func (d *DB) CreateOrUpdateUnverifiedUser(username, email, passwordHash string) (*User, error) {
	if email == "" {
		email = username + "@hitup.local"
	}

	var existingUser User
	err := d.conn.QueryRow(`
		SELECT id, username, email, is_verified 
		FROM users 
		WHERE email = $1 OR username = $2
	`, email, username).Scan(&existingUser.ID, &existingUser.Username, &existingUser.Email, &existingUser.IsVerified)

	if err == nil {
		if existingUser.IsVerified {
			if existingUser.Email == email {
				return nil, fmt.Errorf("bu e-posta adresiyle onaylı bir hesap zaten var. Lütfen giriş yapın.")
			}
			return nil, fmt.Errorf("bu kullanıcı adı zaten kullanımda.")
		}

		updateQuery := `
			UPDATE users 
			SET username = $1, email = $2, password_hash = $3, created_at = CURRENT_TIMESTAMP 
			WHERE id = $4
			RETURNING id, username, email, is_verified, avatar_url, about, privacy_last_seen, privacy_avatar, privacy_about, created_at, last_seen_at
		`
		var u User
		err = d.conn.QueryRow(updateQuery, username, email, passwordHash, existingUser.ID).Scan(
			&u.ID, &u.Username, &u.Email, &u.IsVerified, &u.AvatarURL, &u.About, &u.PrivacyLastSeen, &u.PrivacyAvatar, &u.PrivacyAbout, &u.CreatedAt, &u.LastSeenAt,
		)
		return &u, err
	}

	query := `
		INSERT INTO users (username, email, password_hash, is_verified) 
		VALUES ($1, $2, $3, FALSE) 
		RETURNING id, username, email, is_verified, avatar_url, about, privacy_last_seen, privacy_avatar, privacy_about, created_at, last_seen_at
	`
	var u User
	err = d.conn.QueryRow(query, username, email, passwordHash).Scan(
		&u.ID, &u.Username, &u.Email, &u.IsVerified, &u.AvatarURL, &u.About, &u.PrivacyLastSeen, &u.PrivacyAvatar, &u.PrivacyAbout, &u.CreatedAt, &u.LastSeenAt,
	)
	return &u, err
}

func (d *DB) CreateVerificationCode(email, code, verifyType string, durationMinutes int) error {
	d.conn.Exec(`DELETE FROM email_verifications WHERE email = $1 AND type = $2`, email, verifyType)
	expiresAt := time.Now().Add(time.Duration(durationMinutes) * time.Minute)
	query := `INSERT INTO email_verifications (email, code, type, expires_at) VALUES ($1, $2, $3, $4)`
	_, err := d.conn.Exec(query, email, code, verifyType, expiresAt)
	return err
}

func (d *DB) VerifyCode(email, code, verifyType string) (bool, error) {
	var id int
	err := d.conn.QueryRow(`
		SELECT id FROM email_verifications 
		WHERE email = $1 AND code = $2 AND type = $3 AND expires_at > CURRENT_TIMESTAMP
	`, email, code, verifyType).Scan(&id)
	if err != nil {
		return false, nil
	}
	d.conn.Exec(`DELETE FROM email_verifications WHERE id = $1`, id)
	return true, nil
}

func (d *DB) SetUserVerified(email string) (*User, error) {
	query := `
		UPDATE users SET is_verified = TRUE WHERE email = $1 
		RETURNING id, username, email, is_verified, avatar_url, about, privacy_last_seen, privacy_avatar, privacy_about, created_at, last_seen_at
	`
	var u User
	err := d.conn.QueryRow(query, email).Scan(
		&u.ID, &u.Username, &u.Email, &u.IsVerified, &u.AvatarURL, &u.About, &u.PrivacyLastSeen, &u.PrivacyAvatar, &u.PrivacyAbout, &u.CreatedAt, &u.LastSeenAt,
	)
	return &u, err
}

func (d *DB) ResetPasswordByEmail(email, newPasswordHash string) error {
	query := `UPDATE users SET password_hash = $1 WHERE email = $2`
	_, err := d.conn.Exec(query, newPasswordHash, email)
	return err
}

func (d *DB) GetOrCreateGoogleUser(googleID, email, name, avatarURL string) (*User, error) {
	var u User
	err := d.conn.QueryRow(`
		SELECT id, username, email, google_id, is_verified, avatar_url, about, privacy_last_seen, privacy_avatar, privacy_about, created_at, last_seen_at 
		FROM users WHERE google_id = $1
	`, googleID).Scan(
		&u.ID, &u.Username, &u.Email, &u.GoogleID, &u.IsVerified, &u.AvatarURL, &u.About, &u.PrivacyLastSeen, &u.PrivacyAvatar, &u.PrivacyAbout, &u.CreatedAt, &u.LastSeenAt,
	)
	if err == nil {
		return &u, nil
	}

	err = d.conn.QueryRow(`
		SELECT id, username, email, google_id, is_verified, avatar_url, about, privacy_last_seen, privacy_avatar, privacy_about, created_at, last_seen_at 
		FROM users WHERE email = $1
	`, email).Scan(
		&u.ID, &u.Username, &u.Email, &u.GoogleID, &u.IsVerified, &u.AvatarURL, &u.About, &u.PrivacyLastSeen, &u.PrivacyAvatar, &u.PrivacyAbout, &u.CreatedAt, &u.LastSeenAt,
	)
	if err == nil {
		d.conn.Exec(`UPDATE users SET google_id = $1, is_verified = TRUE, avatar_url = COALESCE(NULLIF(avatar_url, ''), $2) WHERE id = $3`, googleID, avatarURL, u.ID)
		u.IsVerified = true
		if u.AvatarURL == "" {
			u.AvatarURL = avatarURL
		}
		return &u, nil
	}

	username := name
	if username == "" {
		username = email
	}
	var count int
	d.conn.QueryRow(`SELECT COUNT(*) FROM users WHERE username = $1`, username).Scan(&count)
	if count > 0 {
		username = fmt.Sprintf("%s_%d", username, time.Now().Unix()%10000)
	}

	query := `
		INSERT INTO users (username, email, password_hash, google_id, is_verified, avatar_url) 
		VALUES ($1, $2, '', $3, TRUE, $4) 
		RETURNING id, username, email, google_id, is_verified, avatar_url, about, privacy_last_seen, privacy_avatar, privacy_about, created_at, last_seen_at
	`
	err = d.conn.QueryRow(query, username, email, googleID, avatarURL).Scan(
		&u.ID, &u.Username, &u.Email, &u.GoogleID, &u.IsVerified, &u.AvatarURL, &u.About, &u.PrivacyLastSeen, &u.PrivacyAvatar, &u.PrivacyAbout, &u.CreatedAt, &u.LastSeenAt,
	)
	return &u, err
}

func (d *DB) GetUserByUsernameOrEmail(identifier string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, google_id, is_verified, avatar_url, about, privacy_last_seen, privacy_avatar, privacy_about, created_at, last_seen_at 
		FROM users 
		WHERE username = $1 OR email = $1
	`
	var u User
	err := d.conn.QueryRow(query, identifier).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.GoogleID, &u.IsVerified, &u.AvatarURL, &u.About, &u.PrivacyLastSeen, &u.PrivacyAvatar, &u.PrivacyAbout, &u.CreatedAt, &u.LastSeenAt,
	)
	return &u, err
}

func (d *DB) GetUserByID(userID int) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, google_id, is_verified, avatar_url, about, privacy_last_seen, privacy_avatar, privacy_about, created_at, last_seen_at 
		FROM users WHERE id = $1
	`
	var u User
	err := d.conn.QueryRow(query, userID).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.GoogleID, &u.IsVerified, &u.AvatarURL, &u.About, &u.PrivacyLastSeen, &u.PrivacyAvatar, &u.PrivacyAbout, &u.CreatedAt, &u.LastSeenAt,
	)
	return &u, err
}

func (d *DB) UpdateProfile(userID int, avatarURL, about string) error {
	query := `UPDATE users SET avatar_url = $1, about = $2 WHERE id = $3`
	_, err := d.conn.Exec(query, avatarURL, about, userID)
	return err
}

func (d *DB) UpdatePrivacy(userID int, lastSeen, avatar, about string) error {
	query := `UPDATE users SET privacy_last_seen = $1, privacy_avatar = $2, privacy_about = $3 WHERE id = $4`
	_, err := d.conn.Exec(query, lastSeen, avatar, about, userID)
	return err
}

func (d *DB) UpdateLastSeen(userID int) error {
	query := `UPDATE users SET last_seen_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := d.conn.Exec(query, userID)
	return err
}

func (d *DB) ToggleBlockUser(userID, blockedID int) (bool, error) {
	var count int
	d.conn.QueryRow(`SELECT COUNT(*) FROM user_blocks WHERE user_id = $1 AND blocked_id = $2`, userID, blockedID).Scan(&count)
	if count > 0 {
		_, err := d.conn.Exec(`DELETE FROM user_blocks WHERE user_id = $1 AND blocked_id = $2`, userID, blockedID)
		return false, err
	}
	_, err := d.conn.Exec(`INSERT INTO user_blocks (user_id, blocked_id) VALUES ($1, $2)`, userID, blockedID)
	return true, err
}

func (d *DB) IsBlocked(user1, user2 int) bool {
	var count int
	d.conn.QueryRow(`SELECT COUNT(*) FROM user_blocks WHERE (user_id = $1 AND blocked_id = $2) OR (user_id = $2 AND blocked_id = $1)`, user1, user2).Scan(&count)
	return count > 0
}

func (d *DB) ListOtherUsers(currentUserID int) ([]User, error) {
	query := `
		SELECT id, username, email, avatar_url, about, privacy_last_seen, privacy_avatar, privacy_about, created_at, last_seen_at 
		FROM users WHERE id != $1 AND is_verified = TRUE ORDER BY username ASC
	`
	rows, err := d.conn.Query(query, currentUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.AvatarURL, &u.About, &u.PrivacyLastSeen, &u.PrivacyAvatar, &u.PrivacyAbout, &u.CreatedAt, &u.LastSeenAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Conversation operations
func (d *DB) GetOrCreateDirectConversation(user1ID, user2ID int) (int, error) {
	findQuery := `
		SELECT cm1.conversation_id 
		FROM conversation_members cm1
		JOIN conversation_members cm2 ON cm1.conversation_id = cm2.conversation_id
		JOIN conversations c ON c.id = cm1.conversation_id
		WHERE cm1.user_id = $1 AND cm2.user_id = $2 AND c.type = 'direct'
		LIMIT 1;
	`
	var convID int
	err := d.conn.QueryRow(findQuery, user1ID, user2ID).Scan(&convID)
	if err == nil {
		return convID, nil
	}

	tx, err := d.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	err = tx.QueryRow(`INSERT INTO conversations (type) VALUES ('direct') RETURNING id`).Scan(&convID)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1, $2), ($1, $3)`, convID, user1ID, user2ID)
	if err != nil {
		return 0, err
	}

	return convID, tx.Commit()
}

func (d *DB) CreateGroupConversation(name, description, avatarURL string, creatorID int, memberIDs []int) (int, string, error) {
	tx, err := d.conn.Begin()
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback()

	inviteCode := fmt.Sprintf("hitup_grp_%d_%d", creatorID, time.Now().UnixNano())

	var convID int
	err = tx.QueryRow(`
		INSERT INTO conversations (type, name, description, avatar_url, invite_code, created_by) 
		VALUES ('group', $1, $2, $3, $4, $5) RETURNING id
	`, name, description, avatarURL, inviteCode, creatorID).Scan(&convID)
	if err != nil {
		return 0, "", err
	}

	_, err = tx.Exec(`INSERT INTO conversation_members (conversation_id, user_id, role) VALUES ($1, $2, 'admin')`, convID, creatorID)
	if err != nil {
		return 0, "", err
	}

	uniqueMap := map[int]bool{creatorID: true}
	for _, mID := range memberIDs {
		if !uniqueMap[mID] {
			uniqueMap[mID] = true
			_, err = tx.Exec(`INSERT INTO conversation_members (conversation_id, user_id, role) VALUES ($1, $2, 'member')`, convID, mID)
			if err != nil {
				return 0, "", err
			}
		}
	}

	return convID, inviteCode, tx.Commit()
}

func (d *DB) JoinGroupByInvite(userID int, inviteCode string) (int, error) {
	var convID int
	err := d.conn.QueryRow(`SELECT id FROM conversations WHERE invite_code = $1 AND type = 'group'`, inviteCode).Scan(&convID)
	if err != nil {
		return 0, fmt.Errorf("geçersiz davet kodu")
	}

	_, err = d.conn.Exec(`
		INSERT INTO conversation_members (conversation_id, user_id, role) 
		VALUES ($1, $2, 'member') 
		ON CONFLICT (conversation_id, user_id) DO NOTHING
	`, convID, userID)
	return convID, err
}

func (d *DB) AddGroupMember(adminID, convID, newMemberID int) error {
	var role string
	err := d.conn.QueryRow(`SELECT role FROM conversation_members WHERE conversation_id = $1 AND user_id = $2`, convID, adminID).Scan(&role)
	if err != nil || role != "admin" {
		return fmt.Errorf("yalnızca yöneticiler üye ekleyebilir")
	}

	_, err = d.conn.Exec(`INSERT INTO conversation_members (conversation_id, user_id, role) VALUES ($1, $2, 'member') ON CONFLICT DO NOTHING`, convID, newMemberID)
	return err
}

func (d *DB) RemoveGroupMember(adminID, convID, memberID int) error {
	var role string
	err := d.conn.QueryRow(`SELECT role FROM conversation_members WHERE conversation_id = $1 AND user_id = $2`, convID, adminID).Scan(&role)
	if (err != nil || role != "admin") && adminID != memberID {
		return fmt.Errorf("yetkisiz işlem")
	}

	_, err = d.conn.Exec(`DELETE FROM conversation_members WHERE conversation_id = $1 AND user_id = $2`, convID, memberID)
	return err
}

func (d *DB) TogglePinConversation(userID, convID int) (bool, error) {
	var currentPinned bool
	err := d.conn.QueryRow(`SELECT is_pinned FROM conversation_members WHERE user_id = $1 AND conversation_id = $2`, userID, convID).Scan(&currentPinned)
	if err != nil {
		return false, err
	}
	newPinned := !currentPinned
	_, err = d.conn.Exec(`UPDATE conversation_members SET is_pinned = $1 WHERE user_id = $2 AND conversation_id = $3`, newPinned, userID, convID)
	return newPinned, err
}

func (d *DB) ToggleMuteConversation(userID, convID int) (bool, error) {
	var currentMuted bool
	err := d.conn.QueryRow(`SELECT is_muted FROM conversation_members WHERE user_id = $1 AND conversation_id = $2`, userID, convID).Scan(&currentMuted)
	if err != nil {
		return false, err
	}
	newMuted := !currentMuted
	_, err = d.conn.Exec(`UPDATE conversation_members SET is_muted = $1 WHERE user_id = $2 AND conversation_id = $3`, newMuted, userID, convID)
	return newMuted, err
}

func (d *DB) ToggleArchiveConversation(userID, convID int) (bool, error) {
	var currentArchived bool
	err := d.conn.QueryRow(`SELECT is_archived FROM conversation_members WHERE user_id = $1 AND conversation_id = $2`, userID, convID).Scan(&currentArchived)
	if err != nil {
		return false, err
	}
	newArchived := !currentArchived
	_, err = d.conn.Exec(`UPDATE conversation_members SET is_archived = $1 WHERE user_id = $2 AND conversation_id = $3`, newArchived, userID, convID)
	return newArchived, err
}

func (d *DB) SetConversationWallpaper(userID, convID int, wallpaper string) error {
	query := `UPDATE conversation_members SET wallpaper = $1 WHERE user_id = $2 AND conversation_id = $3`
	_, err := d.conn.Exec(query, wallpaper, userID, convID)
	return err
}

func (d *DB) GetUserConversations(userID int) ([]ConversationListItem, error) {
	query := `
		SELECT 
			c.id, 
			c.type, 
			COALESCE(c.name, ''),
			COALESCE(c.avatar_url, ''),
			COALESCE(c.description, ''),
			COALESCE(c.invite_code, ''),
			COALESCE(last_m.content, ''),
			last_m.created_at,
			COALESCE(unread.count, 0) as unread_count,
			cm.is_pinned,
			cm.is_muted,
			cm.is_archived,
			COALESCE(cm.wallpaper, ''),
			COALESCE(other_u.id, 0),
			COALESCE(other_u.username, ''),
			COALESCE(other_u.avatar_url, ''),
			COALESCE(other_u.about, ''),
			other_u.last_seen_at
		FROM conversations c
		JOIN conversation_members cm ON cm.conversation_id = c.id AND cm.user_id = $1
		LEFT JOIN LATERAL (
			SELECT content, created_at FROM messages 
			WHERE conversation_id = c.id 
			AND id NOT IN (SELECT message_id FROM message_deleted_for_users WHERE user_id = $1)
			ORDER BY created_at DESC LIMIT 1
		) last_m ON true
		LEFT JOIN LATERAL (
			SELECT COUNT(*) as count FROM messages 
			WHERE conversation_id = c.id AND sender_id != $1 AND status != 'read'
			AND id NOT IN (SELECT message_id FROM message_deleted_for_users WHERE user_id = $1)
		) unread ON true
		LEFT JOIN LATERAL (
			SELECT u.id, u.username, u.avatar_url, u.about, u.last_seen_at 
			FROM conversation_members cm2 
			JOIN users u ON u.id = cm2.user_id 
			WHERE cm2.conversation_id = c.id AND cm2.user_id != $1 
			LIMIT 1
		) other_u ON c.type = 'direct'
		ORDER BY cm.is_pinned DESC, COALESCE(last_m.created_at, c.created_at) DESC
	`
	rows, err := d.conn.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ConversationListItem
	for rows.Next() {
		var item ConversationListItem
		var targetID int
		var targetUsername, targetAvatar, targetAbout string
		var targetLastSeen *time.Time

		err := rows.Scan(
			&item.ID, &item.Type, &item.Name, &item.AvatarURL, &item.Description, &item.InviteCode,
			&item.LastMessage, &item.LastTime, &item.UnreadCount,
			&item.IsPinned, &item.IsMuted, &item.IsArchived, &item.Wallpaper,
			&targetID, &targetUsername, &targetAvatar, &targetAbout, &targetLastSeen,
		)
		if err != nil {
			return nil, err
		}

		if item.Type == "direct" && targetID > 0 {
			item.Name = targetUsername
			item.AvatarURL = targetAvatar
			item.TargetUser = &User{
				ID:         targetID,
				Username:   targetUsername,
				AvatarURL:  targetAvatar,
				About:      targetAbout,
				LastSeenAt: targetLastSeen,
			}
		}

		list = append(list, item)
	}

	return list, rows.Err()
}

func (d *DB) GetGroupMembers(convID int) ([]User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.avatar_url, u.about, u.privacy_last_seen, u.privacy_avatar, u.privacy_about, u.created_at, u.last_seen_at
		FROM conversation_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.conversation_id = $1
		ORDER BY cm.role DESC, u.username ASC
	`
	rows, err := d.conn.Query(query, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.AvatarURL, &u.About, &u.PrivacyLastSeen, &u.PrivacyAvatar, &u.PrivacyAbout, &u.CreatedAt, &u.LastSeenAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Message operations
type SaveMessageParams struct {
	ConversationID  int
	SenderID        int
	Content         string
	Type            string
	MediaURL        string
	FileName        string
	FileSize        int
	LocationLat     float64
	LocationLng     float64
	LinkPreview     *LinkPreview
	ReplyToID       *int
	ForwardedFromID *int
	DurationSec     int
}

func (d *DB) SaveMessage(p SaveMessageParams) (*MessageRecord, error) {
	if p.Type == "" {
		p.Type = "text"
	}
	var expiresAt *time.Time
	if p.DurationSec > 0 {
		t := time.Now().Add(time.Duration(p.DurationSec) * time.Second)
		expiresAt = &t
	}

	linkPreviewStr := ""
	if p.LinkPreview != nil {
		if b, err := json.Marshal(p.LinkPreview); err == nil {
			linkPreviewStr = string(b)
		}
	}

	query := `
		INSERT INTO messages (
			conversation_id, sender_id, content, type, media_url, file_name, file_size, 
			location_lat, location_lng, link_preview, reply_to_id, forwarded_from_id, status, expires_at, created_at
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'sent', $13, CURRENT_TIMESTAMP) 
		RETURNING id, conversation_id, sender_id, content, type, media_url, file_name, file_size, 
		          location_lat, location_lng, link_preview, reply_to_id, forwarded_from_id, status, is_deleted, is_edited, expires_at, created_at
	`
	var m MessageRecord
	var scannedLinkPreview sql.NullString
	err := d.conn.QueryRow(
		query, p.ConversationID, p.SenderID, p.Content, p.Type, p.MediaURL, p.FileName, p.FileSize,
		p.LocationLat, p.LocationLng, linkPreviewStr, p.ReplyToID, p.ForwardedFromID, expiresAt,
	).Scan(
		&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Type, &m.MediaURL, &m.FileName, &m.FileSize,
		&m.LocationLat, &m.LocationLng, &scannedLinkPreview, &m.ReplyToID, &m.ForwardedFromID, &m.Status, &m.IsDeleted, &m.IsEdited, &m.ExpiresAt, &m.CreatedAt,
	)
	if err != nil {
		log.Printf("❌ [SaveMessage SQL Hatası] %v", err)
		return nil, err
	}
	if scannedLinkPreview.Valid && scannedLinkPreview.String != "" {
		var lp LinkPreview
		if json.Unmarshal([]byte(scannedLinkPreview.String), &lp) == nil {
			m.LinkPreview = &lp
		}
	}
	return &m, nil
}

func (d *DB) CreatePoll(messageID int, question string, multiple bool, options []string) (*PollDTO, error) {
	tx, err := d.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var pollID int
	err = tx.QueryRow(`INSERT INTO polls (message_id, question, multiple_answers) VALUES ($1, $2, $3) RETURNING id`, messageID, question, multiple).Scan(&pollID)
	if err != nil {
		return nil, err
	}

	var pollOpts []PollOptionDTO
	for _, optText := range options {
		var optID int
		err = tx.QueryRow(`INSERT INTO poll_options (poll_id, option_text) VALUES ($1, $2) RETURNING id`, pollID, optText).Scan(&optID)
		if err != nil {
			return nil, err
		}
		pollOpts = append(pollOpts, PollOptionDTO{ID: optID, Text: optText, VoteCount: 0})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &PollDTO{
		ID:              pollID,
		Question:        question,
		MultipleAnswers: multiple,
		Options:         pollOpts,
		TotalVotes:      0,
	}, nil
}

func (d *DB) VotePoll(pollID, optionID, userID int) (*PollDTO, error) {
	var multiple bool
	err := d.conn.QueryRow(`SELECT multiple_answers FROM polls WHERE id = $1`, pollID).Scan(&multiple)
	if err != nil {
		return nil, err
	}

	if !multiple {
		d.conn.Exec(`DELETE FROM poll_votes WHERE poll_id = $1 AND user_id = $2`, pollID, userID)
	}

	var count int
	d.conn.QueryRow(`SELECT COUNT(*) FROM poll_votes WHERE poll_id = $1 AND option_id = $2 AND user_id = $3`, pollID, optionID, userID).Scan(&count)
	if count > 0 {
		d.conn.Exec(`DELETE FROM poll_votes WHERE poll_id = $1 AND option_id = $2 AND user_id = $3`, pollID, optionID, userID)
	} else {
		d.conn.Exec(`INSERT INTO poll_votes (poll_id, option_id, user_id) VALUES ($1, $2, $3)`, pollID, optionID, userID)
	}

	return d.getPollDetails(pollID, userID)
}

func (d *DB) getPollDetails(pollID, userID int) (*PollDTO, error) {
	var p PollDTO
	p.ID = pollID
	err := d.conn.QueryRow(`SELECT question, multiple_answers FROM polls WHERE id = $1`, pollID).Scan(&p.Question, &p.MultipleAnswers)
	if err != nil {
		return nil, err
	}

	rows, err := d.conn.Query(`
		SELECT po.id, po.option_text, COUNT(pv.user_id) as vote_count,
		       EXISTS(SELECT 1 FROM poll_votes WHERE option_id = po.id AND user_id = $2) as has_voted
		FROM poll_options po
		LEFT JOIN poll_votes pv ON pv.option_id = po.id
		WHERE po.poll_id = $1
		GROUP BY po.id, po.option_text
		ORDER BY po.id ASC
	`, pollID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var opt PollOptionDTO
		if err := rows.Scan(&opt.ID, &opt.Text, &opt.VoteCount, &opt.HasVoted); err == nil {
			p.Options = append(p.Options, opt)
			p.TotalVotes += opt.VoteCount
		}
	}
	return &p, nil
}

func (d *DB) DeleteMessageForEveryone(messageID, senderID int) (*MessageRecord, error) {
	query := `
		UPDATE messages 
		SET is_deleted = TRUE, content = 'Bu mesaj silindi', media_url = '', file_name = ''
		WHERE id = $1 AND sender_id = $2
		RETURNING id, conversation_id, sender_id, content, type, media_url, status, is_deleted, is_edited, created_at
	`
	var m MessageRecord
	err := d.conn.QueryRow(query, messageID, senderID).Scan(
		&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Type, &m.MediaURL, &m.Status, &m.IsDeleted, &m.IsEdited, &m.CreatedAt,
	)
	return &m, err
}

func (d *DB) DeleteMessageForMe(messageID, userID int) error {
	query := `INSERT INTO message_deleted_for_users (message_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := d.conn.Exec(query, messageID, userID)
	return err
}

func (d *DB) EditMessage(messageID, senderID int, newContent string) (*MessageRecord, error) {
	query := `
		UPDATE messages 
		SET content = $1, is_edited = TRUE 
		WHERE id = $2 AND sender_id = $3 AND is_deleted = FALSE
		RETURNING id, conversation_id, sender_id, content, type, media_url, status, is_deleted, is_edited, created_at
	`
	var m MessageRecord
	err := d.conn.QueryRow(query, newContent, messageID, senderID).Scan(
		&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Type, &m.MediaURL, &m.Status, &m.IsDeleted, &m.IsEdited, &m.CreatedAt,
	)
	return &m, err
}

func (d *DB) ToggleStarMessage(userID, messageID int) (bool, error) {
	var count int
	d.conn.QueryRow(`SELECT COUNT(*) FROM starred_messages WHERE user_id = $1 AND message_id = $2`, userID, messageID).Scan(&count)
	if count > 0 {
		d.conn.Exec(`DELETE FROM starred_messages WHERE user_id = $1 AND message_id = $2`, userID, messageID)
		return false, nil
	}
	_, err := d.conn.Exec(`INSERT INTO starred_messages (user_id, message_id) VALUES ($1, $2)`, userID, messageID)
	return true, err
}

func (d *DB) GetStarredMessages(userID int) ([]MessageRecord, error) {
	query := `
		SELECT m.id, m.conversation_id, m.sender_id, u.username, u.avatar_url, m.content, m.type, m.media_url, m.status, m.is_deleted, m.is_edited, m.created_at
		FROM starred_messages sm
		JOIN messages m ON m.id = sm.message_id
		JOIN users u ON m.sender_id = u.id
		WHERE sm.user_id = $1 AND m.is_deleted = FALSE
		ORDER BY sm.created_at DESC
	`
	rows, err := d.conn.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageRecord
	for rows.Next() {
		var m MessageRecord
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderUsername, &m.SenderAvatar, &m.Content, &m.Type, &m.MediaURL, &m.Status, &m.IsDeleted, &m.IsEdited, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.IsStarred = true
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (d *DB) ToggleReaction(messageID, userID int, emoji string) ([]ReactionCount, error) {
	var existingEmoji string
	err := d.conn.QueryRow(`SELECT emoji FROM message_reactions WHERE message_id = $1 AND user_id = $2`, messageID, userID).Scan(&existingEmoji)
	if err == nil && existingEmoji == emoji {
		d.conn.Exec(`DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2`, messageID, userID)
	} else {
		query := `
			INSERT INTO message_reactions (message_id, user_id, emoji) VALUES ($1, $2, $3)
			ON CONFLICT (message_id, user_id) DO UPDATE SET emoji = $3, created_at = CURRENT_TIMESTAMP
		`
		d.conn.Exec(query, messageID, userID, emoji)
	}
	return d.getMessageReactions(messageID)
}

func (d *DB) getMessageReactions(messageID int) ([]ReactionCount, error) {
	rows, err := d.conn.Query(`SELECT emoji, COUNT(*) FROM message_reactions WHERE message_id = $1 GROUP BY emoji`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []ReactionCount
	for rows.Next() {
		var rc ReactionCount
		if err := rows.Scan(&rc.Emoji, &rc.Count); err == nil {
			reactions = append(reactions, rc)
		}
	}
	return reactions, nil
}

func (d *DB) GetConversationMediaGallery(convID int) ([]MessageRecord, error) {
	query := `
		SELECT id, conversation_id, sender_id, content, type, media_url, file_name, file_size, created_at
		FROM messages
		WHERE conversation_id = $1 AND type IN ('image', 'video', 'file', 'audio') AND is_deleted = FALSE
		ORDER BY created_at DESC
	`
	rows, err := d.conn.Query(query, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MessageRecord
	for rows.Next() {
		var m MessageRecord
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Type, &m.MediaURL, &m.FileName, &m.FileSize, &m.CreatedAt); err == nil {
			list = append(list, m)
		}
	}
	return list, rows.Err()
}

func (d *DB) ClearConversationHistory(convID, userID int) error {
	query := `
		INSERT INTO message_deleted_for_users (message_id, user_id)
		SELECT id, $2 FROM messages WHERE conversation_id = $1
		ON CONFLICT DO NOTHING
	`
	_, err := d.conn.Exec(query, convID, userID)
	return err
}

func (d *DB) CreateStatus(userID int, content, mediaURL string) (*UserStatusDTO, error) {
	query := `
		INSERT INTO user_statuses (user_id, content, media_url) 
		VALUES ($1, $2, $3) 
		RETURNING id, user_id, content, media_url, created_at, expires_at
	`
	var s UserStatusDTO
	err := d.conn.QueryRow(query, userID, content, mediaURL).Scan(&s.ID, &s.UserID, &s.Content, &s.MediaURL, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	d.conn.QueryRow(`SELECT username, avatar_url FROM users WHERE id = $1`, userID).Scan(&s.Username, &s.AvatarURL)
	return &s, nil
}

func (d *DB) ViewStatus(statusID, viewerID int) error {
	query := `INSERT INTO status_views (status_id, viewer_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := d.conn.Exec(query, statusID, viewerID)
	return err
}

func (d *DB) GetActiveStatuses(currentUserID int) ([]UserStatusDTO, error) {
	query := `
		SELECT s.id, s.user_id, u.username, u.avatar_url, s.content, s.media_url, s.created_at, s.expires_at
		FROM user_statuses s
		JOIN users u ON s.user_id = u.id
		WHERE s.expires_at > CURRENT_TIMESTAMP
		ORDER BY s.created_at DESC
	`
	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []UserStatusDTO
	for rows.Next() {
		var s UserStatusDTO
		if err := rows.Scan(&s.ID, &s.UserID, &s.Username, &s.AvatarURL, &s.Content, &s.MediaURL, &s.CreatedAt, &s.ExpiresAt); err != nil {
			return nil, err
		}

		if s.UserID == currentUserID {
			vRows, _ := d.conn.Query(`
				SELECT u.username FROM status_views sv JOIN users u ON u.id = sv.viewer_id WHERE sv.status_id = $1
			`, s.ID)
			if vRows != nil {
				for vRows.Next() {
					var vName string
					if vRows.Scan(&vName) == nil {
						s.Views = append(s.Views, vName)
					}
				}
				vRows.Close()
			}
		}

		list = append(list, s)
	}
	return list, rows.Err()
}

func (d *DB) LogCall(callerID, receiverID, convID int, callType, status string, duration int) error {
	query := `
		INSERT INTO call_logs (caller_id, receiver_id, conversation_id, type, status, duration_seconds, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
	`
	_, err := d.conn.Exec(query, callerID, receiverID, convID, callType, status, duration)
	return err
}

func (d *DB) GetCallLogs(userID int) ([]CallLogDTO, error) {
	query := `
		SELECT cl.id, cl.caller_id, u1.username, cl.receiver_id, u2.username, cl.type, cl.status, cl.duration_seconds, cl.created_at
		FROM call_logs cl
		JOIN users u1 ON u1.id = cl.caller_id
		JOIN users u2 ON u2.id = cl.receiver_id
		WHERE cl.caller_id = $1 OR cl.receiver_id = $1
		ORDER BY cl.created_at DESC
		LIMIT 50
	`
	rows, err := d.conn.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []CallLogDTO
	for rows.Next() {
		var l CallLogDTO
		if err := rows.Scan(&l.ID, &l.CallerID, &l.CallerName, &l.ReceiverID, &l.ReceiverName, &l.Type, &l.Status, &l.DurationSeconds, &l.CreatedAt); err == nil {
			logs = append(logs, l)
		}
	}
	return logs, rows.Err()
}

func (d *DB) CleanupExpiredMessages() ([]int, error) {
	query := `DELETE FROM messages WHERE expires_at IS NOT NULL AND expires_at <= CURRENT_TIMESTAMP RETURNING id`
	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deletedIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			deletedIDs = append(deletedIDs, id)
		}
	}
	return deletedIDs, rows.Err()
}

func (d *DB) SearchMessages(conversationID, userID int, search string) ([]MessageRecord, error) {
	query := `
		SELECT m.id, m.conversation_id, m.sender_id, u.username, u.avatar_url, m.content, m.type, m.media_url, m.status, m.is_deleted, m.is_edited, m.created_at
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.conversation_id = $1 
		AND m.is_deleted = FALSE 
		AND m.content ILIKE $2
		AND m.id NOT IN (SELECT message_id FROM message_deleted_for_users WHERE user_id = $3)
		ORDER BY m.created_at ASC
	`
	rows, err := d.conn.Query(query, conversationID, "%"+search+"%", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageRecord
	for rows.Next() {
		var m MessageRecord
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderUsername, &m.SenderAvatar, &m.Content, &m.Type, &m.MediaURL, &m.Status, &m.IsDeleted, &m.IsEdited, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Reactions, _ = d.getMessageReactions(m.ID)
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (d *DB) MarkMessagesAsRead(conversationID, readerID int) error {
	query := `UPDATE messages SET status = 'read' WHERE conversation_id = $1 AND sender_id != $2 AND status != 'read'`
	_, err := d.conn.Exec(query, conversationID, readerID)
	return err
}

func (d *DB) GetConversationMessages(conversationID, userID, limit, beforeID int) ([]MessageRecord, error) {
	var query string
	var rows *sql.Rows
	var err error

	if beforeID > 0 {
		query = `
			SELECT 
				m.id, m.conversation_id, m.sender_id, u.username, u.avatar_url, m.content, m.type, m.media_url, 
				m.file_name, m.file_size, m.location_lat, m.location_lng, m.link_preview,
				m.reply_to_id, COALESCE(rm.content, ''), COALESCE(ru.username, ''),
				m.forwarded_from_id,
				m.status, (sm.message_id IS NOT NULL) AS is_starred, m.is_deleted, m.is_edited, m.created_at
			FROM (
				SELECT * FROM messages 
				WHERE conversation_id = $1 
				AND id < $3
				AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
				AND id NOT IN (SELECT message_id FROM message_deleted_for_users WHERE user_id = $2)
				ORDER BY created_at DESC 
				LIMIT $4
			) m
			JOIN users u ON m.sender_id = u.id
			LEFT JOIN messages rm ON m.reply_to_id = rm.id
			LEFT JOIN users ru ON rm.sender_id = ru.id
			LEFT JOIN starred_messages sm ON sm.message_id = m.id AND sm.user_id = $2
			ORDER BY m.created_at ASC
		`
		rows, err = d.conn.Query(query, conversationID, userID, beforeID, limit)
	} else {
		query = `
			SELECT 
				m.id, m.conversation_id, m.sender_id, u.username, u.avatar_url, m.content, m.type, m.media_url, 
				m.file_name, m.file_size, m.location_lat, m.location_lng, m.link_preview,
				m.reply_to_id, COALESCE(rm.content, ''), COALESCE(ru.username, ''),
				m.forwarded_from_id,
				m.status, (sm.message_id IS NOT NULL) AS is_starred, m.is_deleted, m.is_edited, m.created_at
			FROM (
				SELECT * FROM messages 
				WHERE conversation_id = $1 
				AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
				AND id NOT IN (SELECT message_id FROM message_deleted_for_users WHERE user_id = $2)
				ORDER BY created_at DESC 
				LIMIT $3
			) m
			JOIN users u ON m.sender_id = u.id
			LEFT JOIN messages rm ON m.reply_to_id = rm.id
			LEFT JOIN users ru ON rm.sender_id = ru.id
			LEFT JOIN starred_messages sm ON sm.message_id = m.id AND sm.user_id = $2
			ORDER BY m.created_at ASC
		`
		rows, err = d.conn.Query(query, conversationID, userID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageRecord
	for rows.Next() {
		var m MessageRecord
		var linkPrevStr sql.NullString
		if err := rows.Scan(
			&m.ID, &m.ConversationID, &m.SenderID, &m.SenderUsername, &m.SenderAvatar, &m.Content, &m.Type, &m.MediaURL,
			&m.FileName, &m.FileSize, &m.LocationLat, &m.LocationLng, &linkPrevStr,
			&m.ReplyToID, &m.ReplyToContent, &m.ReplyToSender,
			&m.ForwardedFromID,
			&m.Status, &m.IsStarred, &m.IsDeleted, &m.IsEdited, &m.CreatedAt,
		); err != nil {
			log.Printf("❌ [GetConversationMessages Scan Hatası] %v", err)
			return nil, err
		}

		if linkPrevStr.Valid && linkPrevStr.String != "" {
			var lp LinkPreview
			if json.Unmarshal([]byte(linkPrevStr.String), &lp) == nil {
				m.LinkPreview = &lp
			}
		}

		if m.Type == "poll" {
			var pollID int
			if d.conn.QueryRow(`SELECT id FROM polls WHERE message_id = $1`, m.ID).Scan(&pollID) == nil {
				m.Poll, _ = d.getPollDetails(pollID, userID)
			}
		}

		m.Reactions, _ = d.getMessageReactions(m.ID)
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
