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
	client         *mongo.Client
	collection     *mongo.Collection
	paymentCol     *mongo.Collection
	accountCol     *mongo.Collection
	transactionCol *mongo.Collection
}

func NewDebtService(client *mongo.Client, dbName string) *DebtService {
	db := client.Database(dbName)
	return &DebtService{
		client:         client,
		collection:     db.Collection("debts"),
		paymentCol:     db.Collection("debt_payments"),
		accountCol:     db.Collection("accounts"),
		transactionCol: db.Collection("Transaction"),
	}
}

func (s *DebtService) CreateDebt(debt models.Debt) (models.Debt, error) {
	debt.Name = utils.SanitizeMongoValue(debt.Name)
	debt.Creditor = utils.SanitizeMongoValue(debt.Creditor)

	if debt.Name == "" {
		return models.Debt{}, errors.New("name is required")
	}

	// Auto-calculate balance if missing but tenor/payment are present
	if debt.CurrentBalance <= 0 && debt.PaymentAmount > 0 && debt.TenorMonths > 0 {
		debt.CurrentBalance = debt.PaymentAmount * float64(debt.TenorMonths)
	}

	if debt.OriginalAmount <= 0 {
		debt.OriginalAmount = debt.CurrentBalance
	}

	if debt.CurrentBalance <= 0 {
		return models.Debt{}, errors.New("balance must be greater than 0 (or provide payment_amount and tenor_months)")
	}

	if debt.Currency == "" {
		debt.Currency = "IDR"
	}

	debt.IsActive = true
	debt.IsPaidOff = false
	debt.TotalPaid = 0
	debt.TotalInterest = 0

	now := time.Now()

	// 1. Default StartDate to now if empty
	if debt.StartDate.IsZero() {
		debt.StartDate = now
	}

	// 2. Default NextPaymentDate based on frequency if empty
	if debt.NextPaymentDate.IsZero() {
		switch debt.PaymentFrequency {
		case models.RepayWeekly:
			debt.NextPaymentDate = debt.StartDate.AddDate(0, 0, 7)
		case models.RepayBiweekly:
			debt.NextPaymentDate = debt.StartDate.AddDate(0, 0, 14)
		case models.RepayMonthly:
			debt.NextPaymentDate = debt.StartDate.AddDate(0, 1, 0)
		case models.RepayAnnually:
			debt.NextPaymentDate = debt.StartDate.AddDate(1, 0, 0)
		default:
			debt.NextPaymentDate = debt.StartDate.AddDate(0, 1, 0)
		}
	}

	// 3. Auto-calculate EndDate if TenorMonths is provided
	if debt.TenorMonths > 0 {
		debt.EndDate = new(time.Time)
		*debt.EndDate = debt.StartDate.AddDate(0, debt.TenorMonths, 0)
		debt.RemainingPayments = debt.TenorMonths
	}

	debt.CreatedAt = now
	debt.UpdatedAt = now

	// Financial Health Check (Nyusuaikan Kemampuan User)
	income, _ := s.GetUserMonthlyIncome(debt.UserID)
	isHealthy, _ := debt.CheckDebtHealth(income)
	debt.IsRecommended = isHealthy

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

func (s *DebtService) MakePayment(id primitive.ObjectID, userID primitive.ObjectID, accountID primitive.ObjectID, amount float64) (models.DebtPayment, error) {
	session, err := s.client.StartSession()
	if err != nil {
		return models.DebtPayment{}, err
	}
	defer session.EndSession(context.TODO())

	var paymentResult models.DebtPayment

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// 1. Get Debt Details
		var debt models.Debt
		err := s.collection.FindOne(sessCtx, bson.M{"_id": id, "user_id": userID}).Decode(&debt)
		if err != nil {
			return nil, errors.New("debt not found")
		}

		if debt.IsPaidOff {
			return nil, errors.New("debt is already paid off")
		}

		now := time.Now()

		// Calculate next payment date based on frequency
		nextDate := debt.NextPaymentDate
		switch debt.PaymentFrequency {
		case models.RepayWeekly:
			nextDate = nextDate.AddDate(0, 0, 7)
		case models.RepayBiweekly:
			nextDate = nextDate.AddDate(0, 0, 14)
		case models.RepayMonthly:
			nextDate = nextDate.AddDate(0, 1, 0)
		case models.RepayAnnually:
			nextDate = nextDate.AddDate(1, 0, 0)
		default:
			nextDate = nextDate.AddDate(0, 1, 0)
		}

		// 2. Optional Account/Transaction Handling
		if !accountID.IsZero() {
			var account models.Account
			err = s.accountCol.FindOne(sessCtx, bson.M{"_id": accountID, "user_id": userID}).Decode(&account)
			if err != nil {
				return nil, errors.New("payment account not found")
			}

			if account.CurrentBalance < amount {
				return nil, fmt.Errorf("insufficient balance in account %s (available: %.2f)", account.Name, account.CurrentBalance)
			}

			// A. Deduct from Account
			_, err = s.accountCol.UpdateOne(sessCtx, bson.M{"_id": accountID}, bson.M{
				"$inc": bson.M{"current_balance": -amount},
				"$set": bson.M{"updated_at": now},
			})
			if err != nil {
				return nil, err
			}

			// B. Create Global Transaction Record
			globalTrans := models.Transaction{
				ID:          primitive.NewObjectID(),
				User_id:     userID.Hex(),
				Amount:      amount,
				Category:    "Debt Payment",
				Description: fmt.Sprintf("Payment for %s to %s", debt.Name, debt.Creditor),
				Date:        now,
				Month:       now.Format("January 2006"),
			}
			_, err = s.transactionCol.InsertOne(sessCtx, globalTrans)
			if err != nil {
				return nil, err
			}
		}

		// 3. Calculation
		monthlyRate := debt.InterestRate / 12 / 100
		interest := debt.CurrentBalance * monthlyRate
		principal := amount - interest

		if principal < 0 {
			return nil, fmt.Errorf("payment amount (%.2f) must be at least the monthly interest (%.2f)", amount, interest)
		}

		if principal > debt.CurrentBalance {
			principal = debt.CurrentBalance
			amount = principal + interest
		}

		newBalance := debt.CurrentBalance - principal
		totalInterest := debt.TotalInterest + interest
		totalPaid := debt.TotalPaid + amount
		isPaidOff := newBalance <= 0.01

		// 4. Update Debt Record
		_, err = s.collection.UpdateOne(sessCtx, bson.M{"_id": id}, bson.M{
			"$set": bson.M{
				"current_balance":   newBalance,
				"total_paid":        totalPaid,
				"total_interest":    totalInterest,
				"is_paid_off":       isPaidOff,
				"next_payment_date": nextDate,
				"updated_at":        now,
			},
		})
		if err != nil {
			return nil, err
		}

		// 5. Create Debt Payment Log
		paymentResult = models.DebtPayment{
			ID:        primitive.NewObjectID(),
			DebtID:    id,
			UserID:    userID,
			Amount:    amount,
			Principal: principal,
			Interest:  interest,
			Date:      now,
			CreatedAt: now,
		}
		_, err = s.paymentCol.InsertOne(sessCtx, paymentResult)

		return paymentResult, err
	}

	_, err = session.WithTransaction(context.TODO(), callback)
	return paymentResult, err
}

func (s *DebtService) GetPaymentHistory(debtID primitive.ObjectID, userID primitive.ObjectID) ([]models.DebtPayment, error) {
	cursor, err := s.paymentCol.Find(context.TODO(), bson.M{
		"debt_id": debtID,
		"user_id": userID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var payments []models.DebtPayment
	if err := cursor.All(context.TODO(), &payments); err != nil {
		return nil, err
	}

	return payments, nil
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

func (s *DebtService) DeleteDebt(id primitive.ObjectID, userID primitive.ObjectID) error {
	result, err := s.collection.DeleteOne(context.TODO(), bson.M{"_id": id, "user_id": userID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("debt not found")
	}
	return nil
}

// GetUserMonthlyIncome calculates average income from the last 3 months of transactions
func (s *DebtService) GetUserMonthlyIncome(userID primitive.ObjectID) (float64, error) {
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)

	// Categories that count as income
	incomeCategories := []string{"income", "gaji", "salary", "pendapatan", "revenue"}

	cursor, err := s.transactionCol.Find(context.TODO(), bson.M{
		"user_id":  userID.Hex(),
		"date":     bson.M{"$gte": threeMonthsAgo},
		"category": bson.M{"$in": incomeCategories},
	})
	if err != nil {
		return 0, err
	}
	defer cursor.Close(context.TODO())

	totalIncome := 0.0
	count := 0
	for cursor.Next(context.TODO()) {
		var t models.Transaction
		if err := cursor.Decode(&t); err == nil {
			totalIncome += t.Amount
			count++
		}
	}

	if count == 0 {
		return 0, nil
	}

	// Return average monthly income
	return totalIncome / 3, nil
}
