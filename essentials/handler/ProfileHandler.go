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

func UpdateProfileHandler(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		objectID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID in token"})
			return
		}

		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		// Only allow updating these fields
		allowedFields := map[string]bool{
			"username": true,
			"email":    true,
		}

		sanitizedUpdates := make(map[string]interface{})
		for key, value := range updates {
			if allowedFields[key] {
				sanitizedUpdates[key] = value
			}
		}

		// When email is changed, preserve is_email_verified = true so user stays verified
		if _, emailUpdated := sanitizedUpdates["email"]; emailUpdated {
			sanitizedUpdates["is_email_verified"] = true
		}

		// Handle password change separately
		if currentPwd, ok := updates["current_password"].(string); ok {
			if newPwd, ok := updates["new_password"].(string); ok {
				err := u.ChangePassword(objectID, currentPwd, newPwd)
				if err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}
			}
		}

		if len(sanitizedUpdates) == 0 {
			// Only password was updated, return success
			c.JSON(200, gin.H{"message": "Profile updated successfully"})
			return
		}

		updatedUser, err := u.UpdateUser(objectID, sanitizedUpdates)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": "Profile updated successfully",
			"user":    updatedUser,
		})
	}
}

// ChangePasswordHandler - Direct password change without email verification
// Requires JWT authentication, takes new password directly
func ChangePasswordHandler(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		objectID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID in token"})
			return
		}

		var req struct {
			CurrentPassword string `json:"current_password" binding:"required"`
			NewPassword     string `json:"new_password" binding:"required,min=6"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "current_password and new_password (min 6 chars) are required"})
			return
		}

		err = u.ChangePassword(objectID, req.CurrentPassword, req.NewPassword)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": "Password changed successfully",
		})
	}
}
