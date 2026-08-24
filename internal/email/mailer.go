package email

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/smtp"
	"os"
)

type Mailer struct {
	smtpHost string
	smtpPort string
	smtpUser string
	smtpPass string
	from     string
}

func NewMailer() *Mailer {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		host = "smtp.gmail.com"
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = user
	}

	return &Mailer{
		smtpHost: host,
		smtpPort: port,
		smtpUser: user,
		smtpPass: pass,
		from:     from,
	}
}

// GenerateOTP, 6 haneli rastgele bir doğrulama kodu üretir.
func GenerateOTP() string {
	var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}
	b := make([]byte, 6)
	n, err := io.ReadAtLeast(rand.Reader, b, 6)
	if n != 6 || err != nil {
		return "123456"
	}
	for i := 0; i < len(b); i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b)
}

// SendVerificationEmail, e-posta onaylama OTP kodunu gönderir.
func (m *Mailer) SendVerificationEmail(toEmail, username, code string) error {
	subject := "Subject: HitUp — E-posta Dogrulama Kodunuz\r\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 500px; margin: auto; padding: 20px; border: 1px solid #e2e8f0; border-radius: 10px; background-color: #0b141a; color: #e9edef;">
			<div style="text-align: center; margin-bottom: 20px;">
				<h1 style="color: #00a884; margin: 0;">HitUp</h1>
				<p style="color: #8696a0; font-size: 13px;">Gerçek Zamanlı Mesajlaşma</p>
			</div>
			<p>Merhaba <strong>%s</strong>,</p>
			<p>HitUp hesabınızı doğrulamak için aşağıdaki 6 haneli güvenlik kodunu kullanın:</p>
			<div style="text-align: center; margin: 30px 0;">
				<span style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #00a884; background: #202c33; padding: 12px 24px; border-radius: 8px; border: 1px solid #2a3942;">%s</span>
			</div>
			<p style="color: #8696a0; font-size: 12px; text-align: center;">Bu kod 15 dakika boyunca geçerlidir. Eğer bu işlemi siz yapmadıysanız lütfen bu e-postayı dikkate almayın.</p>
		</div>
	`, username, code)

	msg := []byte(subject + mime + body)

	// Eğer SMTP ayarlanmamışsa geliştirme ortamında konsola bas
	if m.smtpUser == "" || m.smtpPass == "" {
		log.Printf("[DEV EMAIL] SMTP yapilandirilmamis. Alıcı: %s, Kod: %s", toEmail, code)
		return nil
	}

	auth := smtp.PlainAuth("", m.smtpUser, m.smtpPass, m.smtpHost)
	addr := fmt.Sprintf("%s:%s", m.smtpHost, m.smtpPort)
	return smtp.SendMail(addr, auth, m.from, []string{toEmail}, msg)
}

// SendPasswordResetEmail, şifre sıfırlama OTP kodunu gönderir.
func (m *Mailer) SendPasswordResetEmail(toEmail, code string) error {
	subject := "Subject: HitUp — Sifre Sifirlama Kodu\r\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 500px; margin: auto; padding: 20px; border: 1px solid #e2e8f0; border-radius: 10px; background-color: #0b141a; color: #e9edef;">
			<div style="text-align: center; margin-bottom: 20px;">
				<h1 style="color: #00a884; margin: 0;">HitUp</h1>
			</div>
			<p>Şifrenizi sıfırlamak için aşağıdaki tek kullanımlık güvenlik kodunu kullanın:</p>
			<div style="text-align: center; margin: 30px 0;">
				<span style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #f59e0b; background: #202c33; padding: 12px 24px; border-radius: 8px; border: 1px solid #2a3942;">%s</span>
			</div>
			<p style="color: #8696a0; font-size: 12px; text-align: center;">Bu kod 15 dakika boyunca geçerlidir.</p>
		</div>
	`, code)

	msg := []byte(subject + mime + body)

	if m.smtpUser == "" || m.smtpPass == "" {
		log.Printf("[DEV RESET EMAIL] Alıcı: %s, Kod: %s", toEmail, code)
		return nil
	}

	auth := smtp.PlainAuth("", m.smtpUser, m.smtpPass, m.smtpHost)
	addr := fmt.Sprintf("%s:%s", m.smtpHost, m.smtpPort)
	return smtp.SendMail(addr, auth, m.from, []string{toEmail}, msg)
}
