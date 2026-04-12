package main

import (
	config "financeapi/essentials/config"
	handler "financeapi/essentials/handler"
	"financeapi/essentials/routes"
	services "financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	godotenv.Load()

	router := gin.Default()
	client := config.ConnectDB()

	// CORS Middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Email Service
	emailService := services.NewEmailService(client.Database("mydb"))

	txService := services.NewTransactionService(client, "mydb")
	userService := services.NewUserService(client, "mydb")
	chatbotService := services.NewChatService(client, "mydb", txService)

	// Investment services
	investmentService := services.NewInvestmentService(client, "mydb")
	currencyService := services.NewCurrencyService(client, "mydb")
	priceService := services.NewPriceService(currencyService)
	investmentService.SetPriceService(priceService)

	// Price handler for public price endpoints
	priceHandler := handler.NewPriceHandler(priceService)

	// Additional services for chatbot
	savingsGoalService := services.NewSavingsGoalService(client, "mydb")
	spendingInsightService := services.NewSpendingInsightService(client, "mydb")
	billReminderService := services.NewBillReminderService(client, "mydb")

	// Initialize all services
	debtService := services.NewDebtService(client, "mydb")
	accountService := services.NewAccountService(client, "mydb")
	recurringService := services.NewRecurringTransactionService(client, "mydb")

	// Analytics Service
	analyticsService := services.NewAnalyticsService()
	analyticsService.SetTransactionService(txService)
	analyticsService.SetInvestmentService(investmentService)
	analyticsService.SetDebtService(debtService)
	analyticsService.SetSavingsGoalService(savingsGoalService)
	analyticsService.SetBillReminderService(billReminderService)
	analyticsService.SetPriceService(priceService)

	// Budget Service
	budgetService := services.NewBudgetService(client, "mydb")

	// Inject services into chatbot
	chatbotService.SetInvestmentService(investmentService)
	chatbotService.SetPriceService(priceService)
	chatbotService.SetSavingsGoalService(savingsGoalService)
	chatbotService.SetSpendingInsightService(spendingInsightService)
	chatbotService.SetBillReminderService(billReminderService)
	chatbotService.SetDebtService(debtService)
	chatbotService.SetRecurringTransactionService(recurringService)
	chatbotService.SetAccountService(accountService)
	chatbotService.SetBudgetService(budgetService)

	// Register Modular Routes
	routes.RegisterAuthRoutes(router, userService, emailService)
	routes.RegisterProfileRoutes(router, userService)
	routes.RegisterTransactionRoutes(router, txService)
	routes.RegisterChatbotRoutes(router, chatbotService)

	routes.RegisterSavingsGoalRoutes(router, savingsGoalService)
	routes.RegisterWebhookRoutes(router, config.GetDB(client))
	routes.RegisterInvestmentRoutes(router, investmentService, priceHandler)
	routes.RegisterAnalyticsRoutes(router, analyticsService)

	routes.RegisterBudgetRoutes(router, budgetService)
	routes.RegisterAccountRoutes(router, accountService)
	routes.RegisterDebtRoutes(router, debtService)
	routes.RegisterRecurringRoutes(router, recurringService)
	routes.RegisterBillRoutes(router, billReminderService)
	routes.RegisterInsightRoutes(router, spendingInsightService)

	router.Run()
}
