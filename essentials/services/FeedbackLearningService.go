package services

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// FeedbackEntry represents a user correction that the system learns from
type FeedbackEntry struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	UserID          string            `bson:"user_id"`
	OriginalMessage string            `bson:"original_message"`
	NormalizedMsg   string           `bson:"normalized_message"` // lowercase, trimmed
	WrongIntent     string           `bson:"wrong_intent"`
	CorrectIntent   string           `bson:"correct_intent"`
	Priority        int              `bson:"priority"`        // how many times corrected
	Context         string           `bson:"context"`        // optional context hint
	CreatedAt       time.Time        `bson:"created_at"`
	UpdatedAt       time.Time        `bson:"updated_at"`
}

// FeedbackLearningService learns from user corrections
type FeedbackLearningService struct {
	collection *mongo.Collection
	cache      map[string]*FeedbackEntry // in-memory cache for fast lookup
	cacheMu    sync.RWMutex
	cacheTTL   time.Duration
}

// NewFeedbackLearningService creates a new FeedbackLearningService
func NewFeedbackLearningService(client *mongo.Client, dbName string) *FeedbackLearningService {
	collection := client.Database(dbName).Collection("chatbot_feedback")
	return &FeedbackLearningService{
		collection: collection,
		cache:      make(map[string]*FeedbackEntry),
		cacheTTL:   1 * time.Hour,
	}
}

// normalizeMessage creates a searchable key from message
func normalizeMessage(msg string) string {
	// Lowercase, trim spaces, remove extra whitespace
	msg = strings.ToLower(msg)
	msg = strings.TrimSpace(msg)
	msg = strings.Join(strings.Fields(msg), " ")
	return msg
}

// LearnFeedback stores a user correction
func (s *FeedbackLearningService) LearnFeedback(userID, originalMsg, wrongIntent, correctIntent, contextHint string) error {
	normalized := normalizeMessage(originalMsg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if this feedback already exists
	var existing FeedbackEntry
	err := s.collection.FindOne(ctx, bson.M{
		"user_id":           userID,
		"normalized_message": normalized,
	}).Decode(&existing)

	if err == mongo.ErrNoDocuments {
		// Create new feedback entry
		entry := FeedbackEntry{
			ID:              primitive.NewObjectID(),
			UserID:          userID,
			OriginalMessage: originalMsg,
			NormalizedMsg:   normalized,
			WrongIntent:     wrongIntent,
			CorrectIntent:   correctIntent,
			Priority:        1,
			Context:         contextHint,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		_, err = s.collection.InsertOne(ctx, entry)
	} else if err == nil {
		// Update existing - increment priority
		entry := FeedbackEntry{
			ID:            existing.ID,
			UserID:        userID,
			OriginalMessage:    originalMsg,
			NormalizedMsg: normalized,
			WrongIntent:   wrongIntent,
			CorrectIntent: correctIntent,
			Priority:      existing.Priority + 1,
			Context:       contextHint,
			CreatedAt:     existing.CreatedAt,
			UpdatedAt:     time.Now(),
		}
		_, err = s.collection.ReplaceOne(ctx, bson.M{"_id": existing.ID}, entry)
	}

	// Update cache
	s.cacheMu.Lock()
	s.cache[userID+"::"+normalized] = &FeedbackEntry{
		NormalizedMsg: normalized,
		CorrectIntent: correctIntent,
		Priority:     1,
	}
	s.cacheMu.Unlock()

	return err
}

// GetCorrectedIntent checks if there's a learned correction for this message
func (s *FeedbackLearningService) GetCorrectedIntent(userID, message string) (string, bool) {
	normalized := normalizeMessage(message)

	// Check cache first
	s.cacheMu.RLock()
	if entry, ok := s.cache[userID+"::"+normalized]; ok {
		s.cacheMu.RUnlock()
		return entry.CorrectIntent, true
	}
	s.cacheMu.RUnlock()

	// Load from database
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var entry FeedbackEntry
	err := s.collection.FindOne(ctx, bson.M{
		"user_id":           userID,
		"normalized_message": normalized,
	}).Decode(&entry)

	if err == nil {
		// Cache the result
		s.cacheMu.Lock()
		s.cache[userID+"::"+normalized] = &entry
		s.cacheMu.Unlock()
		return entry.CorrectIntent, true
	}

	return "", false
}

// GetPartialMatch checks if any learned message is a partial match
func (s *FeedbackLearningService) GetPartialMatch(userID, message string) (string, bool) {
	normalized := normalizeMessage(message)
	words := strings.Fields(normalized)

	if len(words) < 2 {
		return "", false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find entries where all words match (order doesn't matter)
	// Build a regex that matches all words
	pattern := ""
	for i, word := range words {
		if i > 0 {
			pattern += ".*"
		}
		pattern += word
	}

	cursor, err := s.collection.Find(ctx, bson.M{
		"user_id": userID,
		"normalized_message": bson.M{"$regex": pattern, "$options": "i"},
	})
	if err != nil {
		return "", false
	}
	defer cursor.Close(ctx)

	var bestMatch *FeedbackEntry
	bestPriority := 0

	var entries []FeedbackEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return "", false
	}

	for _, entry := range entries {
		if entry.Priority > bestPriority {
			bestPriority = entry.Priority
			bestMatch = &entry
		}
	}

	if bestMatch != nil && bestPriority >= 1 {
		return bestMatch.CorrectIntent, true
	}

	return "", false
}

// GetAllFeedback returns all learned corrections for a user
func (s *FeedbackLearningService) GetAllFeedback(userID string) ([]FeedbackEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.M{
		"user_id": userID,
	}, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var entries []FeedbackEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// ClearFeedback removes all learned feedback for a user
func (s *FeedbackLearningService) ClearFeedback(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.collection.DeleteMany(ctx, bson.M{"user_id": userID})

	// Clear cache
	s.cacheMu.Lock()
	for k := range s.cache {
		if strings.HasPrefix(k, userID+"::") {
			delete(s.cache, k)
		}
	}
	s.cacheMu.Unlock()

	return err
}

// CleanupOldEntries removes entries older than 30 days
func (s *FeedbackLearningService) CleanupOldEntries() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cutoff := time.Now().AddDate(0, 0, -30)
	_, err := s.collection.DeleteMany(ctx, bson.M{
		"created_at": bson.M{"$lt": cutoff},
	})

	return err
}
