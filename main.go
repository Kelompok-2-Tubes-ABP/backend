package main

import (
	"financeapi/essentials/chatbot_commands"
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
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

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
	adminService := services.NewAdminService(client, "mydb")
	notificationService := services.NewNotificationService(client.Database("mydb"))
	
	// Inject notifications into txService
	txService.SetNotificationService(notificationService)

	// Analytics Service
	analyticsService := services.NewAnalyticsService()
	analyticsService.SetTransactionService(txService)
	analyticsService.SetInvestmentService(investmentService)
	analyticsService.SetDebtService(debtService)
	analyticsService.SetSavingsGoalService(savingsGoalService)
	analyticsService.SetBillReminderService(billReminderService)
	analyticsService.SetPriceService(priceService)
	analyticsService.SetAccountService(accountService)

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

	// Register Chatbot Commands
	chatbotService.RegisterCommand("debt", chatbot_commands.NewDebtCommand(debtService, accountService))
	chatbotService.RegisterCommand("transaction", chatbot_commands.NewTransactionCommand(txService))
	chatbotService.RegisterCommand("savings", chatbot_commands.NewSavingsCommand(savingsGoalService))
	chatbotService.RegisterCommand("budget", chatbot_commands.NewBudgetCommand(budgetService))
	chatbotService.RegisterCommand("investment", chatbot_commands.NewInvestCommand(investmentService, priceService))
	chatbotService.RegisterCommand("spending", chatbot_commands.NewSpendingCommand(txService))
	chatbotService.RegisterCommand("bills", chatbot_commands.NewBillsCommand(billReminderService))
	chatbotService.RegisterCommand("recurring", chatbot_commands.NewRecurringCommand(recurringService))
	chatbotService.RegisterCommand("account", chatbot_commands.NewAccountCommand(accountService))
	chatbotService.RegisterCommand("health", chatbot_commands.NewHealthCommand(spendingInsightService, txService))

	// Register Modular Routes
	routes.RegisterAuthRoutes(router, userService, emailService)
	routes.RegisterProfileRoutes(router, userService)
	routes.RegisterTransactionRoutes(router, txService)
	routes.RegisterChatbotRoutes(router, chatbotService)

	routes.RegisterSavingsGoalRoutes(router, savingsGoalService)
	routes.RegisterWebhookRoutes(router, config.GetDB(client))
	routes.RegisterNotificationRoutes(router, config.GetDB(client))
	routes.RegisterInvestmentRoutes(router, investmentService, priceHandler)
	routes.RegisterAnalyticsRoutes(router, analyticsService)

	routes.RegisterBudgetRoutes(router, budgetService)
	routes.RegisterAccountRoutes(router, accountService)
	routes.RegisterDebtRoutes(router, debtService)
	routes.RegisterRecurringRoutes(router, recurringService)
	routes.RegisterBillRoutes(router, billReminderService)
	routes.RegisterInsightRoutes(router, spendingInsightService)
	routes.RegisterAdminRoutes(router, adminService, userService)

	// Serve testing UI on root
	router.GET("/", func(c *gin.Context) {
		c.File("./public/index.html")
	})

	// Serve user testing UI on /user-test
	router.GET("/user-test", func(c *gin.Context) {
		c.File("./public/user.html")
	})

	router.Run()
}
