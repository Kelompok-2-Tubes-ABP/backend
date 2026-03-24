package handler

import (
	model "financeapi/essentials/models"
	service "financeapi/essentials/services"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateTransactionHandler(t *service.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tr model.Transaction
		timee := time.Now()
		userID, exists := c.Get("user_id")
		monthName := timee.Format("January")
		if !exists {
			c.JSON(401, "Not Authorized!!!")
			return
		}
		if err := c.ShouldBindJSON(&tr); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		tr.User_id = userID.(string)
		tr.Date = timee
		tr.Month = monthName
		transaction, err := t.CreateTransaction(tr)

		if err != nil {
			c.JSON(401, gin.H{"Error": "Transaction Not Found!!!"})
			return
		}
		c.JSON(200, gin.H{"Transaction added succesfully": transaction})
	}
}
