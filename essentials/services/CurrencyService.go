package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"financeapi/essentials/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
)

type CurrencyService struct {
	currencyCol     *mongo.Collection
	rateCol         *mongo.Collection
	userCurrencyCol *mongo.Collection
	apiKey          string
	apiURL          string
}

func NewCurrencyService(client *mongo.Client, dbName string) *CurrencyService {
	db := client.Database(dbName)
	return &CurrencyService{
		currencyCol:     db.Collection("currencies"),
		rateCol:         db.Collection("exchange_rates"),
		userCurrencyCol: db.Collection("user_currencies"),
		apiKey:          os.Getenv("EXCHANGE_API_KEY"),
		apiURL:          "https://v6.exchangerate-api.com/v6",
	}
}

func (s *CurrencyService) InitializeSupportedCurrencies() error {
	currencies := models.SupportedCurrencies()

	for _, currency := range currencies {
		var existing models.Currency
		err := s.currencyCol.FindOne(context.TODO(), bson.M{"code": currency.Code}).Decode(&existing)
		if err == nil {
			continue
		}

		currency.CreatedAt = time.Now()
		currency.UpdatedAt = time.Now()
		_, err = s.currencyCol.InsertOne(context.TODO(), currency)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *CurrencyService) GetSupportedCurrencies() ([]models.Currency, error) {
	cursor, err := s.currencyCol.Find(context.TODO(), bson.M{"is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var currencies []models.Currency
	if err := cursor.All(context.TODO(), &currencies); err != nil {
		return nil, err
	}

	return currencies, nil
}

func (s *CurrencyService) GetCurrency(code string) (models.Currency, error) {
	var currency models.Currency
	err := s.currencyCol.FindOne(context.TODO(), bson.M{"code": code, "is_active": true}).Decode(&currency)
	if err != nil {
		return models.Currency{}, errors.New("currency not found")
	}
	return currency, nil
}

func (s *CurrencyService) GetExchangeRate(fromCurrency, toCurrency string) (models.ExchangeRate, error) {
	var rate models.ExchangeRate
	// 1. Cek di Database (Cache)
	err := s.rateCol.FindOne(context.TODO(), bson.M{
		"from_currency":   fromCurrency,
		"to_currency":     toCurrency,
		"expiration_time": bson.M{"$gt": time.Now()},
	}).Decode(&rate)

	if err == nil {
		return rate, nil
	}

	// 2. Jika tidak ada/expired, coba ambil dari API
	if s.apiKey != "" {
		newRate, err := s.FetchExchangeRateFromAPI(fromCurrency, toCurrency)
		if err == nil {
			// Simpan ke Database untuk cache 24 jam ke depan
			newRate.ExpirationTime = time.Now().Add(24 * time.Hour)
			s.SetExchangeRate(newRate)
			return newRate, nil
		}
	}

	// 3. Fallback ke Default Rate jika API gagal
	return models.ExchangeRate{
		FromCurrency:   fromCurrency,
		ToCurrency:     toCurrency,
		Rate:           s.GetDefaultRate(fromCurrency, toCurrency),
		Source:         "default",
		LastUpdated:    time.Now(),
		ExpirationTime: time.Now().Add(1 * time.Hour),
	}, nil
}

func (s *CurrencyService) GetDefaultRate(fromCurrency, toCurrency string) float64 {
	rates := map[string]float64{
		"IDR_USD": 15650.0, // 1 USD = 15650 IDR
		"USD_IDR": 0.000064,
		"IDR_EUR": 17100.0,
		"EUR_IDR": 0.000058,
		"USD_EUR": 0.92,
		"EUR_USD": 1.09,
	}

	key := fromCurrency + "_" + toCurrency
	if rate, ok := rates[key]; ok {
		return rate
	}

	return 1.0 // Default to 1:1 if no rate found
}

func (s *CurrencyService) SetExchangeRate(rate models.ExchangeRate) error {
	rate.LastUpdated = time.Now()
	rate.ExpirationTime = time.Now().Add(24 * time.Hour)

	opts := mongoOptions.Update().SetUpsert(true)
	_, err := s.rateCol.UpdateOne(
		context.TODO(),
		bson.M{
			"from_currency": rate.FromCurrency,
			"to_currency":   rate.ToCurrency,
		},
		bson.M{"$set": rate},
		opts,
	)

	return err
}

func (s *CurrencyService) ConvertCurrency(userID primitive.ObjectID, fromCurrency, toCurrency string, amount float64) (models.CurrencyConversion, error) {
	rate, err := s.GetExchangeRate(fromCurrency, toCurrency)
	if err != nil {
		return models.CurrencyConversion{}, err
	}

	convertedAmount := amount * rate.Rate

	conversion := models.CurrencyConversion{
		UserID:          userID,
		FromCurrency:    fromCurrency,
		ToCurrency:      toCurrency,
		OriginalAmount:  amount,
		ConvertedAmount: convertedAmount,
		Rate:            rate.Rate,
		CreatedAt:       time.Now(),
	}

	return conversion, nil
}

func (s *CurrencyService) ConvertToDefault(userID primitive.ObjectID, amount float64, fromCurrency string) (models.CurrencyConversion, error) {
	userCurrency, err := s.GetUserCurrency(userID)
	if err != nil {
		userCurrency.DefaultCode = "IDR"
	}

	if fromCurrency == userCurrency.DefaultCode {
		return models.CurrencyConversion{
			UserID:          userID,
			FromCurrency:    fromCurrency,
			ToCurrency:      userCurrency.DefaultCode,
			OriginalAmount:  amount,
			ConvertedAmount: amount,
			Rate:            1.0,
			CreatedAt:       time.Now(),
		}, nil
	}

	return s.ConvertCurrency(userID, fromCurrency, userCurrency.DefaultCode, amount)
}

func (s *CurrencyService) GetUserCurrency(userID primitive.ObjectID) (models.UserCurrency, error) {
	var userCurrency models.UserCurrency
	err := s.userCurrencyCol.FindOne(context.TODO(), bson.M{"user_id": userID}).Decode(&userCurrency)
	if err != nil {
		return models.UserCurrency{
			UserID:      userID,
			DefaultCode: "IDR",
			Currencies:  []string{"IDR"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	return userCurrency, nil
}

func (s *CurrencyService) SetUserDefaultCurrency(userID primitive.ObjectID, currencyCode string) error {
	_, err := s.GetCurrency(currencyCode)
	if err != nil {
		return errors.New("currency not found")
	}

	opts := mongoOptions.Update().SetUpsert(true)
	_, err = s.userCurrencyCol.UpdateOne(
		context.TODO(),
		bson.M{"user_id": userID},
		bson.M{
			"$set": bson.M{
				"default_code": currencyCode,
				"updated_at":   time.Now(),
			},
			"$setOnInsert": bson.M{
				"user_id":    userID,
				"currencies": []string{currencyCode},
				"created_at": time.Now(),
			},
		},
		opts,
	)

	return err
}

func (s *CurrencyService) AddUserCurrency(userID primitive.ObjectID, currencyCode string) error {
	_, err := s.GetCurrency(currencyCode)
	if err != nil {
		return errors.New("currency not found")
	}

	opts := mongoOptions.Update().SetUpsert(true)
	_, err = s.userCurrencyCol.UpdateOne(
		context.TODO(),
		bson.M{"user_id": userID},
		bson.M{
			"$addToSet": bson.M{"currencies": currencyCode},
			"$set":      bson.M{"updated_at": time.Now()},
			"$setOnInsert": bson.M{
				"user_id":      userID,
				"default_code": "IDR",
				"created_at":   time.Now(),
			},
		},
		opts,
	)

	return err
}

func (s *CurrencyService) GetAllRatesForCurrency(currencyCode string) ([]models.ExchangeRate, error) {
	cursor, err := s.rateCol.Find(context.TODO(), bson.M{
		"$or": []bson.M{
			{"from_currency": currencyCode},
			{"to_currency": currencyCode},
		},
		"expiration_time": bson.M{"$gt": time.Now()},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var rates []models.ExchangeRate
	if err := cursor.All(context.TODO(), &rates); err != nil {
		return nil, err
	}

	return rates, nil
}

func (s *CurrencyService) FetchExchangeRateFromAPI(from, to string) (models.ExchangeRate, error) {
	url := fmt.Sprintf("%s/%s/pair/%s/%s", s.apiURL, s.apiKey, from, to)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return models.ExchangeRate{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.ExchangeRate{}, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var result struct {
		Result         string  `json:"result"`
		ConversionRate float64 `json:"conversion_rate"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.ExchangeRate{}, err
	}

	if result.Result != "success" {
		return models.ExchangeRate{}, fmt.Errorf("API error: %s", result.Result)
	}

	return models.ExchangeRate{
		FromCurrency: from,
		ToCurrency:   to,
		Rate:         result.ConversionRate,
		Source:       "exchangerate-api",
		LastUpdated:  time.Now(),
	}, nil
}
