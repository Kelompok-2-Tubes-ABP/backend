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

// ConversationContext tracks the state of a conversation
type ConversationContext struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         string    `bson:"user_id"`
	SessionID      string    `bson:"session_id"`
	LastIntent     string    `bson:"last_intent"`
	LastEntities   ParsedEntities `bson:"last_entities"`
	LastTopic      string    `bson:"last_topic"`      // What user was discussing
	LastSubjectID  string    `bson:"last_subject_id"` // ID of the subject they were working on
	TurnCount      int       `bson:"turn_count"`
	LastMessage    string    `bson:"last_message"`
	CreatedAt      time.Time `bson:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at"`
}

// ParsedEntities stores extracted entities from last message
type ParsedEntities struct {
	Action    string  `bson:"action"`
	Amount    float64 `bson:"amount"`
	Category  string  `bson:"category"`
	Note      string  `bson:"note"`
	Time      string  `bson:"time"`
	SubjectID string  `bson:"subject_id"` // ID if they were editing something specific
}

// ConversationContextService manages conversation state per user
type ConversationContextService struct {
	collection *mongo.Collection
	cache      map[string]*ConversationContext // In-memory cache for fast access
	cacheMu    sync.RWMutex
	cacheTTL   time.Duration
}

// NewConversationContextService creates a new ConversationContextService
func NewConversationContextService(client *mongo.Client, dbName string) *ConversationContextService {
	collection := client.Database(dbName).Collection("conversation_contexts")
	return &ConversationContextService{
		collection: collection,
		cache:      make(map[string]*ConversationContext),
		cacheTTL:   30 * time.Minute,
	}
}

// cacheKey generates a unique key for user+session
func (s *ConversationContextService) cacheKey(userID, sessionID string) string {
	return userID + "::" + sessionID
}

// GetContext retrieves or creates conversation context
func (s *ConversationContextService) GetContext(userID, sessionID string) (*ConversationContext, error) {
	key := s.cacheKey(userID, sessionID)

	// Check memory cache first
	s.cacheMu.RLock()
	if ctx, ok := s.cache[key]; ok {
		if time.Since(ctx.UpdatedAt) < s.cacheTTL {
			s.cacheMu.RUnlock()
			return ctx, nil
		}
	}
	s.cacheMu.RUnlock()

	// Load from MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var conversationCtx ConversationContext
	err := s.collection.FindOne(ctx, bson.M{
		"user_id":    userID,
		"session_id": sessionID,
	}).Decode(&conversationCtx)

	if err == mongo.ErrNoDocuments {
		// Create new context
		conversationCtx = ConversationContext{
			ID:        primitive.NewObjectID(),
			UserID:    userID,
			SessionID: sessionID,
			TurnCount: 0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, err = s.collection.InsertOne(ctx, conversationCtx)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Update cache
	s.cacheMu.Lock()
	s.cache[key] = &conversationCtx
	s.cacheMu.Unlock()

	return &conversationCtx, nil
}

// UpdateContext updates the conversation context after each turn
func (s *ConversationContextService) UpdateContext(userID, sessionID string, intent string, entities ParsedEntities, message string) error {
	key := s.cacheKey(userID, sessionID)

	// Get current context
	ctx, err := s.GetContext(userID, sessionID)
	if err != nil {
		return err
	}

	// Update fields
	ctx.LastIntent = intent
	ctx.LastEntities = entities
	ctx.LastMessage = message
	ctx.TurnCount++
	ctx.UpdatedAt = time.Now()

	// Update topic if this is a new intent type
	if intent != ctx.LastIntent && intent != "ai" {
		ctx.LastTopic = intent
	}

	// Persist to MongoDB
	mongoCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = s.collection.UpdateOne(mongoCtx,
		bson.M{"user_id": userID, "session_id": sessionID},
		bson.M{"$set": ctx},
	)
	if err != nil {
		return err
	}

	// Update cache
	s.cacheMu.Lock()
	s.cache[key] = ctx
	s.cacheMu.Unlock()

	return nil
}

// GetLastIntent returns the last intent from this conversation
func (s *ConversationContextService) GetLastIntent(userID, sessionID string) (string, error) {
	ctx, err := s.GetContext(userID, sessionID)
	if err != nil {
		return "", err
	}
	return ctx.LastIntent, nil
}

// GetLastEntities returns the last extracted entities
func (s *ConversationContextService) GetLastEntities(userID, sessionID string) (*ParsedEntities, error) {
	ctx, err := s.GetContext(userID, sessionID)
	if err != nil {
		return nil, err
	}
	return &ctx.LastEntities, nil
}

// ResolveReference resolves "itu", "yang ini" references to actual context
func (s *ConversationContextService) ResolveReference(userID, sessionID, message string) (string, *ParsedEntities, error) {
	ctx, err := s.GetContext(userID, sessionID)
	if err != nil {
		return "", nil, err
	}

	// Check if message contains reference keywords
	referenceKeywords := []string{"itu", "yang ini", "yang itu", "tersebut", "dengan itu"}
	hasReference := false
	for _, kw := range referenceKeywords {
		if contains(strings.ToLower(message), kw) {
			hasReference = true
			break
		}
	}

	if !hasReference {
		return "", nil, nil
	}

	// Return the last context
	return ctx.LastTopic, &ctx.LastEntities, nil
}

// ClearContext clears conversation context (e.g., on new session)
func (s *ConversationContextService) ClearContext(userID, sessionID string) error {
	key := s.cacheKey(userID, sessionID)

	// Remove from cache
	s.cacheMu.Lock()
	delete(s.cache, key)
	s.cacheMu.Unlock()

	// Remove from MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.collection.DeleteOne(ctx, bson.M{
		"user_id":    userID,
		"session_id": sessionID,
	})

	return err
}

// GetConversationSummary returns a summary of the conversation
func (s *ConversationContextService) GetConversationSummary(userID, sessionID string) (map[string]interface{}, error) {
	ctx, err := s.GetContext(userID, sessionID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"turn_count":   ctx.TurnCount,
		"last_intent":  ctx.LastIntent,
		"last_topic":   ctx.LastTopic,
		"last_message": ctx.LastMessage,
		"created_at":   ctx.CreatedAt,
		"updated_at":   ctx.UpdatedAt,
	}, nil
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}