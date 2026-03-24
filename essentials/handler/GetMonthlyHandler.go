package handler

import (
	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"fmt"

	"github.com/gin-gonic/gin"
)

func GetMonthlyHandler(t *services.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var report models.Report
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"Error": "Not Authorized!!!"})
			return
		}
		fmt.Printf("Looking for transactions with user_id = %s\n", userID)

		if err := c.ShouldBindJSON(&report); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		monthly, errr := t.GetMonthly(userID.(string), report)

		if errr != nil {
			c.JSON(401, gin.H{"Error": "Transactions Not Found!!!"})
			return
		}
		c.JSON(200, gin.H{"Message": monthly})
	}
}
