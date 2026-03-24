package handler

import (
	model "financeapi/essentials/models"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func GetReportHandler(t *services.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var f model.FilterTransaction
		if err := c.ShouldBindJSON(&f); err != nil {
			c.JSON(400, gin.H{"Error": err.Error()})
			return
		}
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"Error": "Not Authorized!!!!"})
			return
		}
		report, err := t.GetReport(userID.(string), f)

		if err != nil {
			c.JSON(401, gin.H{"Error": "Transactions Not Found!!!"})
			return
		}
		c.JSON(200, gin.H{"Message": report})
	}
}
