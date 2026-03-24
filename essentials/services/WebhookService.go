package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"financeapi/essentials/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// WebhookService handles webhook business logic
type WebhookService struct {
	collection     *mongo.Collection
	logCollection  *mongo.Collection
	prefCollection *mongo.Collection
	client         *mongo.Client
}

// NewWebhookService creates a new webhook service
func NewWebhookService(db *mongo.Database) *WebhookService {
	return &WebhookService{
		collection:     db.Collection("webhooks"),
		logCollection:  db.Collection("webhook_logs"),
		prefCollection: db.Collection("notification_preferences"),
		client:         db.Client(),
	}
}

// CreateWebhook creates a new webhook
func (s *WebhookService) CreateWebhook(ctx context.Context, webhook *models.Webhook) error {
	webhook.ID = primitive.NewObjectID()
	_, err := s.collection.InsertOne(ctx, webhook)
	return err
}

// GetWebhooksByUser retrieves all webhooks for a user
func (s *WebhookService) GetWebhooksByUser(ctx context.Context, userID primitive.ObjectID) ([]models.Webhook, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var webhooks []models.Webhook
	if err := cursor.All(ctx, &webhooks); err != nil {
		return nil, err
	}

	return webhooks, nil
}

// GetWebhook retrieves a specific webhook
func (s *WebhookService) GetWebhook(ctx context.Context, id primitive.ObjectID) (*models.Webhook, error) {
	var webhook models.Webhook
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&webhook)
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

// UpdateWebhook updates a webhook
func (s *WebhookService) UpdateWebhook(ctx context.Context, webhook *models.Webhook) error {
	_, err := s.collection.ReplaceOne(ctx, bson.M{"_id": webhook.ID}, webhook)
	return err
}

// DeleteWebhook deletes a webhook
func (s *WebhookService) DeleteWebhook(ctx context.Context, id, userID primitive.ObjectID) error {
	result, err := s.collection.DeleteOne(ctx, bson.M{
		"_id":     id,
		"user_id": userID,
	})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrWebhookNotFound
	}
	return nil
}

// TriggerWebhook sends a webhook request
func (s *WebhookService) TriggerWebhook(url, secret, event string, payload []byte) (int, string, error) {
	// Generate signature
	signature := generateHMACSignature(secret, payload)

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", event)
	req.Header.Set("X-Webhook-Timestamp", time.Now().UTC().Format(time.RFC3339))

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	return resp.StatusCode, buf.String(), nil
}

// LogWebhookTrigger logs a webhook trigger
func (s *WebhookService) LogWebhookTrigger(ctx context.Context, log *models.WebhookLog) error {
	log.ID = primitive.NewObjectID()
	_, err := s.logCollection.InsertOne(ctx, log)
	return err
}

// GetNotificationPreferences retrieves notification preferences
func (s *WebhookService) GetNotificationPreferences(ctx context.Context, userID primitive.ObjectID) (*models.UserNotificationPreference, error) {
	var pref models.UserNotificationPreference
	err := s.prefCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&pref)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default preferences
			return &models.UserNotificationPreference{
				UserID:         userID,
				EmailEnabled:   true,
				BillReminders:  true,
				DebtAlerts:     true,
				GoalUpdates:    true,
				WeeklyReports:  true,
				MonthlyReports: false,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}, nil
		}
		return nil, err
	}
	return &pref, nil
}

// UpdateNotificationPreferences updates notification preferences
func (s *WebhookService) UpdateNotificationPreferences(ctx context.Context, pref *models.UserNotificationPreference) error {
	filter := bson.M{"user_id": pref.UserID}
	update := bson.M{"$set": pref}
	opts := options.Update().SetUpsert(true)
	_, err := s.prefCollection.UpdateOne(ctx, filter, update, opts)
	return err
}

// TriggerEvent triggers all webhooks subscribed to an event
func (s *WebhookService) TriggerEvent(ctx context.Context, event string, data map[string]interface{}) {
	webhooks, err := s.GetWebhooksByEvent(ctx, event)
	if err != nil {
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"event":     event,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})

	for _, webhook := range webhooks {
		go func(w models.Webhook) {
			statusCode, response, err := s.TriggerWebhook(w.URL, w.Secret, event, payload)

			s.LogWebhookTrigger(ctx, &models.WebhookLog{
				WebhookID:   w.ID,
				Event:       event,
				Payload:     string(payload),
				StatusCode:  statusCode,
				Success:     err == nil,
				Response:    response,
				TriggeredAt: time.Now(),
			})

			// Update webhook stats
			if err == nil {
				w.SuccessCount++
			} else {
				w.FailCount++
			}
			w.LastTriggered = func() *time.Time { t := time.Now(); return &t }()
			s.UpdateWebhook(ctx, &w)
		}(webhook)
	}
}

// GetWebhooksByEvent retrieves all active webhooks for an event
func (s *WebhookService) GetWebhooksByEvent(ctx context.Context, event string) ([]models.Webhook, error) {
	cursor, err := s.collection.Find(ctx, bson.M{
		"events":    event,
		"is_active": true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var webhooks []models.Webhook
	if err := cursor.All(ctx, &webhooks); err != nil {
		return nil, err
	}

	return webhooks, nil
}

// Helper functions
func GenerateSecret() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "default-secret-" + time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}

func generateHMACSignature(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// Error definitions
var ErrWebhookNotFound = &WebhookError{Message: "Webhook not found"}

type WebhookError struct {
	Message string
}

func (e *WebhookError) Error() string {
	return e.Message
}
