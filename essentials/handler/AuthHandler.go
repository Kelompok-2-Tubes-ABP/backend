package handler

import (
	"net/http"

	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	userService  *services.UserService
	emailService *services.EmailService
}

func NewAuthHandler(userService *services.UserService, emailService *services.EmailService) *AuthHandler {
	return &AuthHandler{userService: userService, emailService: emailService}
}

func (h *AuthHandler) SendVerificationEmail() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var userIDStr string
		switch v := userID.(type) {
		case string:
			userIDStr = v
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
			return
		}

		user, err := h.userService.GetUserByStringID(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
			return
		}

		code, email, err := h.userService.SendVerificationEmailByString(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		username := user.Username

		if h.emailService != nil {
			err = h.emailService.SendVerificationEmail(email, code, username)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"message":   "verification code generated",
					"email":     email,
					"email_err": err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "verification email sent",
			"email":   email,
		})
	}
}

func (h *AuthHandler) VerifyEmail() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "verification token required"})
			return
		}

		err := h.userService.VerifyEmail(token)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "email verified successfully",
		})
	}
}

func (h *AuthHandler) VerifyCode() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Code string `json:"code" binding:"required,len=6"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "6-digit verification code required"})
			return
		}

		err := h.userService.VerifyVerificationCode(req.Code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"valid": false,
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"valid":   true,
			"message": "email verified successfully",
		})
	}
}

func (h *AuthHandler) RequestPasswordReset() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "valid email required"})
			return
		}

		ipAddress := c.ClientIP()

		token, err := h.userService.RequestPasswordReset(req.Email, ipAddress)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message": "if the email exists, a reset code has been sent",
			})
			return
		}

		user, err := h.userService.GetUserByEmail(req.Email)
		if err == nil && user.Username != "" {
			go func() {
				_ = h.emailService.SendPasswordResetEmail(req.Email, token, user.Username)
			}()
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "password reset code sent to your email",
		})
	}
}

func (h *AuthHandler) VerifyResetToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
			Token string `json:"token" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email and token required"})
			return
		}

		err := h.userService.VerifyResetToken(req.Email, req.Token)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "token verified",
		})
	}
}

func (h *AuthHandler) ResetPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			Token       string `json:"token" binding:"required"`
			NewPassword string `json:"new_password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email, token, and new_password required"})
			return
		}

		err := h.userService.ResetPassword(req.Email, req.Token, req.NewPassword)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "password reset successfully",
		})
	}
}

func (h *AuthHandler) ChangePassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			CurrentPassword string `json:"current_password" binding:"required"`
			NewPassword     string `json:"new_password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "current_password and new_password required"})
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var userIDStr string
		switch v := userID.(type) {
		case string:
			userIDStr = v
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
			return
		}

		err := h.userService.ChangePasswordByString(userIDStr, req.CurrentPassword, req.NewPassword)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "password changed successfully",
		})
	}
}

func (h *AuthHandler) ResendVerification() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}

		user, err := h.userService.GetUserByEmail(req.Email)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message": "if the email exists, a verification code has been sent",
			})
			return
		}

		if user.IsEmailVerified {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email already verified"})
			return
		}

		code, err := h.userService.ResendVerificationEmail(req.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if h.emailService != nil {
			err = h.emailService.SendVerificationEmail(req.Email, code, user.Username)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"message":   "verification code generated",
					"email":     req.Email,
					"email_err": err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "verification email sent",
			"email":   req.Email,
		})
	}
}

// DebugVerifyUserHandler - ONLY for development/testing
// Verifies a user's email directly without code
func DebugVerifyUserHandler(userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "email required"})
			return
		}

		user, err := userService.GetUserByEmail(req.Email)
		if err != nil {
			c.JSON(404, gin.H{"error": "user not found"})
			return
		}

		err = userService.VerifyUserEmail(user.ID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": "email verified successfully",
			"email":   req.Email,
		})
	}
}

// DebugResetPasswordHandler - ONLY for development/testing
// Resets password directly without email
func DebugResetPasswordHandler(userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			NewPassword string `json:"new_password" binding:"required,min=6"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "email and new_password required"})
			return
		}

		user, err := userService.GetUserByEmail(req.Email)
		if err != nil {
			c.JSON(404, gin.H{"error": "user not found"})
			return
		}

		err = userService.ResetPasswordDebug(user.ID, req.NewPassword)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": "password reset successfully",
			"email":   req.Email,
		})
	}
}
