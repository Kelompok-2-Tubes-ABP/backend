package services

import (
	"context"
	"errors"
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
)

type SavingsGoalService struct {
	collection      *mongo.Collection
	contributionCol *mongo.Collection
}

func NewSavingsGoalService(client *mongo.Client, dbName string) *SavingsGoalService {
	db := client.Database(dbName)
	return &SavingsGoalService{
		collection:      db.Collection("savings_goals"),
		contributionCol: db.Collection("savings_contributions"),
	}
}

func (s *SavingsGoalService) CreateSavingsGoal(goal models.SavingsGoal) (models.SavingsGoal, error) {
	goal.Name = utils.SanitizeMongoValue(goal.Name)
	goal.Description = utils.SanitizeMongoValue(goal.Description)
	goal.Category = utils.SanitizeMongoValue(goal.Category)

	if goal.TargetAmount <= 0 {
		return models.SavingsGoal{}, errors.New("target amount must be greater than 0")
	}

	if goal.TargetDate.Before(time.Now()) {
		return models.SavingsGoal{}, errors.New("target date must be in the future")
	}

	goal.CurrentAmount = 0
	goal.Status = "active"
	goal.CreatedAt = time.Now()
	goal.UpdatedAt = time.Now()

	result, err := s.collection.InsertOne(context.TODO(), goal)
	if err != nil {
		return models.SavingsGoal{}, err
	}

	goal.ID = result.InsertedID.(primitive.ObjectID)
	return goal, nil
}

func (s *SavingsGoalService) GetSavingsGoal(goalID primitive.ObjectID, userID string) (models.SavingsGoal, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return models.SavingsGoal{}, errors.New("invalid user ID")
	}

	var goal models.SavingsGoal
	err = s.collection.FindOne(context.TODO(), bson.M{
		"_id":     goalID,
		"user_id": objID,
	}).Decode(&goal)

	if err != nil {
		return models.SavingsGoal{}, errors.New("savings goal not found")
	}

	return goal, nil
}

func (s *SavingsGoalService) GetUserSavingsGoals(userID string) ([]models.SavingsGoal, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	cursor, err := s.collection.Find(context.TODO(), bson.M{"user_id": objID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var goals []models.SavingsGoal
	if err := cursor.All(context.TODO(), &goals); err != nil {
		return nil, err
	}

	return goals, nil
}

func (s *SavingsGoalService) UpdateSavingsGoal(goalID primitive.ObjectID, userID string, updates bson.M) (models.SavingsGoal, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return models.SavingsGoal{}, errors.New("invalid user ID")
	}

	if name, ok := updates["name"].(string); ok {
		updates["name"] = utils.SanitizeMongoValue(name)
	}
	if desc, ok := updates["description"].(string); ok {
		updates["description"] = utils.SanitizeMongoValue(desc)
	}

	updates["updated_at"] = time.Now()

	opts := mongoOptions.FindOneAndUpdate().SetReturnDocument(mongoOptions.After)
	result := s.collection.FindOneAndUpdate(
		context.TODO(),
		bson.M{"_id": goalID, "user_id": objID},
		bson.M{"$set": updates},
		opts,
	)

	var goal models.SavingsGoal
	if err := result.Decode(&goal); err != nil {
		return models.SavingsGoal{}, errors.New("savings goal not found")
	}

	return goal, nil
}

func (s *SavingsGoalService) AddContribution(contribution models.SavingsContribution) (models.SavingsContribution, error) {
	if contribution.Amount <= 0 {
		return models.SavingsContribution{}, errors.New("contribution amount must be greater than 0")
	}

	contribution.Date = time.Now()
	contribution.CreatedAt = time.Now()

	result, err := s.contributionCol.InsertOne(context.TODO(), contribution)
	if err != nil {
		return models.SavingsContribution{}, err
	}

	contribution.ID = result.InsertedID.(primitive.ObjectID)

	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": contribution.SavingsGoalID},
		bson.M{
			"$inc": bson.M{"current_amount": contribution.Amount},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)

	if err != nil {
		return models.SavingsContribution{}, err
	}

	var goal models.SavingsGoal
	s.collection.FindOne(context.TODO(), bson.M{"_id": contribution.SavingsGoalID}).Decode(&goal)

	if goal.CurrentAmount >= goal.TargetAmount {
		s.collection.UpdateOne(
			context.TODO(),
			bson.M{"_id": contribution.SavingsGoalID},
			bson.M{"$set": bson.M{"status": "completed", "updated_at": time.Now()}},
		)
	}

	return contribution, nil
}

func (s *SavingsGoalService) GetGoalContributions(goalID primitive.ObjectID, userID string) ([]models.SavingsContribution, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	cursor, err := s.contributionCol.Find(context.TODO(), bson.M{
		"savings_goal_id": goalID,
		"user_id":         objID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var contributions []models.SavingsContribution
	if err := cursor.All(context.TODO(), &contributions); err != nil {
		return nil, err
	}

	return contributions, nil
}

func (s *SavingsGoalService) DeleteSavingsGoal(goalID primitive.ObjectID, userID string) error {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = s.contributionCol.DeleteMany(context.TODO(), bson.M{
		"savings_goal_id": goalID,
		"user_id":         objID,
	})
	if err != nil {
		return err
	}

	result, err := s.collection.DeleteOne(context.TODO(), bson.M{
		"_id":     goalID,
		"user_id": objID,
	})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("savings goal not found")
	}

	return nil
}

func (s *SavingsGoalService) GetSavingsSummary(userID string) (map[string]interface{}, error) {
	goals, err := s.GetUserSavingsGoals(userID)
	if err != nil {
		return nil, err
	}

	totalTarget := 0.0
	totalSaved := 0.0
	activeGoals := 0
	completedGoals := 0

	for _, goal := range goals {
		totalTarget += goal.TargetAmount
		totalSaved += goal.CurrentAmount
		if goal.Status == "active" {
			activeGoals++
		}
		if goal.Status == "completed" {
			completedGoals++
		}
	}

	return map[string]interface{}{
		"total_goals":     len(goals),
		"active_goals":    activeGoals,
		"completed_goals": completedGoals,
		"total_target":    totalTarget,
		"total_saved":     totalSaved,
		"overall_progress": func() float64 {
			if totalTarget <= 0 {
				return 0
			}
			return (totalSaved / totalTarget) * 100
		}(),
	}, nil
}
