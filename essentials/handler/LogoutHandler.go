package handler

import (
	"context"
	config "financeapi/essentials/config"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func LogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		jtiVal, exists := c.Get("jti")
		if !exists {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		jti := jtiVal.(string)

		collection := config.GetCollection(config.DB, "active_tokens")
		_, err := collection.DeleteOne(context.TODO(), bson.M{"jti": jti})
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to logout"})
			return
		}

		c.JSON(200, gin.H{"message": "Logged out successfully"})
	}
}
