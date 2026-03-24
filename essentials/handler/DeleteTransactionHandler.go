package handler

import (
	service "financeapi/essentials/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func DeleteTransactionHandler(t *service.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		if err := t.DeleteTransaction(objID); err != nil {
			c.JSON(401, err)
			return
		} else {
			c.JSON(200, gin.H{"message": "Deleted Successfully!!!"})
		}
	}
}
