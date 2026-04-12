package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/middleware"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(r *gin.Engine, userService *services.UserService, emailService *services.EmailService) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", middleware.RateLimitMiddleware(), handler.LoginHandler(userService))
		auth.POST("/register", middleware.RateLimitMiddleware(), handler.RegisterHandler(userService, emailService))

		// Email verification (public)
		auth.POST("/verify/resend", handler.NewAuthHandler(userService, emailService).ResendVerification())
		auth.POST("/verify/code", handler.NewAuthHandler(userService, emailService).VerifyCode())
		auth.GET("/verify/:token", handler.NewAuthHandler(userService, emailService).VerifyEmail())

		// Password reset (public)
		auth.POST("/reset/request", handler.NewAuthHandler(userService, emailService).RequestPasswordReset())
		auth.POST("/reset/verify", handler.NewAuthHandler(userService, emailService).VerifyResetToken())
		auth.POST("/reset/confirm", handler.NewAuthHandler(userService, emailService).ResetPassword())
	}

	authProtected := r.Group("/auth", authh.AuthMiddleware())
	{
		authProtected.POST("/logout", handler.LogoutHandler())
		authProtected.POST("/verify/send", handler.NewAuthHandler(userService, emailService).SendVerificationEmail())
		authProtected.POST("/password/change", handler.NewAuthHandler(userService, emailService).ChangePassword())
	}
}
