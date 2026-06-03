package handler

import (
	model "financeapi/essentials/models"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func GetReportHandler(t *services.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"Error": "Not Authorized!!!!"})
			return
		}

		filter := model.FilterTransaction{
			FromDate: c.DefaultQuery("from_date", ""),
			ToDate:   c.DefaultQuery("to_date", ""),
			Category: c.DefaultQuery("category", ""),
			Type:     c.DefaultQuery("type", ""),
		}

		report, err := t.GetReport(userID.(string), filter)
		if err != nil {
			c.JSON(401, gin.H{"Error": "Transactions Not Found!!!"})
			return
		}
		c.JSON(200, gin.H{"Message": report})
	}
}
