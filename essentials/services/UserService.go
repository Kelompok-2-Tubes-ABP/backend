package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	models "financeapi/essentials/models"
	"financeapi/essentials/utils"
	"fmt"
	"math/big"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserService struct {
	collection              *mongo.Collection
	verificationCollection  *mongo.Collection
	passwordResetCollection *mongo.Collection
}

func NewUserService(client *mongo.Client, dbName string) *UserService {
	db := client.Database(dbName)
	return &UserService{
		collection:              db.Collection("users"),
		verificationCollection:  db.Collection("email_verifications"),
		passwordResetCollection: db.Collection("password_resets"),
	}
}

func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func generateNumericToken(length int) (string, error) {
	const digits = "0123456789"
	token := make([]byte, length)
	for i := range token {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		token[i] = digits[n.Int64()]
	}
	return string(token), nil
}

func (s *UserService) CreateUser(user models.User) (models.User, error) {
	var existingByEmail models.User
	errEmail := s.collection.FindOne(context.TODO(), bson.M{"email": user.Email}).Decode(&existingByEmail)

	if errEmail == nil {
		if existingByEmail.IsActive || existingByEmail.IsEmailVerified {
			return models.User{}, errors.New("email already exists")
		}
	}

	var existingByUsername models.User
	errUsername := s.collection.FindOne(context.TODO(), bson.M{"username": user.Username}).Decode(&existingByUsername)

	if errUsername == nil {
		if existingByUsername.Email != user.Email {
			return models.User{}, errors.New("username already exists")
		}
	}

	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		return models.User{}, errors.New("failed to process password")
	}
	user.Password = hashedPassword
	user.IsActive = false
	user.IsEmailVerified = false
	user.UpdatedAt = time.Now()

	if errEmail == nil && !existingByEmail.IsActive {
		update := bson.M{
			"$set": bson.M{
				"username":            user.Username,
				"password":            user.Password,
				"updated_at":          user.UpdatedAt,
				"password_changed_at": time.Now(),
			},
		}
		_, err := s.collection.UpdateOne(context.TODO(), bson.M{"_id": existingByEmail.ID}, update)
		if err != nil {
			return models.User{}, err
		}
		user.ID = existingByEmail.ID
		return user, nil
	}

	user.CreatedAt = time.Now()
	user.PasswordChangedAt = time.Now()

	result, err := s.collection.InsertOne(context.TODO(), user)
	if err != nil {
		return models.User{}, err
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	return user, nil
}

func (s *UserService) GetUser(id primitive.ObjectID) (models.User, error) {
	var user models.User
	err := s.collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return models.User{}, errors.New("user not found")
	}
	return user, nil
}

func (s *UserService) GetUserByStringID(idStr string) (models.User, error) {
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return models.User{}, errors.New("invalid user ID")
	}
	return s.GetUser(objID)
}

func (s *UserService) GetUserByEmail(email string) (models.User, error) {
	var user models.User
	err := s.collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		return models.User{}, errors.New("user not found")
	}
	return user, nil
}

func (s *UserService) LoginUser(email string, password string) (models.User, error) {
	var user models.User
	err := s.collection.FindOne(
		context.TODO(),
		bson.M{"email": email},
	).Decode(&user)
	if err != nil {
		return models.User{}, errors.New("invalid credentials")
	}

	if !utils.CheckPassword(password, user.Password) {
		return models.User{}, errors.New("invalid credentials")
	}

	if !user.IsActive {
		return models.User{}, errors.New("account is disabled")
	}

	return user, nil
}

func (s *UserService) SendVerificationEmail(userID primitive.ObjectID) (string, string, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return "", "", err
	}

	if user.IsEmailVerified {
		return "", "", errors.New("email already verified")
	}

	randNum, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", "", errors.New("failed to generate code")
	}
	code := fmt.Sprintf("%06d", randNum.Int64())

	verification := models.VerificationRequest{
		UserID:    userID,
		Token:     code,
		Email:     user.Email,
		ExpiresAt: time.Now().Add(1 * time.Minute),
		CreatedAt: time.Now(),
	}

	s.verificationCollection.DeleteMany(context.TODO(), bson.M{"user_id": userID})

	_, err = s.verificationCollection.InsertOne(context.TODO(), verification)
	if err != nil {
		return "", "", errors.New("failed to create verification record")
	}

	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"verification_token": code}},
	)
	if err != nil {
		return "", "", errors.New("failed to update user")
	}

	return code, user.Email, nil
}

func (s *UserService) VerifyVerificationCode(code string) error {
	var verification models.VerificationRequest
	err := s.verificationCollection.FindOne(
		context.TODO(),
		bson.M{"token": code, "used_at": nil},
	).Decode(&verification)

	if err != nil {
		return errors.New("invalid or expired verification code")
	}
	if time.Now().After(verification.ExpiresAt) {
		return errors.New("verification code has expired")
	}

	now := time.Now()
	_, err = s.verificationCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": verification.ID},
		bson.M{"$set": bson.M{"used_at": &now}},
	)
	if err != nil {
		return errors.New("failed to update verification record")
	}

	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": verification.UserID},
		bson.M{"$set": bson.M{
			"is_email_verified":  true,
			"email_verified_at":  time.Now(),
			"verification_token": "",
			"is_active":          true,
		}},
	)
	if err != nil {
		return errors.New("failed to verify email")
	}

	return nil
}

func (s *UserService) VerifyEmail(token string) error {
	var verification models.VerificationRequest
	err := s.verificationCollection.FindOne(
		context.TODO(),
		bson.M{"token": token, "used_at": nil},
	).Decode(&verification)

	if err != nil {
		return errors.New("invalid or expired verification token")
	}

	if time.Now().After(verification.ExpiresAt) {
		return errors.New("verification token has expired")
	}

	now := time.Now()
	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": verification.UserID},
		bson.M{"$set": bson.M{
			"is_email_verified":  true,
			"email_verified_at":  now,
			"verification_token": "",
		}},
	)
	if err != nil {
		return errors.New("failed to verify email")
	}

	_, err = s.verificationCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": verification.ID},
		bson.M{"$set": bson.M{"used_at": now}},
	)
	if err != nil {
	}

	return nil
}

func (s *UserService) ResendVerificationEmail(email string) (string, error) {
	user, err := s.GetUserByEmail(email)
	if err != nil {
		return "", errors.New("user not found")
	}

	if user.IsEmailVerified {
		return "", errors.New("email already verified")
	}

	s.verificationCollection.DeleteMany(context.TODO(), bson.M{"user_id": user.ID})

	token, _, err := s.SendVerificationEmail(user.ID)
	return token, err
}

func (s *UserService) RequestPasswordReset(email string, ipAddress string) (string, error) {
	user, err := s.GetUserByEmail(email)
	if err != nil {
		return "", nil
	}

	if !user.IsActive {
		return "", nil
	}

	token, err := generateNumericToken(6)
	if err != nil {
		return "", errors.New("failed to generate token")
	}

	reset := models.PasswordResetRequest{
		UserID:    user.ID,
		Token:     token,
		Email:     email,
		ExpiresAt: time.Now().Add(1 * time.Minute),
		IPAddress: ipAddress,
		CreatedAt: time.Now(),
	}

	s.passwordResetCollection.DeleteMany(context.TODO(), bson.M{"user_id": user.ID})

	_, err = s.passwordResetCollection.InsertOne(context.TODO(), reset)
	if err != nil {
		return "", errors.New("failed to create reset record")
	}

	expiry := time.Now().Add(15 * time.Minute)
	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"reset_token":        token,
			"reset_token_expiry": expiry,
		}},
	)
	if err != nil {
		return "", errors.New("failed to update user")
	}

	return token, nil
}

func (s *UserService) VerifyResetToken(email string, token string) error {
	user, err := s.GetUserByEmail(email)
	if err != nil {
		return errors.New("invalid token")
	}

	if user.ResetToken != token {
		return errors.New("invalid token")
	}

	if user.ResetTokenExpiry == nil || time.Now().After(*user.ResetTokenExpiry) {
		return errors.New("token has expired")
	}

	return nil
}

func (s *UserService) ResetPassword(email string, token string, newPassword string) error {
	if err := s.VerifyResetToken(email, token); err != nil {
		return err
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return errors.New("failed to hash password")
	}

	now := time.Now()
	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"email": email},
		bson.M{"$set": bson.M{
			"password":            hashedPassword,
			"password_changed_at": now,
			"reset_token":         "",
			"reset_token_expiry":  nil,
		}},
	)
	if err != nil {
		return errors.New("failed to update password")
	}

	s.passwordResetCollection.DeleteMany(context.TODO(), bson.M{"user_id": email})

	return nil
}

func (s *UserService) ChangePassword(userID primitive.ObjectID, currentPassword string, newPassword string) error {
	user, err := s.GetUser(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if !utils.CheckPassword(currentPassword, user.Password) {
		return errors.New("current password is incorrect")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return errors.New("failed to hash password")
	}

	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{
			"password":            hashedPassword,
			"password_changed_at": time.Now(),
		}},
	)
	if err != nil {
		return errors.New("failed to update password")
	}

	return nil
}

func (s *UserService) UpdateUser(userID primitive.ObjectID, updates bson.M) (models.User, error) {
	updates["updated_at"] = time.Now()

	var user models.User
	err := s.collection.FindOneAndUpdate(
		context.TODO(),
		bson.M{"_id": userID},
		bson.M{"$set": updates},
	).Decode(&user)

	if err != nil {
		return models.User{}, errors.New("user not found")
	}

	return user, nil
}

func (s *UserService) DeleteUser(userID primitive.ObjectID) error {
	_, err := s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{
			"is_active":  false,
			"updated_at": time.Now(),
		}},
	)
	return err
}
func (s *UserService) HardDeleteUser(userID primitive.ObjectID) error {
	_, err := s.collection.DeleteOne(context.TODO(), bson.M{"_id": userID})
	return err
}
func (s *UserService) SendVerificationEmailByString(userIDStr string) (string, string, error) {
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return "", "", errors.New("invalid user id")
	}
	return s.SendVerificationEmail(userID)
}

func (s *UserService) ChangePasswordByString(userIDStr string, currentPassword string, newPassword string) error {
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return errors.New("invalid user id")
	}
	return s.ChangePassword(userID, currentPassword, newPassword)
}

// CleanupExpiredVerifications removes expired verification requests and inactive users
func (s *UserService) CleanupExpiredVerifications() error {
	now := time.Now()

	// Find all expired verification requests for inactive users
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"expires_at": bson.M{"$lt": now},
				"used_at":    nil,
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "user_id",
				"foreignField": "_id",
				"as":           "user",
			},
		},
		{
			"$match": bson.M{
				"user.is_active": false,
			},
		},
	}

	cursor, err := s.verificationCollection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	var expiredVerifications []models.VerificationRequest
	if err = cursor.All(context.TODO(), &expiredVerifications); err != nil {
		return err
	}

	// Delete expired verification requests and inactive users
	for _, verification := range expiredVerifications {
		// Delete verification request
		s.verificationCollection.DeleteOne(context.TODO(), bson.M{"_id": verification.ID})

		// Delete inactive user
		s.collection.DeleteOne(context.TODO(), bson.M{"_id": verification.UserID})
	}

	return nil
}
