package handler

import (
	"financeapi/essentials/models"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func GetMonthlyHandler(t *services.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"Error": "Not Authorized!!!"})
			return
		}

		month := c.DefaultQuery("month", "")

		report := models.Report{Month: month}
		monthly, errr := t.GetMonthly(userID.(string), report)

		if errr != nil {
			c.JSON(401, gin.H{"Error": "Transactions Not Found!!!"})
			return
		}
		c.JSON(200, gin.H{"Message": monthly})
	}
}
