package services

import (
	"context"
	"financeapi/essentials/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NotificationService struct {
	collection *mongo.Collection
}

func NewNotificationService(db *mongo.Database) *NotificationService {
	return &NotificationService{
		collection: db.Collection("user_notifications"),
	}
}

// CreateNotification creates a new in-app notification for a user
func (s *NotificationService) CreateNotification(ctx context.Context, userID primitive.ObjectID, title, message string, notifType models.NotificationType, link string) error {
	notification := models.UserNotification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Type:      notifType,
		IsRead:    false,
		Link:      link,
		CreatedAt: time.Now(),
	}

	_, err := s.collection.InsertOne(ctx, notification)
	return err
}

// GetUserNotifications gets all notifications for a specific user
func (s *NotificationService) GetUserNotifications(ctx context.Context, userID primitive.ObjectID, unreadOnly bool, limit int64) ([]models.UserNotification, error) {
	filter := bson.M{"user_id": userID}
	if unreadOnly {
		filter["is_read"] = false
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(limit)
	}

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []models.UserNotification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

// MarkAsRead marks a specific notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID primitive.ObjectID, userID primitive.ObjectID) error {
	filter := bson.M{"_id": notificationID, "user_id": userID}
	update := bson.M{"$set": bson.M{"is_read": true}}

	_, err := s.collection.UpdateOne(ctx, filter, update)
	return err
}

// MarkAllAsRead marks all notifications for a user as read
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{"user_id": userID, "is_read": false}
	update := bson.M{"$set": bson.M{"is_read": true}}

	_, err := s.collection.UpdateMany(ctx, filter, update)
	return err
}

// GetUnreadCount gets the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	filter := bson.M{"user_id": userID, "is_read": false}
	return s.collection.CountDocuments(ctx, filter)
}
