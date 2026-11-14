package mailer

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type Mailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	BaseURL  string
}

func New(host string, port int, username, password, from, baseURL string) *Mailer {
	if port <= 0 {
		port = 587
	}
	if from == "" {
		from = "E-LAYAN Desa Sabah Balau <no-reply@sabahbalau.desa.id>"
	}
	return &Mailer{
		Host:     strings.TrimSpace(host),
		Port:     port,
		Username: strings.TrimSpace(username),
		Password: password,
		From:     strings.TrimSpace(from),
		BaseURL:  strings.TrimRight(strings.TrimSpace(baseURL), "/"),
	}
}

// BuildPasswordResetMessage merakit pesan email MIME multipart/alternative (Plain Text dan HTML).
func (m *Mailer) BuildPasswordResetMessage(toEmail, username, resetURL string) ([]byte, error) {
	fromAddr, err := mail.ParseAddress(m.From)
	if err != nil {
		fromAddr = &mail.Address{Name: "E-LAYAN Desa Sabah Balau", Address: "no-reply@sabahbalau.desa.id"}
	}
	toAddr, err := mail.ParseAddress(toEmail)
	if err != nil {
		return nil, fmt.Errorf("alamat email penerima tidak valid: %w", err)
	}

	boundary := fmt.Sprintf("=_boundary_%d", time.Now().UnixNano())
	subject := "Pemulihan Kata Sandi Akun E-LAYAN Desa Sabah Balau"

	plainText := fmt.Sprintf(`Halo %s,

Kami menerima permintaan untuk mengatur ulang kata sandi akun E-LAYAN Desa Sabah Balau Anda.

Untuk membuat kata sandi baru, silakan buka tautan berikut di browser Anda:
%s

PERHATIAN KEAMANAN:
- Tautan pemulihan ini hanya berlaku selama 15 menit sejak email ini dikirim.
- Tautan ini hanya dapat digunakan satu kali.
- Jika Anda tidak pernah meminta pemulihan kata sandi, abaikan email ini. Akun Anda tetap aman.

Salam,
Tim Pelayanan Administrasi Desa Sabah Balau
Kec. Tanjung Bintang, Kab. Lampung Selatan
`, username, resetURL)

	htmlBody := fmt.Sprintf(`<!doctype html>
<html lang="id">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Pemulihan Kata Sandi</title>
</head>
<body style="margin:0;padding:0;background-color:#f1f5f9;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#0f172a;line-height:1.6;">
  <table role="presentation" width="100%%" border="0" cellspacing="0" cellpadding="0" style="background-color:#f1f5f9;padding:40px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" border="0" cellspacing="0" cellpadding="0" style="max-width:540px;background:#ffffff;border-radius:12px;border:1px solid #e2e8f0;overflow:hidden;box-shadow:0 4px 6px -1px rgba(0,0,0,0.05);">
          <!-- Header -->
          <tr>
            <td style="background-color:#15803d;padding:24px 32px;text-align:left;">
              <table role="presentation" border="0" cellspacing="0" cellpadding="0">
                <tr>
                  <td style="vertical-align:middle;padding-right:12px;">
                    <div style="background:#ffffff;color:#15803d;font-weight:900;font-size:16px;width:36px;height:36px;line-height:36px;text-align:center;border-radius:8px;">E</div>
                  </td>
                  <td style="vertical-align:middle;">
                    <div style="color:#ffffff;font-size:18px;font-weight:700;letter-spacing:0.5px;">E-LAYAN</div>
                    <div style="color:#bbf7d0;font-size:11px;font-weight:500;letter-spacing:1px;text-transform:uppercase;">Pemerintah Desa Sabah Balau</div>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <!-- Body -->
          <tr>
            <td style="padding:32px;">
              <h1 style="margin:0 0 16px;font-size:20px;font-weight:700;color:#0f172a;">Pemulihan Kata Sandi</h1>
              <p style="margin:0 0 16px;font-size:15px;color:#334155;">Halo <strong>%s</strong>,</p>
              <p style="margin:0 0 24px;font-size:14px;color:#475569;">
                Kami menerima permintaan untuk mengatur ulang kata sandi akun administrator/operator E-LAYAN Anda. Klik tombol di bawah ini untuk melanjutkan ke pembuatan kata sandi baru:
              </p>
              <!-- Action Button -->
              <table role="presentation" border="0" cellspacing="0" cellpadding="0" style="margin:0 0 28px;">
                <tr>
                  <td align="center" style="border-radius:8px;background-color:#15803d;">
                    <a href="%s" target="_blank" rel="noopener noreferrer" style="display:inline-block;padding:12px 28px;font-size:14px;font-weight:600;color:#ffffff;text-decoration:none;border-radius:8px;background-color:#15803d;border:1px solid #15803d;">
                      Atur Ulang Kata Sandi
                    </a>
                  </td>
                </tr>
              </table>
              <p style="margin:0 0 12px;font-size:12px;color:#64748b;">
                Jika tombol di atas tidak berfungsi, salin dan buka tautan berikut pada browser Anda:
              </p>
              <p style="margin:0 0 24px;font-size:12px;word-break:break-all;color:#15803d;background:#f8fafc;padding:10px 12px;border-radius:6px;border:1px solid #e2e8f0;">
                <a href="%s" style="color:#15803d;text-decoration:none;">%s</a>
              </p>
              <!-- Security Warning Box -->
              <div style="background-color:#fef2f2;border-left:4px solid #ef4444;padding:12px 16px;border-radius:4px;margin:0 0 24px;">
                <p style="margin:0 0 4px;font-size:12px;font-weight:700;color:#991b1b;">Perhatian Keamanan:</p>
                <ul style="margin:0;padding-left:18px;font-size:12px;color:#b91c1c;">
                  <li style="margin-bottom:2px;">Tautan ini hanya berlaku selama <strong>15 menit</strong>.</li>
                  <li style="margin-bottom:2px;">Tautan hanya dapat digunakan <strong>satu kali</strong>.</li>
                  <li>Jika Anda tidak meminta pemulihan ini, abaikan email ini. Akun Anda tetap aman.</li>
                </ul>
              </div>
              <p style="margin:0;font-size:13px;color:#64748b;">
                Salam hangat,<br><strong>Tim Pelayanan Desa Sabah Balau</strong>
              </p>
            </td>
          </tr>
          <!-- Footer -->
          <tr>
            <td style="background-color:#f8fafc;border-top:1px solid #e2e8f0;padding:20px 32px;text-align:center;">
              <p style="margin:0 0 4px;font-size:12px;color:#64748b;font-weight:500;">
                Sistem Informasi Pelayanan Administrasi Desa Sabah Balau
              </p>
              <p style="margin:0;font-size:11px;color:#94a3b8;">
                Kecamatan Tanjung Bintang, Kabupaten Lampung Selatan, Lampung
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, username, resetURL, resetURL, resetURL)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", fromAddr.String()))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", toAddr.String()))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	sb.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary))

	// Plain text part
	sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	sb.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	sb.WriteString(plainText)
	sb.WriteString("\r\n\r\n")

	// HTML part
	sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	sb.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	sb.WriteString(htmlBody)
	sb.WriteString("\r\n\r\n")

	sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return []byte(sb.String()), nil
}

// SendPasswordReset mengirimkan email reset password ke toEmail.
func (m *Mailer) SendPasswordReset(toEmail, username, resetURL string) error {
	msgBytes, err := m.BuildPasswordResetMessage(toEmail, username, resetURL)
	if err != nil {
		return err
	}

	// Jika SMTP_HOST tidak diset, catat ke log untuk kemudahan pengujian lokal/development.
	if m.Host == "" {
		log.Printf("[MAILER DEV SIMULATION] Reset password link untuk '%s' (%s): %s", username, toEmail, resetURL)
		return nil
	}

	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	fromAddr, err := mail.ParseAddress(m.From)
	fromEmail := m.From
	if err == nil {
		fromEmail = fromAddr.Address
	}

	// Port 465 biasanya menggunakan direct TLS / SSL
	if m.Port == 465 {
		tlsConfig := &tls.Config{
			ServerName: m.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("tls.Dial smtp %s gagal: %w", addr, err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, m.Host)
		if err != nil {
			return fmt.Errorf("smtp.NewClient gagal: %w", err)
		}
		defer client.Quit()

		if m.Username != "" && m.Password != "" {
			auth := smtp.PlainAuth("", m.Username, m.Password, m.Host)
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth gagal: %w", err)
			}
		}

		if err := client.Mail(fromEmail); err != nil {
			return fmt.Errorf("smtp MAIL FROM gagal: %w", err)
		}
		if err := client.Rcpt(toEmail); err != nil {
			return fmt.Errorf("smtp RCPT TO gagal: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("smtp DATA gagal: %w", err)
		}
		if _, err := w.Write(msgBytes); err != nil {
			_ = w.Close()
			return fmt.Errorf("smtp write gagal: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("smtp data close gagal: %w", err)
		}
		return nil
	}

	// Port 587 atau 25 dengan STARTTLS
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("koneksi smtp ke %s gagal: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.Host)
	if err != nil {
		return fmt.Errorf("smtp.NewClient gagal: %w", err)
	}
	defer client.Quit()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: m.Host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp StartTLS gagal: %w", err)
		}
	}

	if m.Username != "" && m.Password != "" {
		auth := smtp.PlainAuth("", m.Username, m.Password, m.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth gagal: %w", err)
		}
	}

	if err := client.Mail(fromEmail); err != nil {
		return fmt.Errorf("smtp MAIL FROM gagal: %w", err)
	}
	if err := client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("smtp RCPT TO gagal: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA gagal: %w", err)
	}
	if _, err := w.Write(msgBytes); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write gagal: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp data close gagal: %w", err)
	}
	return nil
}

