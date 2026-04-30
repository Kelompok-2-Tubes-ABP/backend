package routes

import (
	"financeapi/essentials/auth"
	"financeapi/essentials/handler"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func RegisterNotificationRoutes(r *gin.Engine, db *mongo.Database) {
	notificationHandler := handler.NewNotificationHandler(db)

	notifications := r.Group("/api/notifications/feed")
	notifications.Use(auth.AuthMiddleware())
	{
		notifications.GET("", notificationHandler.GetMyNotifications)
		notifications.PATCH("/:id/read", notificationHandler.MarkAsRead)
		notifications.PATCH("/read-all", notificationHandler.MarkAllAsRead)
	}
}
