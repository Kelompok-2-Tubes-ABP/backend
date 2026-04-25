package auth

import (
	"financeapi/essentials/config"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		id, err := primitive.ObjectIDFromHex(userID.(string))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			c.Abort()
			return
		}

		adminColl := config.GetCollection(config.DB, "admins")
		var admin struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		err = adminColl.FindOne(c.Request.Context(), bson.M{"_id": id}).Decode(&admin)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Requires admin privileges"})
			c.Abort()
			return
		}

		c.Next()
	}
}
