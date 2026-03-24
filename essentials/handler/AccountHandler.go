package handler

import (
	"financeapi/essentials/models"
	"financeapi/essentials/services"

	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AccountHandler struct {
	service *services.AccountService
}

func NewAccountHandler(service *services.AccountService) *AccountHandler {
	return &AccountHandler{service: service}
}

func (h *AccountHandler) CreateAccount() gin.HandlerFunc {
	return func(c *gin.Context) {
		var account models.Account
		if err := c.ShouldBindJSON(&account); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		account.UserID = userID.(primitive.ObjectID)

		createdAccount, err := h.service.CreateAccount(account)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, createdAccount)
	}
}

func (h *AccountHandler) GetAccount() gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid account ID"})
			return
		}

		userID, _ := c.Get("user_id")
		account, err := h.service.GetAccount(accountID, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, account)
	}
}

func (h *AccountHandler) GetUserAccounts() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		accounts, err := h.service.GetUserAccounts(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, accounts)
	}
}

func (h *AccountHandler) GetAccountsByType() gin.HandlerFunc {
	return func(c *gin.Context) {
		accountType := models.AccountType(c.Param("type"))
		userID, _ := c.Get("user_id")
		accounts, err := h.service.GetUserAccountsByType(userID.(primitive.ObjectID), accountType)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, accounts)
	}
}

func (h *AccountHandler) UpdateAccount() gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid account ID"})
			return
		}

		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		account, err := h.service.UpdateAccount(accountID, userID.(primitive.ObjectID), updates)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, account)
	}
}

func (h *AccountHandler) UpdateBalance() gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid account ID"})
			return
		}

		var req struct {
			NewBalance float64 `json:"new_balance" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		err = h.service.UpdateBalance(accountID, userID.(primitive.ObjectID), req.NewBalance)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Balance updated successfully"})
	}
}

func (h *AccountHandler) Transfer() gin.HandlerFunc {
	return func(c *gin.Context) {
		var transfer models.AccountTransaction
		if err := c.ShouldBindJSON(&transfer); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		transfer.UserID = userID.(primitive.ObjectID)
		transfer.TransactionDate = time.Now()

		err := h.service.TransferBetweenAccounts(userID.(primitive.ObjectID), transfer)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Transfer successful"})
	}
}

func (h *AccountHandler) DeleteAccount() gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid account ID"})
			return
		}

		userID, _ := c.Get("user_id")
		err = h.service.DeleteAccount(accountID, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Account deleted successfully"})
	}
}

func (h *AccountHandler) GetAccountSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		summary, err := h.service.GetAccountSummary(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, summary)
	}
}

func (h *AccountHandler) GetAccountGroups() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		groups, err := h.service.GetAccountGroups(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, groups)
	}
}

func (h *AccountHandler) CreateAccountGroup() gin.HandlerFunc {
	return func(c *gin.Context) {
		var group models.AccountGroup
		if err := c.ShouldBindJSON(&group); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		group.UserID = userID.(primitive.ObjectID)

		createdGroup, err := h.service.CreateAccountGroup(group)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, createdGroup)
	}
}

func (h *AccountHandler) SyncAccount() gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid account ID"})
			return
		}

		userID, _ := c.Get("user_id")
		err = h.service.SyncAccount(accountID, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Account synced successfully"})
	}
}
