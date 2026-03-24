package handler

import (
	"context"
	auth "financeapi/essentials/auth"
	config "financeapi/essentials/config"
	"financeapi/essentials/models"
	services "financeapi/essentials/services"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func LoginHandler(userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var u models.User
		if err := c.ShouldBindJSON(&u); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		u.Email = sanitizeEmail(u.Email)

		user, err := userService.LoginUser(u.Email, u.Password)
		if err != nil {
			c.JSON(401, gin.H{"error": "Invalid email or password"})
			return
		}

		if !user.IsEmailVerified {
			c.JSON(401, gin.H{
				"error":          "Email not verified",
				"email_verified": false,
				"message":        "Please verify your email before logging in",
			})
			return
		}

		tokenString, jti, err := auth.CreateToken(user.ID.Hex(), user.Username)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate token"})
			return
		}

		collection := config.GetCollection(config.DB, "active_tokens")
		_, err = collection.InsertOne(context.TODO(), bson.M{
			"jti":       jti,
			"user_id":   user.ID.Hex(),
			"username":  user.Username,
			"expiresAt": time.Now().Add(2 * time.Hour),
		})
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to save token"})
			return
		}

		c.JSON(200, gin.H{
			"token":      tokenString,
			"expires_in": 7200,
			"token_type": "Bearer",
		})
	}
}

func sanitizeEmail(email string) string {
	email = trimAndLower(email)
	return email
}

func trimAndLower(s string) string {
	s = s
	for i := 0; i < len(s); i++ {
		if s[i] > ' ' {
			s = s[i:]
			break
		}
	}
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] > ' ' {
			s = s[:i+1]
			break
		}
	}
	result := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else {
			result += string(c)
		}
	}
	return result
}
