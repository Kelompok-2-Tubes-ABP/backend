package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"strings"
	"time"

	"financeapi/essentials/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EmailService struct {
	collection *mongo.Collection
	queueColl  *mongo.Collection
	smtpHost   string
	smtpPort   string
	username   string
	password   string
	fromName   string
}

func NewEmailService(db *mongo.Database) *EmailService {
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "smtp.gmail.com"
	}

	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "587"
	}

	username := os.Getenv("SMTP_EMAIL")
	if username == "" {
		username = "cryptoniac25@gmail.com"
	}

	password := os.Getenv("SMTP_APP_PASSWORD")
	if password == "" {
		log.Println("Warning: SMTP_APP_PASSWORD not set, email sending will fail")
	}

	fromName := os.Getenv("SMTP_FROM_NAME")
	if fromName == "" {
		fromName = "FinanceAPI"
	}

	return &EmailService{
		collection: db.Collection("email_logs"),
		queueColl:  db.Collection("email_queue"),
		smtpHost:   smtpHost,
		smtpPort:   smtpPort,
		username:   username,
		password:   password,
		fromName:   fromName,
	}
}

func (s *EmailService) SendEmail(to, subject, body, htmlBody string) error {
	if s.username == "" || s.password == "" {
		return fmt.Errorf("SMTP credentials not configured")
	}

	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.fromName, s.username))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if htmlBody != "" {
		boundary := generateBoundary()
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
		msg.WriteString("\r\n")

		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(body)
		msg.WriteString("\r\n\r\n")

		// HTML part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(htmlBody)
		msg.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(body)
	}

	auth := smtp.PlainAuth("", s.username, s.password, s.smtpHost)

	err := smtp.SendMail(
		s.smtpHost+":"+s.smtpPort,
		auth,
		s.username,
		[]string{to},
		[]byte(msg.String()),
	)

	if err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
		return err
	}

	log.Printf("Email sent successfully to %s", to)
	return nil
}

func (s *EmailService) SendVerificationEmail(to, code, username string) error {
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:8080"
	}

	data := map[string]interface{}{
		"username":          username,
		"verification_code": code,
	}

	subject := "Kode Verifikasi Email - FinanceAPI"
	body := s.formatVerificationBody(data)
	htmlBody := s.formatVerificationHTML(data)

	return s.SendEmail(to, subject, body, htmlBody)
}

func (s *EmailService) SendPasswordResetEmail(to, otp, username string) error {
	data := map[string]interface{}{
		"name": username,
		"otp":  otp,
	}

	subject := "Kode Reset Password - FinanceAPI"
	body := s.formatPasswordResetBody(data)
	htmlBody := s.formatPasswordResetHTML(data)

	return s.SendEmail(to, subject, body, htmlBody)
}

func (s *EmailService) SendTemplatedEmail(to string, templateType models.EmailType, data map[string]interface{}) error {
	var subject, body, htmlBody string

	switch templateType {
	case models.EmailTypeBillReminder:
		subject = s.formatString("Pengingat Tagihan: {{.BillName}}", data)
		body = s.formatBillReminderBody(data)
		htmlBody = s.formatBillReminderHTML(data)

	case models.EmailTypeDebtDue:
		subject = s.formatString("Peringatan Utang: {{.DebtName}}", data)
		body = s.formatDebtReminderBody(data)
		htmlBody = s.formatDebtReminderHTML(data)

	case models.EmailTypeGoalReached:
		subject = "Selamat! Tujuan Tabungan Tercapai 🎉"
		body = s.formatGoalReachedBody(data)
		htmlBody = s.formatGoalReachedHTML(data)

	case models.EmailTypeWeeklyReport:
		subject = "Laporan Keuangan Mingguan Anda"
		body = s.formatWeeklyReportBody(data)
		htmlBody = s.formatWeeklyReportHTML(data)

	case models.EmailTypeMonthlyReport:
		subject = "Laporan Keuangan Bulanan Anda"
		body = s.formatMonthlyReportBody(data)
		htmlBody = s.formatMonthlyReportHTML(data)

	case models.EmailTypeBudgetWarning:
		subject = s.formatString("Peringatan Anggaran: {{.Category}}", data)
		body = s.formatBudgetWarningBody(data)
		htmlBody = s.formatBudgetWarningHTML(data)

	case models.EmailTypeTransaction:
		subject = s.formatString("Transaksi: {{.Description}}", data)
		body = s.formatTransactionBody(data)
		htmlBody = s.formatTransactionHTML(data)

	case models.EmailTypeSecurityAlert:
		subject = "Peringatan Keamanan Akun"
		body = s.formatSecurityAlertBody(data)
		htmlBody = s.formatSecurityAlertHTML(data)

	case models.EmailTypeVerification:
		subject = "Verifikasi Email Anda"
		body = s.formatVerificationBody(data)
		htmlBody = s.formatVerificationHTML(data)

	case models.EmailTypePasswordReset:
		subject = "Reset Password Anda"
		body = s.formatPasswordResetBody(data)
		htmlBody = s.formatPasswordResetHTML(data)

	default:
		subject = s.formatString("Notifikasi: {{.Subject}}", data)
		body = s.formatString("{{.Message}}", data)
		htmlBody = body
	}

	return s.SendEmail(to, subject, body, htmlBody)
}

func (s *EmailService) QueueEmail(ctx context.Context, item *models.EmailQueueItem) error {
	item.ID = primitive.NewObjectID()
	item.CreatedAt = time.Now()
	item.Retries = 0
	item.MaxRetries = 3

	_, err := s.queueColl.InsertOne(ctx, item)
	return err
}

func (s *EmailService) ProcessEmailQueue(ctx context.Context) {
	cursor, err := s.queueColl.Find(ctx, bson.M{
		"status":       "pending",
		"scheduled_at": bson.M{"$lte": time.Now()},
	})
	if err != nil {
		log.Printf("Error fetching email queue: %v", err)
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var item models.EmailQueueItem
		if err := cursor.Decode(&item); err != nil {
			log.Printf("Error decoding email queue item: %v", err)
			continue
		}

		err := s.SendEmail(item.Recipient, item.Subject, item.Body, item.HTMLBody)

		status := "sent"
		if err != nil {
			status = "failed"
			item.Retries++
			if item.Retries < item.MaxRetries {
				status = "pending"
				item.ScheduledAt = time.Now().Add(5 * time.Minute)
				s.queueColl.ReplaceOne(ctx, bson.M{"_id": item.ID}, item)
			}
		}

		// Log the email
		s.LogEmail(ctx, &models.EmailLog{
			UserID:       item.UserID,
			Recipient:    item.Recipient,
			Subject:      item.Subject,
			Type:         item.Type,
			Status:       status,
			ErrorMessage: errString(err),
			SentAt:       time.Now(),
		})

		// Remove from queue if sent
		if status == "sent" {
			s.queueColl.DeleteOne(ctx, bson.M{"_id": item.ID})
		}
	}
}

func (s *EmailService) LogEmail(ctx context.Context, log *models.EmailLog) error {
	log.ID = primitive.NewObjectID()
	log.SentAt = time.Now()
	_, err := s.collection.InsertOne(ctx, log)
	return err
}

func (s *EmailService) GetEmailLogs(ctx context.Context, userID primitive.ObjectID, limit int64) ([]models.EmailLog, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"user_id": userID}, options.Find().SetSort(bson.M{"sent_at": -1}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []models.EmailLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}

func (s *EmailService) formatString(str string, data map[string]interface{}) string {
	tmpl, err := template.New("").Parse(str)
	if err != nil {
		return str
	}
	var buf bytes.Buffer
	tmpl.Execute(&buf, data)
	return buf.String()
}

func (s *EmailService) formatBillReminderBody(data map[string]interface{}) string {
	return fmt.Sprintf(`Halo %s,

Ini adalah pengingat untuk tagihan Anda:

Tagihan: %s
Jumlah: Rp %s
Tanggal Jatuh Tempo: %s
Sisa Hari: %d

Segera lakukan pembayaran untuk menghindari denda.

Salam,
FinanceAPI Team`,
		data["name"],
		data["bill_name"],
		data["amount"],
		data["due_date"],
		data["days_left"],
	)
}

func (s *EmailService) formatBillReminderHTML(data map[string]interface{}) string {
	daysLeft := data["days_left"].(int)
	urgencyColor := "#4CAF50"
	if daysLeft <= 3 {
		urgencyColor = "#FF9800"
	}
	if daysLeft <= 1 {
		urgencyColor = "#F44336"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; }
        .amount { font-size: 24px; font-weight: bold; color: #667eea; }
        .urgency { display: inline-block; padding: 5px 15px; border-radius: 20px; color: white; font-weight: bold; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Pengingat Tagihan</h1>
        </div>
        <div class="content">
            <p>Halo %s,</p>
            <p>Ini adalah pengingat untuk tagihan Anda:</p>
            <table style="width: 100%%; margin: 20px 0;">
                <tr><td style="padding: 10px 0;">Tagihan:</td><td><strong>%s</strong></td></tr>
                <tr><td style="padding: 10px 0;">Jumlah:</td><td class="amount">Rp %s</td></tr>
                <tr><td style="padding: 10px 0;">Jatuh Tempo:</td><td>%s</td></tr>
                <tr><td style="padding: 10px 0;">Sisa Hari:</td><td><span class="urgency" style="background: %s;">%d hari</span></td></tr>
            </table>
            <p>Segera lakukan pembayaran untuk menghindari denda.</p>
        </div>
        <div class="footer">
            <p>FinanceAPI - Kelola Keuanganmu dengan Bijak</p>
        </div>
    </div>
</body>
</html>`,
		data["name"], data["bill_name"], data["amount"], data["due_date"], urgencyColor, daysLeft)
}

func (s *EmailService) formatDebtReminderBody(data map[string]interface{}) string {
	return fmt.Sprintf(`Halo %s,

Peringatan utang Anda:

Utang: %s
Jumlah: Rp %s
Tanggal Jatuh Tempo: %s
Sisa Pokok: Rp %s

Harap lakukan pembayaran tepat waktu.

Salam,
FinanceAPI Team`,
		data["name"],
		data["debt_name"],
		data["amount"],
		data["due_date"],
		data["remaining_principal"],
	)
}

func (s *EmailService) formatDebtReminderHTML(data map[string]interface{}) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #f093fb 0%%, #f5576c 100%%); color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; }
        .amount { font-size: 24px; font-weight: bold; color: #f5576c; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Peringatan Utang</h1>
        </div>
        <div class="content">
            <p>Halo %s,</p>
            <p>Berikut status utang Anda:</p>
            <table style="width: 100%%; margin: 20px 0;">
                <tr><td style="padding: 10px 0;">Utang:</td><td><strong>%s</strong></td></tr>
                <tr><td style="padding: 10px 0;">Jumlah:</td><td class="amount">Rp %s</td></tr>
                <tr><td style="padding: 10px 0;">Jatuh Tempo:</td><td>%s</td></tr>
                <tr><td style="padding: 10px 0;">Sisa Pokok:</td><td>Rp %s</td></tr>
            </table>
        </div>
        <div class="footer" style="text-align: center; padding: 20px; color: #666; font-size: 12px;">
            <p>FinanceAPI</p>
        </div>
    </div>
</body>
</html>`,
		data["name"], data["debt_name"], data["amount"], data["due_date"], data["remaining_principal"])
}

func (s *EmailService) formatGoalReachedBody(data map[string]interface{}) string {
	return fmt.Sprintf(`Halo %s,

Selamat! Anda telah mencapai tujuan tabungan Anda:

Tujuan: %s
Target: Rp %s
Terkumpul: Rp %s
Tanggal Tercapai: %s

Kerja yang bagus! Sekarang Anda bisa menetapkan tujuan baru.

Salam,
FinanceAPI Team`,
		data["name"],
		data["goal_name"],
		data["target_amount"],
		data["current_amount"],
		data["achieved_date"],
	)
}

func (s *EmailService) formatGoalReachedHTML(data map[string]interface{}) string {
	progress := float64(data["progress"].(float64))
	_ = progress

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #11998e 0%%, #38ef7d 100%%); color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; text-align: center; }
        .amount { font-size: 32px; font-weight: bold; color: #11998e; }
        .progress-bar { width: 100%%; height: 20px; background: #ddd; border-radius: 10px; overflow: hidden; margin: 20px 0; }
        .progress-fill { width: 100%%; height: 100%%; background: linear-gradient(135deg, #11998e 0%%, #38ef7d 100%%); }
        .celebration { font-size: 48px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="celebration">🎉</div>
            <h1>Selamat!</h1>
            <p>Tujuan Tabungan Tercapai!</p>
        </div>
        <div class="content">
            <p>Halo %s,</p>
            <p>Kerja yang bagus! Anda telah mencapai tujuan:</p>
            <h2>%s</h2>
            <div class="progress-bar">
                <div class="progress-fill"></div>
            </div>
            <p>Target: <strong>Rp %s</strong></p>
            <p class="amount">Rp %s</p>
            <p>Tercapai pada: %s</p>
            <p>Sekarang Anda bisa menetapkan tujuan baru!</p>
        </div>
        <div class="footer" style="text-align: center; padding: 20px; color: #666; font-size: 12px;">
            <p>FinanceAPI</p>
        </div>
    </div>
</body>
</html>`,
		data["name"], data["goal_name"], data["target_amount"], data["current_amount"], data["achieved_date"])
}

func (s *EmailService) formatWeeklyReportBody(data map[string]interface{}) string {
	return fmt.Sprintf(`Halo %s,

Laporan Keuangan Mingguan Anda ( %s - %s ):

PEMASUKAN:
Total: Rp %s

PENGELUARAN:
Total: Rp %s

SALDO: Rp %s

Kategori Pengeluaran Terbesar:
%s

Analisis:
%s

Salam,
FinanceAPI Team`,
		data["name"],
		data["start_date"],
		data["end_date"],
		data["total_income"],
		data["total_expense"],
		data["balance"],
		data["top_categories"],
		data["analysis"],
	)
}

func (s *EmailService) formatWeeklyReportHTML(data map[string]interface{}) string {
	expense := data["total_expense"].(float64)
	income := data["total_income"].(float64)
	balance := income - expense

	status := "Sehat"
	_ = status // Suppress unused warning
	statusColor := "#4CAF50"
	if balance < 0 {
		status = "Perhatikan"
		statusColor = "#FF9800"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; }
        .summary { display: flex; justify-content: space-between; margin: 20px 0; }
        .card { background: white; padding: 15px; border-radius: 8px; text-align: center; flex: 1; margin: 0 5px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
        .income { color: #4CAF50; }
        .expense { color: #F44336; }
        .balance { color: %s; font-size: 18px; font-weight: bold; }
        .chart { margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Laporan Mingguan</h1>
            <p>%s - %s</p>
        </div>
        <div class="content">
            <p>Halo %s,</p>
            <div class="summary">
                <div class="card">
                    <p>Pemasukan</p>
                    <p class="income">Rp %s</p>
                </div>
                <div class="card">
                    <p>Pengeluaran</p>
                    <p class="expense">Rp %s</p>
                </div>
                <div class="card">
                    <p>Saldo</p>
                    <p class="balance">Rp %s</p>
                </div>
            </div>
            <h3>Kategori Pengeluaran Terbesar:</h3>
            <p>%s</p>
            <h3>Analisis:</h3>
            <p>%s</p>
        </div>
        <div class="footer" style="text-align: center; padding: 20px; color: #666; font-size: 12px;">
            <p>FinanceAPI</p>
        </div>
    </div>
</body>
</html>`,
		statusColor, data["start_date"], data["end_date"], data["name"],
		data["total_income"], data["total_expense"], data["balance"],
		data["top_categories"], data["analysis"])
}

func (s *EmailService) formatMonthlyReportBody(data map[string]interface{}) string {
	return fmt.Sprintf(`Halo %s,

Laporan Keuangan Bulanan Anda (%s):

RINGKASAN
Pemasukan: Rp %s
Pengeluaran: Rp %s
Saldo: Rp %s

TREN BULAN INI
vs Bulan Lalu: %s

Investasi: Rp %s
Hutang: Rp %s
Tabungan: Rp %s

Saran AI:
%s

Salam,
FinanceAPI Team`,
		data["name"],
		data["month"],
		data["total_income"],
		data["total_expense"],
		data["balance"],
		data["trend"],
		data["investments"],
		data["debts"],
		data["savings"],
		data["recommendations"],
	)
}

func (s *EmailService) formatMonthlyReportHTML(data map[string]interface{}) string {
	trend := data["trend"].(string)
	trendEmoji := "📊"
	if strings.Contains(trend, "naik") || strings.Contains(trend, "meningkat") {
		trendEmoji = "📈"
	} else if strings.Contains(trend, "turun") || strings.Contains(trend, "menurun") {
		trendEmoji = "📉"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #fa709a 0%%, #fee140 100%%); color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; }
        .stats { display: grid; grid-template-columns: 1fr 1fr; gap: 15px; margin: 20px 0; }
        .stat-box { background: white; padding: 15px; border-radius: 8px; text-align: center; }
        .stat-value { font-size: 20px; font-weight: bold; }
        .stat-label { font-size: 12px; color: #666; }
        .recommendation { background: #e3f2fd; padding: 15px; border-radius: 8px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Laporan Bulanan</h1>
            <p>%s %s</p>
        </div>
        <div class="content">
            <p>Halo %s,</p>
            <div class="stats">
                <div class="stat-box">
                    <div class="stat-value" style="color: #4CAF50;">Rp %s</div>
                    <div class="stat-label">Pemasukan</div>
                </div>
                <div class="stat-box">
                    <div class="stat-value" style="color: #F44336;">Rp %s</div>
                    <div class="stat-label">Pengeluaran</div>
                </div>
                <div class="stat-box">
                    <div class="stat-value">Rp %s</div>
                    <div class="stat-label">Saldo</div>
                </div>
                <div class="stat-box">
                    <div class="stat-value">%s %s</div>
                    <div class="stat-label">vs Bulan Lalu</div>
                </div>
            </div>
            <h3>Aset & Kewajiban</h3>
            <p>Investasi: Rp %s | Hutang: Rp %s | Tabungan: Rp %s</p>
            <div class="recommendation">
                <h4>💡 Saran AI</h4>
                <p>%s</p>
            </div>
        </div>
        <div class="footer" style="text-align: center; padding: 20px; color: #666; font-size: 12px;">
            <p>FinanceAPI</p>
        </div>
    </div>
</body>
</html>`,
		data["month"], trendEmoji, data["name"], data["total_income"],
		data["total_expense"], data["balance"], trendEmoji, trend,
		data["investments"], data["debts"], data["savings"], data["recommendations"])
}

func (s *EmailService) formatBudgetWarningBody(data map[string]interface{}) string {
	return fmt.Sprintf(`Halo %s,

Peringatan anggaran!

Kategori: %s
Budget: Rp %s
Terpakai: Rp %s
Sisa: Rp %s
Persentase: %.1f%%

Anda telah menggunakan %s dari budget bulan ini.

Salam,
FinanceAPI Team`,
		data["name"],
		data["category"],
		data["budget_amount"],
		data["spent_amount"],
		data["remaining"],
		data["percentage"],
		data["usage_level"],
	)
}

func (s *EmailService) formatBudgetWarningHTML(data map[string]interface{}) string {
	percentage := data["percentage"].(float64)
	warningColor := "#FF9800"
	if percentage >= 90 {
		warningColor = "#F44336"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #ff6b6b 0%%, #ee5a24 100%%); color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; }
        .progress-bar { width: 100%%; height: 25px; background: #ddd; border-radius: 12px; overflow: hidden; margin: 20px 0; }
        .progress-fill { height: 100%%; background: %s; transition: width 0.3s; }
        .stats { display: flex; justify-content: space-between; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚠️ Peringatan Anggaran</h1>
        </div>
        <div class="content">
            <p>Halo %s,</p>
            <p>Anda telah menggunakan:</p>
            <h2 style="color: %s;">%.1f%%</h2>
            <p>dari budget %s</p>
            <div class="progress-bar">
                <div class="progress-fill" style="width: %.1f%%;"></div>
            </div>
            <div class="stats">
                <span>Terpakai: Rp %s</span>
                <span>Sisa: Rp %s</span>
            </div>
            <p><strong>Budget:</strong> Rp %s</p>
        </div>
        <div class="footer" style="text-align: center; padding: 20px; color: #666; font-size: 12px;">
            <p>FinanceAPI</p>
        </div>
    </div>
</body>
</html>`,
		warningColor, data["name"], warningColor, percentage, data["category"],
		percentage, data["spent_amount"], data["remaining"], data["budget_amount"])
}

func (s *EmailService) formatTransactionBody(data map[string]interface{}) string {
	transactionType := data["type"].(string)
	emoji := "💰"
	if transactionType == "expense" {
		emoji = "💸"
	} else if transactionType == "income" {
		emoji = "💵"
	}

	return fmt.Sprintf(`Halo %s,

Transaksi baru tercatat:

%s %s
Kategori: %s
Jumlah: Rp %s
Tanggal: %s
Catatan: %s

Salam,
FinanceAPI Team`,
		data["name"],
		emoji,
		data["description"],
		data["category"],
		data["amount"],
		data["date"],
		data["notes"],
	)
}

func (s *EmailService) formatTransactionHTML(data map[string]interface{}) string {
	transactionType := data["type"].(string)
	amountColor := "#4CAF50"
	if transactionType == "expense" {
		amountColor = "#F44336"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; }
        .transaction { background: white; padding: 20px; border-radius: 10px; margin: 20px 0; }
        .amount { font-size: 28px; font-weight: bold; color: %s; }
        .category { display: inline-block; background: #e3f2fd; padding: 5px 15px; border-radius: 20px; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Transaksi Baru</h1>
        </div>
        <div class="content">
            <p>Halo %s,</p>
            <div class="transaction">
                <p style="font-size: 18px;">%s</p>
                <p class="category">%s</p>
                <p class="amount">Rp %s</p>
                <p>Tanggal: %s</p>
                <p>Catatan: %s</p>
            </div>
        </div>
        <div class="footer" style="text-align: center; padding: 20px; color: #666; font-size: 12px;">
            <p>FinanceAPI</p>
        </div>
    </div>
</body>
</html>`,
		amountColor, data["name"], data["description"], data["category"],
		data["amount"], data["date"], data["notes"])
}

func (s *EmailService) formatSecurityAlertBody(data map[string]interface{}) string {
	return fmt.Sprintf(`Halo %s,

Peringatan keamanan untuk akun Anda:

Peristiwa: %s
Waktu: %s
Lokasi: %s
Perangkat: %s

Jika ini bukan Anda, segera amankan akun Anda dengan:
1. Mengubah password
2. Mengaktifkan 2FA
3. Memeriksa aktivitas login

Salam,
FinanceAPI Team`,
		data["name"],
		data["event"],
		data["time"],
		data["location"],
		data["device"],
	)
}

func (s *EmailService) formatSecurityAlertHTML(data map[string]interface{}) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #F44336 0%%, #FF5722 100%%); color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; }
        .alert-box { background: #ffebee; border: 2px solid #F44336; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .warning-text { color: #F44336; font-weight: bold; }
        .details { background: white; padding: 15px; border-radius: 8px; margin: 15px 0; }
        .btn { display: inline-block; background: #F44336; color: white; padding: 12px 30px; border-radius: 5px; text-decoration: none; margin-top: 15px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚠️ Peringatan Keamanan</h1>
        </div>
        <div class="content">
            <p>Halo %s,</p>
            <div class="alert-box">
                <p class="warning-text">Kami mendeteksi aktivitas mencurigakan di akun Anda</p>
            </div>
            <div class="details">
                <p><strong>Peristiwa:</strong> %s</p>
                <p><strong>Waktu:</strong> %s</p>
                <p><strong>Lokasi:</strong> %s</p>
                <p><strong>Perangkat:</strong> %s</p>
            </div>
            <p>Jika ini bukan Anda, segera amankan akun Anda:</p>
            <a href="#" class="btn">Amankan Akun Saya</a>
        </div>
        <div class="footer" style="text-align: center; padding: 20px; color: #666; font-size: 12px;">
            <p>FinanceAPI</p>
        </div>
    </div>
</body>
</html>`,
		data["name"], data["event"], data["time"], data["location"], data["device"])
}

func (s *EmailService) formatVerificationBody(data map[string]interface{}) string {
	username := ""
	if v, ok := data["username"].(string); ok {
		username = v
	}
	code := ""
	if v, ok := data["verification_code"].(string); ok {
		code = v
	}

	return fmt.Sprintf(`Halo %s,

Terima kasih telah mendaftar di FinanceAPI.

Kode verifikasi email Anda: %s

Kode ini berlaku selama 24 jam.

Salam,
FinanceAPI Team`,
		username,
		code,
	)
}

func (s *EmailService) formatVerificationHTML(data map[string]interface{}) string {
	username := ""
	if v, ok := data["username"].(string); ok {
		username = v
	}
	code := ""
	if v, ok := data["verification_code"].(string); ok {
		code = v
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; text-align: center; }
        .code { font-size: 36px; font-weight: bold; letter-spacing: 10px; color: #667eea; background: #f0f0f0; padding: 20px; border-radius: 8px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Verifikasi Email</h1>
        </div>
        <div class="content">
            <p>Halo %s,</p>
            <p>Terima kasih telah mendaftar di FinanceAPI.</p>
            <p>Kode verifikasi Anda:</p>
            <div class="code">%s</div>
            <p>Kode ini berlaku selama 24 jam.</p>
        </div>
        <div class="footer" style="text-align: center; padding: 20px; color: #666; font-size: 12px;">
            <p>FinanceAPI</p>
        </div>
    </div>
</body>
</html>`,
		username, code)
}

func (s *EmailService) formatPasswordResetBody(data map[string]interface{}) string {
	return fmt.Sprintf(`Halo %s,

Anda meminta reset password.

Kode verifikasi Anda: %s

Kode ini berlaku selama 15 menit.

Jika Anda tidak meminta reset password, abaikan email ini.

Salam,
FinanceAPI Team`,
		data["name"],
		data["otp"],
	)
}

func (s *EmailService) formatPasswordResetHTML(data map[string]interface{}) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; text-align: center; }
        .code { display: inline-block; background: #fff; padding: 15px 30px; border-radius: 8px; font-size: 32px; font-weight: bold; letter-spacing: 8px; margin: 20px 0; border: 2px solid #667eea; }
        .warning { background: #fff3cd; padding: 15px; border-radius: 8px; color: #856404; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Reset Password</h1>
        </div>
        <div class="content">
            <p>Halo %s,</p>
            <p>Anda meminta reset password. Berikut kode verifikasi Anda:</p>
            <div class="code">%s</div>
            <p>Kode ini berlaku selama 15 menit.</p>
            <div class="warning">
                Jangan berikan kode ini kepada siapa pun. Jika Anda tidak meminta reset password, abaikan email ini.
            </div>
        </div>
        <div class="footer" style="text-align: center; padding: 20px; color: #666; font-size: 12px;">
            <p>FinanceAPI</p>
        </div>
    </div>
</body>
</html>`,
		data["name"], data["otp"])
}

// Helper functions
func generateBoundary() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func errString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
