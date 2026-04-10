package handler

import (
	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"financeapi/essentials/utils"

	"github.com/gin-gonic/gin"
)

func RegisterHandler(userService *services.UserService, emailService *services.EmailService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var u models.User
		if err := c.ShouldBindJSON(&u); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		u.Username = utils.SanitizeInput(u.Username)
		u.Email = utils.SanitizeInput(u.Email)

		valid, msg := utils.ValidateUsername(u.Username)
		if !valid {
			c.JSON(400, gin.H{"error": msg})
			return
		}

		if !utils.ValidateEmail(u.Email) {
			c.JSON(400, gin.H{"error": "Invalid email format"})
			return
		}

		valid, msg = utils.ValidatePasswordStrength(u.Password)
		if !valid {
			c.JSON(400, gin.H{"error": msg})
			return
		}

		u.IsActive = false

		createdUser, err := userService.CreateUser(u)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		verificationToken, email, err := userService.SendVerificationEmail(createdUser.ID)
		if err != nil {
			c.JSON(201, gin.H{
				"message":    "User registered, but we couldn't send the email.",
				"user_id":    createdUser.ID.Hex(),
				"username":   createdUser.Username,
				"email":      createdUser.Email,
				"email_sent": false,
				"warning":    "Please use the 'Resend Verification' feature to get your code."})
			return
		}

		if emailService != nil {
			err = emailService.SendVerificationEmail(email, verificationToken, createdUser.Username)
			if err != nil {
				c.JSON(201, gin.H{
					"message":    "User registered, but we couldn't send the email.",
					"user_id":    createdUser.ID.Hex(),
					"username":   createdUser.Username,
					"email":      createdUser.Email,
					"email_sent": false,
					"warning":    "Please use the 'Resend Verification' feature to get your code."})
				return
			}
		}

		c.JSON(201, gin.H{
			"message":    "User registered successfully. Please check your email for verification code.",
			"user_id":    createdUser.ID.Hex(),
			"username":   createdUser.Username,
			"email":      createdUser.Email,
			"email_sent": true,
		})
	}
}
