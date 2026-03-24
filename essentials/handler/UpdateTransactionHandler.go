package handler

import (
	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func UpdateTransactionHandler(t *services.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		var u models.Transaction
		if err := c.ShouldBindJSON(&u); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if err := t.UpdateTransaction(objectID, u); err != nil {
			c.JSON(401, gin.H{"Error": err})
			return
		} else {
			c.JSON(200, gin.H{"message: ": "Update Successfully!!!"})
		}
	}
}
