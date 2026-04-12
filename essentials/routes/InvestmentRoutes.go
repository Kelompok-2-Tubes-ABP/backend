package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterInvestmentRoutes(r *gin.Engine, investmentService *services.InvestmentService, priceHandler *handler.PriceHandler) {
	// Public Price routes
	r.GET("/prices/crypto", priceHandler.GetCryptoPrices())
	r.GET("/prices/crypto/:symbol", priceHandler.GetCryptoPrice())
	r.GET("/prices/stocks", priceHandler.GetStockPrices())
	r.GET("/prices/stock/:symbol", priceHandler.GetStockPrice())
	r.GET("/prices/all", priceHandler.GetAllPrices())

	// Setup Investment Handler
	investmentHandler := handler.NewInvestmentHandler(investmentService)
	investmentProtected := r.Group("/investment", authh.AuthMiddleware())
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

		// === Real-Time Price Endpoints ===
		investmentProtected.POST("/refresh-prices", investmentHandler.RefreshAllPrices())
		investmentProtected.POST("/:id/refresh-price", investmentHandler.RefreshSinglePrice())
		investmentProtected.GET("/portfolio", investmentHandler.GetPortfolioWithLivePrices())
		investmentProtected.GET("/summary-live", investmentHandler.GetPortfolioSummaryWithLivePrices())
		investmentProtected.POST("/transaction", investmentHandler.AddTransaction())
		investmentProtected.GET("/:id/transactions", investmentHandler.GetInvestmentTransactions())
	}
}
