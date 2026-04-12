package main

import (
	authh "financeapi/essentials/auth"
	config "financeapi/essentials/config"
	handler "financeapi/essentials/handler"
	"financeapi/essentials/middleware"
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

	// Analytics Service
	analyticsService := services.NewAnalyticsService()
	analyticsService.SetTransactionService(txService)
	analyticsService.SetInvestmentService(investmentService)
	analyticsService.SetDebtService(services.NewDebtService(client, "mydb"))
	analyticsService.SetSavingsGoalService(savingsGoalService)
	analyticsService.SetBillReminderService(billReminderService)
	analyticsService.SetPriceService(priceService)

	// Budget Service
	budgetService := services.NewBudgetService(client, "mydb")
	budgetHandler := handler.NewBudgetHandler(budgetService)

	// Inject services into chatbot
	chatbotService.SetInvestmentService(investmentService)
	chatbotService.SetPriceService(priceService)
	chatbotService.SetSavingsGoalService(savingsGoalService)
	chatbotService.SetSpendingInsightService(spendingInsightService)
	chatbotService.SetBillReminderService(billReminderService)

	auth := router.Group("/auth")
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
	authProtected := router.Group("/auth", authh.AuthMiddleware())
	{
		authProtected.POST("/logout", handler.LogoutHandler())
		authProtected.POST("/verify/send", handler.NewAuthHandler(userService, emailService).SendVerificationEmail())
		authProtected.POST("/password/change", handler.NewAuthHandler(userService, emailService).ChangePassword())
	}
	profileProtected := router.Group("/profile", authh.AuthMiddleware())
	{
		profileProtected.GET("/", handler.ProfileHandler(userService))
	}
	transactionProtected := router.Group("/transaction", authh.AuthMiddleware())
	{
		transactionProtected.POST("/new", handler.CreateTransactionHandler(txService))
		transactionProtected.DELETE("/delete/:id", handler.DeleteTransactionHandler(txService))
		transactionProtected.GET("/", handler.GetAllTransaction(txService))
		transactionProtected.PATCH("/update/:id", handler.UpdateTransactionHandler(txService))
		transactionProtected.GET("/filter", handler.FilterByCategoryHandler(txService))
		transactionProtected.GET("/getMonthly", handler.GetMonthlyHandler(txService))
		transactionProtected.GET("/getReport", handler.GetReportHandler(txService))
	}

	// Chatbot routes
	chatbotProtected := router.Group("/chatbot", authh.AuthMiddleware())
	{
		chatbotProtected.POST("/message", handler.NewChatbotHandler(chatbotService).ProcessMessage())
		chatbotProtected.GET("/history", handler.NewChatbotHandler(chatbotService).GetChatHistory())
		chatbotProtected.GET("/summary", handler.NewChatbotHandler(chatbotService).GetFinancialSummary())
		chatbotProtected.GET("/ask", handler.NewChatbotHandler(chatbotService).QuickAsk())
		chatbotProtected.DELETE("/clear", handler.NewChatbotHandler(chatbotService).ClearConversation())
	}

	// Savings Goal Service - create handler (service already created above)
	savingsGoalHandler := handler.NewSavingsGoalHandler(savingsGoalService)

	// Savings Goal routes
	savingsProtected := router.Group("/savings_goal", authh.AuthMiddleware())
	{
		savingsProtected.POST("/", savingsGoalHandler.CreateSavingsGoal())
		savingsProtected.GET("/get", savingsGoalHandler.GetUserSavingsGoals())
		savingsProtected.GET("/summary", savingsGoalHandler.GetSavingsSummary())
		savingsProtected.GET("/get/:id", savingsGoalHandler.GetSavingsGoal())
		savingsProtected.PATCH("/update/:id", savingsGoalHandler.UpdateSavingsGoal())
		savingsProtected.POST("/:id/contribute", savingsGoalHandler.AddContribution())
		savingsProtected.GET("/:id/contributions", savingsGoalHandler.GetContributions())
		savingsProtected.DELETE("/:id", savingsGoalHandler.DeleteSavingsGoal())
	}

	// Webhook routes
	webhookHandler := handler.NewWebhookHandler(config.GetDB(client))
	webhookHandler.SetupRoutes(router)

	// Public Price routes (no auth needed - for frontend to fetch prices)
	router.GET("/prices/crypto", priceHandler.GetCryptoPrices())
	router.GET("/prices/crypto/:symbol", priceHandler.GetCryptoPrice())
	router.GET("/prices/stocks", priceHandler.GetStockPrices())
	router.GET("/prices/stock/:symbol", priceHandler.GetStockPrice())
	router.GET("/prices/all", priceHandler.GetAllPrices())

	// Setup Investment Handler
	investmentHandler := handler.NewInvestmentHandler(investmentService)
	investmentProtected := router.Group("/investment", authh.AuthMiddleware())
	{
		// Basic CRUD
		investmentProtected.POST("/", investmentHandler.CreateInvestment())
		investmentProtected.GET("/", investmentHandler.GetUserInvestments())
		investmentProtected.GET("/:id", investmentHandler.GetInvestment())
		investmentProtected.DELETE("/:id", investmentHandler.DeleteInvestment())

		// Manual price update
		investmentProtected.PATCH("/:id/price", investmentHandler.UpdatePrice())

		// Portfolio
		investmentProtected.GET("/summary", investmentHandler.GetPortfolioSummary())

		// === NEW: Real-Time Price Endpoints ===
		// Refresh semua harga dari API (CoinGecko untuk crypto, Finnhub untuk stocks)
		investmentProtected.POST("/refresh-prices", investmentHandler.RefreshAllPrices())

		// Refresh harga satu investasi
		investmentProtected.POST("/:id/refresh-price", investmentHandler.RefreshSinglePrice())

		// Portfolio dengan harga real-time
		investmentProtected.GET("/portfolio", investmentHandler.GetPortfolioWithLivePrices())

		// Summary dengan harga real-time
		investmentProtected.GET("/summary-live", investmentHandler.GetPortfolioSummaryWithLivePrices())
		investmentProtected.POST("/transaction", investmentHandler.AddTransaction())
		investmentProtected.GET("/:id/transactions", investmentHandler.GetInvestmentTransactions())
	}

	// Analytics routes
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	analyticsProtected := router.Group("/analytics", authh.AuthMiddleware())
	{
		analyticsProtected.GET("/", analyticsHandler.GetFullAnalytics())
		analyticsProtected.GET("/quick", analyticsHandler.GetQuickStats())
		analyticsProtected.GET("/goals", analyticsHandler.GetGoalProgress())
		analyticsProtected.GET("/net-worth", analyticsHandler.GetNetWorthDetail())
	}

	// Budget routes
	budgetProtected := router.Group("/budget", authh.AuthMiddleware())
	{
		// Basic CRUD - root level
		budgetProtected.POST("/", budgetHandler.CreateBudget())
		budgetProtected.GET("/", budgetHandler.GetUserBudgets())

		// Summary & reports
		budgetProtected.GET("/summary", budgetHandler.GetBudgetSummary())
		budgetProtected.GET("/all-spending", budgetHandler.GetAllBudgetsWithSpending())

		// Monthly routes (specific path - no conflict)
		budgetProtected.GET("/by-month/:month", budgetHandler.GetBudgetByMonth())
		budgetProtected.GET("/by-month/:month/spending", budgetHandler.GetBudgetWithSpending())

		// ID-based routes (wildcard - must be last)
		budgetProtected.GET("/:id", budgetHandler.GetBudget())
		budgetProtected.PATCH("/:id", budgetHandler.UpdateBudget())
		budgetProtected.DELETE("/:id", budgetHandler.DeleteBudget())

		// Category Budget routes
		budgetProtected.POST("/category", budgetHandler.CreateCategoryBudget())
		budgetProtected.GET("/category", budgetHandler.GetUserCategoryBudgets())
		budgetProtected.GET("/category/summary", budgetHandler.GetCategoryBudgetSummary())

		// Category monthly routes (specific path)
		budgetProtected.GET("/category/by-month/:month", budgetHandler.GetCategoryBudgetByMonth())
		budgetProtected.GET("/category/by-month/:month/all", budgetHandler.GetAllCategoryBudgetsWithSpending())
		budgetProtected.GET("/category/by-month/:month/:category/spending", budgetHandler.GetCategoryBudgetWithSpending())

		// Category ID routes (wildcard - last)
		budgetProtected.GET("/category/:id", budgetHandler.GetCategoryBudget())
		budgetProtected.PATCH("/category/:id", budgetHandler.UpdateCategoryBudget())
		budgetProtected.DELETE("/category/:id", budgetHandler.DeleteCategoryBudget())
	}

	// Account routes - create once and reuse
	accountService := services.NewAccountService(client, "mydb")
	accountHandler := handler.NewAccountHandler(accountService)
	accountProtected := router.Group("/account", authh.AuthMiddleware())
	{
		accountProtected.POST("/", accountHandler.CreateAccount())
		accountProtected.GET("/", accountHandler.GetUserAccounts())
		accountProtected.GET("/summary", accountHandler.GetAccountSummary())
		accountProtected.GET("/type/:type", accountHandler.GetAccountsByType())
		accountProtected.GET("/groups", accountHandler.GetAccountGroups())
		accountProtected.POST("/groups", accountHandler.CreateAccountGroup())
		accountProtected.GET("/:id", accountHandler.GetAccount())
		accountProtected.PATCH("/:id", accountHandler.UpdateAccount())
		accountProtected.PATCH("/:id/balance", accountHandler.UpdateBalance())
		accountProtected.POST("/:id/sync", accountHandler.SyncAccount())
		accountProtected.DELETE("/:id", accountHandler.DeleteAccount())
		accountProtected.POST("/transfer", accountHandler.Transfer())
		accountProtected.GET("/transfers", accountHandler.GetTransferHistory())
	}

	// Debt routes
	debtService := services.NewDebtService(client, "mydb")
	debtHandler := handler.NewDebtHandler(debtService)
	debtProtected := router.Group("/debt", authh.AuthMiddleware())
	{
		debtProtected.POST("/", debtHandler.CreateDebt())
		debtProtected.GET("/", debtHandler.GetUserDebts())
		debtProtected.GET("/summary", debtHandler.GetDebtSummary())
		debtProtected.GET("/:id", debtHandler.GetDebt())
		debtProtected.GET("/:id/history", debtHandler.GetPaymentHistory())
		debtProtected.POST("/:id/pay", debtHandler.MakePayment())
		debtProtected.DELETE("/:id", debtHandler.DeleteDebt())
	}

	// Recurring Transaction routes
	recurringService := services.NewRecurringTransactionService(client, "mydb")
	recurringHandler := handler.NewRecurringTransactionHandler(recurringService)
	recurringProtected := router.Group("/recurring", authh.AuthMiddleware())
	{
		recurringProtected.POST("/", recurringHandler.CreateRecurringTransaction())
		recurringProtected.GET("/", recurringHandler.GetUserRecurringTransactions())
		recurringProtected.GET("/active", recurringHandler.GetActiveRecurringTransactions())
		recurringProtected.GET("/due", recurringHandler.GetDueRecurringTransactions())
		recurringProtected.GET("/summary", recurringHandler.GetRecurringSummary())
		recurringProtected.GET("/:id", recurringHandler.GetRecurringTransaction())
		recurringProtected.GET("/:id/generated", recurringHandler.GetGeneratedTransactions())
		recurringProtected.PATCH("/:id", recurringHandler.UpdateRecurringTransaction())
		recurringProtected.DELETE("/:id", recurringHandler.DeleteRecurringTransaction())
		recurringProtected.POST("/:id/skip", recurringHandler.SkipNextRun())
		recurringProtected.POST("/:id/pause", recurringHandler.PauseRecurringTransaction())
		recurringProtected.POST("/:id/resume", recurringHandler.ResumeRecurringTransaction())
		recurringProtected.POST("/process", recurringHandler.ProcessDueTransactions())
	}

	// Bill Reminder routes
	billHandler := handler.NewBillReminderHandler(billReminderService)
	billProtected := router.Group("/bill", authh.AuthMiddleware())
	{
		billProtected.POST("/", billHandler.CreateBillReminder())
		billProtected.GET("/", billHandler.GetUserBillReminders())
		billProtected.GET("/due", billHandler.GetDueBillReminders())
		billProtected.GET("/overdue", billHandler.GetOverdueBillReminders())
		billProtected.GET("/summary", billHandler.GetBillSummary())
		billProtected.GET("/:id", billHandler.GetBillReminder())
		billProtected.GET("/:id/history", billHandler.GetPaymentHistory())
		billProtected.PATCH("/:id", billHandler.UpdateBillReminder())
		billProtected.POST("/:id/pay", billHandler.MarkAsPaid())
		billProtected.DELETE("/:id", billHandler.DeleteBillReminder())
	}

	// Spending Insight routes
	insightHandler := handler.NewSpendingInsightHandler(spendingInsightService)
	insightProtected := router.Group("/insights", authh.AuthMiddleware())
	{
		insightProtected.POST("/create", insightHandler.CreateInsight())
		insightProtected.GET("/", insightHandler.GetInsights())
		insightProtected.GET("/health", insightHandler.GetHealthScore())
		insightProtected.POST("/:id/read", insightHandler.MarkAsRead())
		insightProtected.POST("/:id/action", insightHandler.MarkAsActioned())
	}

	router.Run()
}
