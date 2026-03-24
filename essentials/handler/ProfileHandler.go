package handler

import (
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ProfileHandler(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		objectID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID in token"})
			return
		}

		profile, exists := u.GetUser(objectID)

		if exists != nil {
			c.JSON(401, gin.H{"Error": "can't retrieve the data"})
			return
		}

		c.JSON(200, gin.H{"Message": profile})
	}
}
