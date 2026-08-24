package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GoogleTokenInfo struct {
	Sub           string `json:"sub"` // Google Unique User ID
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	ErrorDesc     string `json:"error_description,omitempty"`
}

// VerifyGoogleIDToken, Google'dan dönen credential (JWT ID Token) dizesini doğrular.
func VerifyGoogleIDToken(idToken string) (*GoogleTokenInfo, error) {
	client := http.Client{Timeout: 5 * time.Second}
	verifyURL := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)

	resp, err := client.Get(verifyURL)
	if err != nil {
		return nil, fmt.Errorf("google doğrulama servisine ulaşılamadı: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geçersiz google token'ı")
	}

	var tokenInfo GoogleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("token yanıtı çözülemedi: %w", err)
	}

	if tokenInfo.Email == "" || tokenInfo.Sub == "" {
		return nil, fmt.Errorf("eksik kullanıcı bilgisi")
	}

	return &tokenInfo, nil
}
