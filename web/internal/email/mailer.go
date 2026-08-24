package email

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/smtp"
	"os"
	"strings"
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
	if from == "" {
		from = "no-reply@hitup.app"
	}

	return &Mailer{
		smtpHost: host,
		smtpPort: port,
		smtpUser: user,
		smtpPass: pass,
		from:     from,
	}
}

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

func (m *Mailer) sendHTMLMail(toEmail, subject, htmlBody string) error {
	log.Printf("==================================================")
	log.Printf("📩 [HITUP EMAIL ISLEMI] Alıcı: %s | Konu: %s", toEmail, subject)
	log.Printf("==================================================")

	if m.smtpUser == "" || m.smtpPass == "" {
		log.Printf("⚠️ [DEV MOD] SMTP_USER veya SMTP_PASS tanimlanmadigi icin gercek e-posta gonderilmedi.")
		return nil
	}

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("HitUp <%s>", m.smtpUser)
	headers["To"] = toEmail
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n" + htmlBody)

	auth := smtp.PlainAuth("", m.smtpUser, m.smtpPass, m.smtpHost)
	addr := fmt.Sprintf("%s:%s", m.smtpHost, m.smtpPort)
	err := smtp.SendMail(addr, auth, m.smtpUser, []string{toEmail}, []byte(message.String()))
	if err != nil {
		log.Printf("❌ [SMTP HATASI] E-posta gonderilemedi: %v", err)
		return err
	}
	log.Printf("✅ [SMTP BASARILI] E-posta basariyla iletildi: %s", toEmail)
	return nil
}

func (m *Mailer) SendVerificationEmail(toEmail, username, code string) error {
	log.Printf("🔑 [DOGRULAMA KODU]: %s (Alıcı: %s)", code, toEmail)
	subject := "HitUp — E-posta Dogrulama Kodunuz: " + code
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 500px; margin: auto; padding: 24px; border: 1px solid #222e35; border-radius: 12px; background-color: #0b141a; color: #e9edef;">
			<div style="text-align: center; margin-bottom: 24px;">
				<h1 style="color: #00a884; margin: 0; font-size: 28px;">HitUp</h1>
				<p style="color: #8696a0; font-size: 13px; margin-top: 4px;">Gerçek Zamanlı Bulut Mesajlaşma</p>
			</div>
			<p style="font-size: 15px;">Merhaba <strong>%s</strong>,</p>
			<p style="font-size: 14px; color: #aebac1; line-height: 1.5;">HitUp hesabınızı doğrulamak ve oturumunuzu açmak için aşağıdaki 6 haneli güvenlik kodunu kullanın:</p>
			<div style="text-align: center; margin: 32px 0;">
				<span style="font-size: 34px; font-weight: 800; letter-spacing: 10px; color: #00a884; background: #202c33; padding: 14px 28px; border-radius: 10px; border: 1px solid #2a3942; display: inline-block;">%s</span>
			</div>
			<p style="color: #8696a0; font-size: 12px; text-align: center; margin-top: 24px; border-top: 1px solid #222e35; padding-top: 16px;">Bu kod 15 dakika boyunca geçerlidir. Eğer bu işlemi siz yapmadıysanız lütfen bu e-postayı dikkate almayın.</p>
		</div>
	`, username, code)

	return m.sendHTMLMail(toEmail, subject, body)
}

func (m *Mailer) SendPasswordResetEmail(toEmail, code string) error {
	log.Printf("🔑 [SIFRE SIFIRLAMA KODU]: %s (Alıcı: %s)", code, toEmail)
	subject := "HitUp — Sifre Sifirlama Kodunuz: " + code
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 500px; margin: auto; padding: 24px; border: 1px solid #222e35; border-radius: 12px; background-color: #0b141a; color: #e9edef;">
			<div style="text-align: center; margin-bottom: 24px;">
				<h1 style="color: #00a884; margin: 0; font-size: 28px;">HitUp</h1>
				<p style="color: #8696a0; font-size: 13px; margin-top: 4px;">Şifre Sıfırlama Talebi</p>
			</div>
			<p style="font-size: 15px;">Merhaba,</p>
			<p style="font-size: 14px; color: #aebac1; line-height: 1.5;">HitUp hesabınızın şifresini sıfırlamak için aşağıdaki 6 haneli güvenlik kodunu kullanın:</p>
			<div style="text-align: center; margin: 32px 0;">
				<span style="font-size: 34px; font-weight: 800; letter-spacing: 10px; color: #f59e0b; background: #202c33; padding: 14px 28px; border-radius: 10px; border: 1px solid #2a3942; display: inline-block;">%s</span>
			</div>
			<p style="color: #8696a0; font-size: 12px; text-align: center; margin-top: 24px; border-top: 1px solid #222e35; padding-top: 16px;">Bu kod 15 dakika boyunca geçerlidir.</p>
		</div>
	`, code)

	return m.sendHTMLMail(toEmail, subject, body)
}
