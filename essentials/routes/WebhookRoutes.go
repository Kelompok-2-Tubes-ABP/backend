package routes

import (
	"financeapi/essentials/handler"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func RegisterWebhookRoutes(r *gin.Engine, db *mongo.Database) {
	webhookHandler := handler.NewWebhookHandler(db)
	webhookHandler.SetupRoutes(r)
}
