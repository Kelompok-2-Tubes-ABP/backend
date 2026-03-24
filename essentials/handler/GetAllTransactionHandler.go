package handler

import (
	service "financeapi/essentials/services"
	"fmt"

	"github.com/gin-gonic/gin"
)

func GetAllTransaction(t *service.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"Error": "Not Authorized!!!!"})
			return
		}
		fmt.Printf("Looking for transactions with user_id = %s\n", userID)

		result, err := t.ShowTransaction(userID.(string))

		if err != nil {
			c.JSON(401, "Transaction Not Found!!!")
			return
		}
		c.JSON(200, gin.H{"Transaction Retrieved Successfully!!": result})
	}
}
