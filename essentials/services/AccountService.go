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

type AccountService struct {
	collection      *mongo.Collection
	groupCollection *mongo.Collection
}

func NewAccountService(client *mongo.Client, dbName string) *AccountService {
	db := client.Database(dbName)
	return &AccountService{
		collection:      db.Collection("accounts"),
		groupCollection: db.Collection("account_groups"),
	}
}

func (s *AccountService) CreateAccount(account models.Account) (models.Account, error) {
	account.Name = utils.SanitizeMongoValue(account.Name)
	account.Institution = utils.SanitizeMongoValue(account.Institution)
	account.AccountNumber = utils.SanitizeMongoValue(account.AccountNumber)
	account.Notes = utils.SanitizeMongoValue(account.Notes)

	if account.Name == "" {
		return models.Account{}, errors.New("account name is required")
	}

	if account.Currency == "" {
		account.Currency = "IDR"
	}

	account.CurrentBalance = account.InitialBalance
	account.IsActive = true
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()

	result, err := s.collection.InsertOne(context.TODO(), account)
	if err != nil {
		return models.Account{}, err
	}

	account.ID = result.InsertedID.(primitive.ObjectID)
	return account, nil
}

func (s *AccountService) GetAccount(accountID primitive.ObjectID, userID primitive.ObjectID) (models.Account, error) {
	var account models.Account
	err := s.collection.FindOne(context.TODO(), bson.M{
		"_id":     accountID,
		"user_id": userID,
	}).Decode(&account)
	if err != nil {
		return models.Account{}, errors.New("account not found")
	}
	return account, nil
}

func (s *AccountService) GetUserAccounts(userID primitive.ObjectID) ([]models.Account, error) {
	cursor, err := s.collection.Find(context.TODO(), bson.M{
		"user_id":   userID,
		"is_active": true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var accounts []models.Account
	if err := cursor.All(context.TODO(), &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (s *AccountService) GetUserAccountsByType(userID primitive.ObjectID, accountType models.AccountType) ([]models.Account, error) {
	cursor, err := s.collection.Find(context.TODO(), bson.M{
		"user_id":   userID,
		"type":      accountType,
		"is_active": true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var accounts []models.Account
	if err := cursor.All(context.TODO(), &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (s *AccountService) UpdateAccount(accountID primitive.ObjectID, userID primitive.ObjectID, updates bson.M) (models.Account, error) {
	if name, ok := updates["name"].(string); ok {
		updates["name"] = utils.SanitizeMongoValue(name)
	}
	if institution, ok := updates["institution"].(string); ok {
		updates["institution"] = utils.SanitizeMongoValue(institution)
	}

	updates["updated_at"] = time.Now()

	opts := mongoOptions.FindOneAndUpdate().SetReturnDocument(mongoOptions.After)
	result := s.collection.FindOneAndUpdate(
		context.TODO(),
		bson.M{"_id": accountID, "user_id": userID},
		bson.M{"$set": updates},
		opts,
	)

	var account models.Account
	if err := result.Decode(&account); err != nil {
		return models.Account{}, errors.New("account not found")
	}

	return account, nil
}

func (s *AccountService) UpdateBalance(accountID primitive.ObjectID, userID primitive.ObjectID, newBalance float64) error {
	_, err := s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": accountID, "user_id": userID},
		bson.M{
			"$set": bson.M{
				"current_balance": newBalance,
				"updated_at":      time.Now(),
			},
		},
	)
	return err
}

func (s *AccountService) TransferBetweenAccounts(userID primitive.ObjectID, transfer models.AccountTransaction) error {
	_, err := s.GetAccount(transfer.FromAccountID, userID)
	if err != nil {
		return errors.New("source account not found")
	}

	_, err = s.GetAccount(transfer.ToAccountID, userID)
	if err != nil {
		return errors.New("destination account not found")
	}

	fromAccount, _ := s.GetAccount(transfer.FromAccountID, userID)
	if fromAccount.CurrentBalance < transfer.Amount {
		return errors.New("insufficient balance")
	}

	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": transfer.FromAccountID, "user_id": userID},
		bson.M{
			"$inc": bson.M{"current_balance": -transfer.Amount},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return err
	}

	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": transfer.ToAccountID, "user_id": userID},
		bson.M{
			"$inc": bson.M{"current_balance": transfer.Amount},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *AccountService) DeleteAccount(accountID primitive.ObjectID, userID primitive.ObjectID) error {
	result, err := s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": accountID, "user_id": userID},
		bson.M{
			"$set": bson.M{
				"is_active":  false,
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		return errors.New("account not found")
	}

	return nil
}

func (s *AccountService) GetAccountSummary(userID primitive.ObjectID) (map[string]interface{}, error) {
	accounts, err := s.GetUserAccounts(userID)
	if err != nil {
		return nil, err
	}

	totalAssets := 0.0
	totalDebt := 0.0
	bankAccounts := 0
	walletAccounts := 0
	cashAccounts := 0
	creditAccounts := 0

	for _, account := range accounts {
		if account.Type == models.AccountTypeCredit {
			totalDebt += account.CalculateDebt()
			creditAccounts++
		} else {
			totalAssets += account.CalculateTotalAssets()
		}

		switch account.Type {
		case models.AccountTypeBank:
			bankAccounts++
		case models.AccountTypeWallet:
			walletAccounts++
		case models.AccountTypeCash:
			cashAccounts++
		}
	}

	return map[string]interface{}{
		"total_accounts":  len(accounts),
		"total_assets":    totalAssets,
		"total_debt":      totalDebt,
		"net_worth":       totalAssets - totalDebt,
		"bank_accounts":   bankAccounts,
		"wallet_accounts": walletAccounts,
		"cash_accounts":   cashAccounts,
		"credit_accounts": creditAccounts,
	}, nil
}

func (s *AccountService) GetAccountGroups(userID primitive.ObjectID) ([]models.AccountGroup, error) {
	cursor, err := s.groupCollection.Find(context.TODO(), bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var groups []models.AccountGroup
	if err := cursor.All(context.TODO(), &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

func (s *AccountService) CreateAccountGroup(group models.AccountGroup) (models.AccountGroup, error) {
	group.Name = utils.SanitizeMongoValue(group.Name)
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	result, err := s.groupCollection.InsertOne(context.TODO(), group)
	if err != nil {
		return models.AccountGroup{}, err
	}

	group.ID = result.InsertedID.(primitive.ObjectID)
	return group, nil
}

func (s *AccountService) AddAccountToGroup(groupID primitive.ObjectID, userID primitive.ObjectID, accountID primitive.ObjectID) error {
	_, err := s.groupCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": groupID, "user_id": userID},
		bson.M{
			"$addToSet": bson.M{"account_ids": accountID},
			"$set":      bson.M{"updated_at": time.Now()},
		},
	)
	return err
}

func (s *AccountService) SyncAccount(accountID primitive.ObjectID, userID primitive.ObjectID) error {
	_, err := s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": accountID, "user_id": userID},
		bson.M{
			"$set": bson.M{
				"last_synced": time.Now(),
				"updated_at":  time.Now(),
			},
		},
	)
	return err
}
