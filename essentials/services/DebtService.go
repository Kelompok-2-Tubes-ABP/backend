package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DebtService struct {
	collection *mongo.Collection
}

func NewDebtService(client *mongo.Client, dbName string) *DebtService {
	return &DebtService{
		collection: client.Database(dbName).Collection("debts"),
	}
}

func (s *DebtService) CreateDebt(debt models.Debt) (models.Debt, error) {
	debt.Name = utils.SanitizeMongoValue(debt.Name)
	debt.Creditor = utils.SanitizeMongoValue(debt.Creditor)

	if debt.Name == "" {
		return models.Debt{}, errors.New("name is required")
	}
	if debt.CurrentBalance <= 0 {
		return models.Debt{}, errors.New("balance must be greater than 0")
	}

	if debt.Currency == "" {
		debt.Currency = "IDR"
	}

	debt.IsActive = true
	debt.IsPaidOff = false
	debt.TotalPaid = 0
	debt.CreatedAt = time.Now()
	debt.UpdatedAt = time.Now()

	result, err := s.collection.InsertOne(context.TODO(), debt)
	if err != nil {
		return models.Debt{}, err
	}

	debt.ID = result.InsertedID.(primitive.ObjectID)
	return debt, nil
}

func (s *DebtService) GetDebt(id primitive.ObjectID, userID primitive.ObjectID) (models.Debt, error) {
	var debt models.Debt
	err := s.collection.FindOne(context.TODO(), bson.M{"_id": id, "user_id": userID}).Decode(&debt)
	if err != nil {
		return models.Debt{}, errors.New("debt not found")
	}
	return debt, nil
}

func (s *DebtService) GetUserDebts(userID primitive.ObjectID) ([]models.Debt, error) {
	cursor, err := s.collection.Find(context.TODO(), bson.M{"user_id": userID, "is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var debts []models.Debt
	if err := cursor.All(context.TODO(), &debts); err != nil {
		return nil, err
	}

	return debts, nil
}

func (s *DebtService) UpdateBalance(id primitive.ObjectID, userID primitive.ObjectID, newBalance float64) error {
	_, err := s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": bson.M{
			"current_balance": newBalance,
			"updated_at":      time.Now(),
		}},
	)
	return err
}

func (s *DebtService) MakePayment(id primitive.ObjectID, userID primitive.ObjectID, amount float64) error {
	debt, err := s.GetDebt(id, userID)
	if err != nil {
		return err
	}

	monthlyRate := debt.InterestRate / 12 / 100
	interest := debt.CurrentBalance * monthlyRate

	if amount < interest {
		return fmt.Errorf("payment amount (%.2f) must be at least the monthly interest (%.2f)", amount, interest)
	}

	principal := amount - interest

	if principal > debt.CurrentBalance {
		principal = debt.CurrentBalance
	}

	newBalance := debt.CurrentBalance - principal
	totalPaid := debt.TotalPaid + amount

	isPaidOff := newBalance <= 0
	if isPaidOff {
		newBalance = 0
	}

	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": bson.M{
			"current_balance": newBalance,
			"total_paid":      totalPaid,
			"is_paid_off":     isPaidOff,
			"updated_at":      time.Now(),
		}},
	)

	return err
}

func (s *DebtService) GetDebtSummary(userID primitive.ObjectID) (map[string]interface{}, error) {
	debts, err := s.GetUserDebts(userID)
	if err != nil {
		return nil, err
	}

	totalDebt := 0.0
	totalMonthlyPayment := 0.0
	byType := make(map[string]float64)

	for _, debt := range debts {
		totalDebt += debt.CurrentBalance
		totalMonthlyPayment += debt.PaymentAmount
		byType[string(debt.Type)] += debt.CurrentBalance
	}

	return map[string]interface{}{
		"total_debts":           len(debts),
		"total_debt":            totalDebt,
		"total_monthly_payment": totalMonthlyPayment,
		"by_type":               byType,
	}, nil
}
