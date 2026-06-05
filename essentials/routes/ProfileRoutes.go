package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterProfileRoutes(r *gin.Engine, userService *services.UserService) {
	profileProtected := r.Group("/profile", authh.AuthMiddleware())
	{
		profileProtected.GET("/", handler.ProfileHandler(userService))
		profileProtected.PATCH("/", handler.UpdateProfileHandler(userService))
	}
}
