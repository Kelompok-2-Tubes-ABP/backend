package handler

import (
	model "financeapi/essentials/models"
	service "financeapi/essentials/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func FilterByCategoryHandler(t *service.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var f model.FilterTransaction
		if err := c.ShouldBindJSON(&f); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authorized"})
			return
		}
		result, err := t.ShowTransactionByFilter(userID.(string), f)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
			return
		}
		c.JSON(200, gin.H{"Filtered Transactions: ": result})
	}
}
