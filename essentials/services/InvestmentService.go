package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type InvestmentService struct {
	collection     *mongo.Collection
	transactionCol *mongo.Collection
	priceService   *PriceService
}

func NewInvestmentService(client *mongo.Client, dbName string) *InvestmentService {
	db := client.Database(dbName)
	return &InvestmentService{
		collection:     db.Collection("investments"),
		transactionCol: db.Collection("investment_transactions"),
	}
}

func (s *InvestmentService) SetPriceService(ps *PriceService) {
	s.priceService = ps
}

func (s *InvestmentService) CreateInvestment(inv models.Investment) (models.Investment, error) {
	inv.Name = utils.SanitizeMongoValue(inv.Name)
	inv.Symbol = utils.SanitizeMongoValue(inv.Symbol)

	if inv.Name == "" || inv.Symbol == "" {
		return models.Investment{}, errors.New("name and symbol are required")
	}
	if inv.Quantity <= 0 {
		return models.Investment{}, errors.New("quantity must be greater than 0")
	}

	if inv.Currency == "" {
		inv.Currency = "IDR"
	}

	// Auto-fetch current price if not provided
	if inv.CurrentPrice <= 0 && s.priceService != nil {
		var price float64
		var err error
		if inv.Type == "crypto" {
			price, err = s.priceService.GetCryptoPrice(inv.Symbol, inv.Currency)
		} else {
			price, err = s.priceService.GetStockPrice(inv.Symbol, true)
		}
		if err == nil && price > 0 {
			inv.CurrentPrice = price
		}
	}

	// Set default average_cost if not provided (use current price)
	if inv.AverageCost <= 0 {
		inv.AverageCost = inv.CurrentPrice
	}

	inv.CalculateValues()
	inv.IsActive = true
	inv.LastPriceUpdate = time.Now()
	inv.CreatedAt = time.Now()
	inv.UpdatedAt = time.Now()

	result, err := s.collection.InsertOne(context.TODO(), inv)
	if err != nil {
		return models.Investment{}, err
	}

	inv.ID = result.InsertedID.(primitive.ObjectID)
	return inv, nil
}

func (s *InvestmentService) GetInvestment(id primitive.ObjectID, userID primitive.ObjectID) (models.Investment, error) {
	var inv models.Investment
	err := s.collection.FindOne(context.TODO(), bson.M{"_id": id, "user_id": userID}).Decode(&inv)
	if err != nil {
		return models.Investment{}, errors.New("investment not found")
	}
	return inv, nil
}

func (s *InvestmentService) GetUserInvestments(userID primitive.ObjectID) ([]models.Investment, error) {
	cursor, err := s.collection.Find(context.TODO(), bson.M{"user_id": userID, "is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var investments []models.Investment
	if err := cursor.All(context.TODO(), &investments); err != nil {
		return nil, err
	}

	return investments, nil
}

func (s *InvestmentService) GetInvestmentsByType(userID primitive.ObjectID, invType models.InvestmentType) ([]models.Investment, error) {
	cursor, err := s.collection.Find(context.TODO(), bson.M{"user_id": userID, "type": invType, "is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var investments []models.Investment
	if err := cursor.All(context.TODO(), &investments); err != nil {
		return nil, err
	}

	return investments, nil
}

func (s *InvestmentService) UpdatePrice(id primitive.ObjectID, userID primitive.ObjectID, newPrice float64) error {
	if newPrice <= 0 {
		return errors.New("price must be greater than 0")
	}

	inv, err := s.GetInvestment(id, userID)
	if err != nil {
		return err
	}

	inv.CurrentPrice = newPrice
	inv.CalculateValues()
	inv.LastPriceUpdate = time.Now()
	inv.UpdatedAt = time.Now()

	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": bson.M{
			"current_price":     newPrice,
			"total_value":       inv.TotalValue,
			"gain_loss":         inv.GainLoss,
			"gain_loss_percent": inv.GainLossPercent,
			"last_price_update": inv.LastPriceUpdate,
			"updated_at":        inv.UpdatedAt,
		}},
	)
	return err
}

func (s *InvestmentService) DeleteInvestment(id primitive.ObjectID, userID primitive.ObjectID) error {
	_, err := s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": bson.M{"is_active": false, "updated_at": time.Now()}},
	)
	return err
}

func (s *InvestmentService) GetPortfolioSummary(userID primitive.ObjectID) (map[string]interface{}, error) {
	investments, err := s.GetUserInvestments(userID)
	if err != nil {
		return nil, err
	}

	totalValue := 0.0
	totalCost := 0.0
	byType := make(map[string]float64)

	for _, inv := range investments {
		totalValue += inv.TotalValue
		totalCost += inv.TotalCost
		byType[string(inv.Type)] += inv.TotalValue
	}

	gainLoss := totalValue - totalCost
	var gainLossPercent float64
	if totalCost > 0 {
		gainLossPercent = (gainLoss / totalCost) * 100
	}

	return map[string]interface{}{
		"total_investments": len(investments),
		"total_value":       totalValue,
		"total_cost":        totalCost,
		"gain_loss":         gainLoss,
		"gain_loss_percent": gainLossPercent,
		"by_type":           byType,
	}, nil
}

func (s *InvestmentService) RefreshAllPrices(userID primitive.ObjectID, currency string) (int, error) {
	if s.priceService == nil {
		return 0, errors.New("PriceService not initialized")
	}

	investments, err := s.GetUserInvestments(userID)
	if err != nil {
		return 0, err
	}

	if len(investments) == 0 {
		return 0, nil
	}

	var cryptoSymbols []string
	var stockSymbols []string

	for _, inv := range investments {
		switch string(inv.Type) {
		case "crypto":
			cryptoSymbols = append(cryptoSymbols, strings.ToLower(inv.Symbol))
		case "stock":
			stockSymbols = append(stockSymbols, inv.Symbol)
		}
	}

	cryptoPrices := make(map[string]float64)
	stockPrices := make(map[string]float64)

	if len(cryptoSymbols) > 0 {
		cryptoPrices, err = s.priceService.GetCryptoPrices(cryptoSymbols, currency)
		if err != nil {
		}
	}

	if len(stockSymbols) > 0 {
		stockPrices, err = s.priceService.GetStockPrices(stockSymbols, strings.ToUpper(currency) == "IDR")
		if err != nil {
		}
	}

	updatedCount := 0
	for _, inv := range investments {
		var newPrice float64
		var found bool

		switch string(inv.Type) {
		case "crypto":
			symbolLower := strings.ToLower(inv.Symbol)
			newPrice, found = cryptoPrices[symbolLower]
		case "stock":
			newPrice, found = stockPrices[strings.ToUpper(inv.Symbol)]
		}

		if found && newPrice > 0 {
			err = s.UpdatePrice(inv.ID, userID, newPrice)
			if err == nil {
				updatedCount++
			}
		}
	}

	return updatedCount, nil
}

func (s *InvestmentService) RefreshSinglePrice(id primitive.ObjectID, userID primitive.ObjectID, currency string) (float64, error) {
	if s.priceService == nil {
		return 0, errors.New("PriceService not initialized")
	}

	inv, err := s.GetInvestment(id, userID)
	if err != nil {
		return 0, err
	}

	newPrice, err := s.priceService.GetPrice(inv.Symbol, string(inv.Type), currency)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch price: %w", err)
	}

	err = s.UpdatePrice(id, userID, newPrice)
	if err != nil {
		return 0, err
	}

	return newPrice, nil
}

func (s *InvestmentService) GetPortfolioWithLivePrices(userID primitive.ObjectID, currency string) ([]models.Investment, error) {
	if s.priceService == nil {
		return nil, errors.New("PriceService not initialized")
	}

	_, err := s.RefreshAllPrices(userID, currency)
	if err != nil {
	}

	return s.GetUserInvestments(userID)
}

func (s *InvestmentService) GetPortfolioSummaryWithLivePrices(userID primitive.ObjectID, currency string) (map[string]interface{}, error) {
	_, err := s.RefreshAllPrices(userID, currency)
	if err != nil {
	}

	return s.GetPortfolioSummary(userID)
}
func (s *InvestmentService) AddTransaction(tx models.InvestmentTransaction) (models.InvestmentTransaction, error) {
	// 1. Get the investment
	inv, err := s.GetInvestment(tx.InvestmentID, tx.UserID)
	if err != nil {
		return models.InvestmentTransaction{}, err
	}

	tx.CreatedAt = time.Now()
	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}

	// 2. Adjust Quantity and Average Cost based on type
	switch strings.ToLower(tx.Type) {
	case "buy":
		totalCostOld := inv.Quantity * inv.AverageCost
		totalCostNew := tx.Quantity * tx.Price
		inv.Quantity += tx.Quantity
		inv.AverageCost = (totalCostOld + totalCostNew) / inv.Quantity
	case "sell":
		if tx.Quantity > inv.Quantity {
			return models.InvestmentTransaction{}, errors.New("insufficient quantity to sell")
		}
		inv.Quantity -= tx.Quantity
		// Average Cost doesn't change when selling
	case "dividend":
		// Dividend usually doesn't change quantity/avg cost of units,
		// but it contributes to overall return. Realistically, some people
		// like to subtract it from average cost, but typically it's just a cash inflow.
	default:
		return models.InvestmentTransaction{}, fmt.Errorf("invalid transaction type: %s", tx.Type)
	}

	// 3. Update Investment values
	inv.CalculateValues()
	inv.UpdatedAt = time.Now()

	// 4. Save Transaction
	result, err := s.transactionCol.InsertOne(context.TODO(), tx)
	if err != nil {
		return models.InvestmentTransaction{}, err
	}
	tx.ID = result.InsertedID.(primitive.ObjectID)

	// 5. Update Investment in DB
	_, err = s.collection.ReplaceOne(context.TODO(), bson.M{"_id": inv.ID}, inv)
	if err != nil {
		return tx, fmt.Errorf("transaction saved but failed to update investment: %w", err)
	}

	return tx, nil
}

func (s *InvestmentService) GetInvestmentTransactions(investmentID primitive.ObjectID, userID primitive.ObjectID) ([]models.InvestmentTransaction, error) {
	cursor, err := s.transactionCol.Find(context.TODO(), bson.M{
		"investment_id": investmentID,
		"user_id":       userID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var transactions []models.InvestmentTransaction
	if err := cursor.All(context.TODO(), &transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}
