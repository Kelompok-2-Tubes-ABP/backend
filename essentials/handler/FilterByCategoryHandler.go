package handler

import (
	model "financeapi/essentials/models"
	service "financeapi/essentials/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func FilterByCategoryHandler(t *service.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authorized"})
			return
		}

		filter := model.FilterTransaction{
			Category: c.DefaultQuery("category", ""),
			Type:     c.DefaultQuery("type", ""),
			FromDate: c.DefaultQuery("from_date", ""),
			ToDate:   c.DefaultQuery("to_date", ""),
		}

		result, err := t.ShowTransactionByFilter(userID.(string), filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
			return
		}
		c.JSON(200, gin.H{"Filtered Transactions: ": result})
	}
}
