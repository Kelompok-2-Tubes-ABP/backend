package handler

import (
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

type PriceHandler struct {
	priceService *services.PriceService
}

func NewPriceHandler(priceService *services.PriceService) *PriceHandler {
	return &PriceHandler{priceService: priceService}
}

func (h *PriceHandler) GetCryptoPrices() gin.HandlerFunc {
	return func(c *gin.Context) {
		page := c.DefaultQuery("page", "1")
		perPage := c.DefaultQuery("per_page", "20")
		currency := c.DefaultQuery("currency", "idr")

		if currency != "idr" && currency != "usd" && currency != "eur" {
			c.JSON(400, gin.H{"error": "currency must be idr, usd, or eur"})
			return
		}

		pageNum := 1
		perPageNum := 20

		if page != "" {
			if p, err := parseInt(page); err == nil && p > 0 {
				pageNum = p
			}
		}
		if perPage != "" {
			if pp, err := parseInt(perPage); err == nil && pp > 0 && pp <= 250 {
				perPageNum = pp
			}
		}

		coins, err := h.priceService.GetTrendingCoins(currency, perPageNum, pageNum)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    coins,
			"pagination": gin.H{
				"page":     pageNum,
				"per_page": perPageNum,
				"total":    100,
			},
		})
	}
}

func (h *PriceHandler) GetStockPrices() gin.HandlerFunc {
	return func(c *gin.Context) {
		symbolsParam := c.Query("symbols")
		currency := c.DefaultQuery("currency", "idr")

		if currency != "idr" && currency != "usd" && currency != "eur" {
			c.JSON(400, gin.H{"error": "currency must be idr, usd, or eur"})
			return
		}

		defaultStocks := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN", "NVDA", "META", "NFLX", "AMD", "INTC"}
		stocks := defaultStocks

		if symbolsParam != "" {
			stocks = splitSymbols(symbolsParam)
		}

		convertToIDR := currency == "idr"

		type StockPrice struct {
			Symbol    string  `json:"symbol"`
			Name      string  `json:"name"`
			Price     float64 `json:"price"`
			Currency  string  `json:"currency"`
			Change    float64 `json:"change,omitempty"`
			ChangePct float64 `json:"change_pct,omitempty"`
		}

		stockNames := map[string]string{
			"AAPL":  "Apple",
			"GOOGL": "Google",
			"MSFT":  "Microsoft",
			"TSLA":  "Tesla",
			"AMZN":  "Amazon",
			"NVDA":  "Nvidia",
			"META":  "Meta",
			"NFLX":  "Netflix",
			"AMD":   "AMD",
			"INTC":  "Intel",
		}

		var results []StockPrice
		for _, symbol := range stocks {
			price, err := h.priceService.GetStockPrice(symbol, convertToIDR)
			if err == nil {
				name := symbol
				if n, ok := stockNames[symbol]; ok {
					name = n
				}
				results = append(results, StockPrice{
					Symbol:   symbol,
					Name:     name,
					Price:    price,
					Currency: currency,
				})
			}
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    results,
		})
	}
}

func (h *PriceHandler) GetCryptoPrice() gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := c.Param("symbol")
		if symbol == "" {
			c.JSON(400, gin.H{"error": "symbol is required"})
			return
		}

		currency := c.DefaultQuery("currency", "idr")
		if currency != "idr" && currency != "usd" && currency != "eur" {
			c.JSON(400, gin.H{"error": "currency must be idr, usd, or eur"})
			return
		}

		price, err := h.priceService.GetCryptoPrice(symbol, currency)
		if err != nil {
			c.JSON(404, gin.H{
				"success": false,
				"error":   "Crypto not found: " + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data": gin.H{
				"symbol":   symbol,
				"price":    price,
				"currency": currency,
			},
		})
	}
}

func (h *PriceHandler) GetStockPrice() gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := c.Param("symbol")
		if symbol == "" {
			c.JSON(400, gin.H{"error": "symbol is required"})
			return
		}

		currency := c.DefaultQuery("currency", "idr")
		if currency != "idr" && currency != "usd" && currency != "eur" {
			c.JSON(400, gin.H{"error": "currency must be idr, usd, or eur"})
			return
		}

		convertToIDR := currency == "idr"

		price, err := h.priceService.GetStockPrice(symbol, convertToIDR)
		if err != nil {
			c.JSON(404, gin.H{
				"success": false,
				"error":   "Stock not found: " + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data": gin.H{
				"symbol":   symbol,
				"price":    price,
				"currency": currency,
			},
		})
	}
}

func (h *PriceHandler) GetAllPrices() gin.HandlerFunc {
	return func(c *gin.Context) {
		currency := c.DefaultQuery("currency", "idr")
		if currency != "idr" && currency != "usd" && currency != "eur" {
			c.JSON(400, gin.H{"error": "currency must be idr, usd, or eur"})
			return
		}

		convertToIDR := currency == "idr"

		coins, _ := h.priceService.GetTrendingCoins(currency, 20, 1)

		stocks := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN", "NVDA", "META", "NFLX"}
		stockNames := map[string]string{
			"AAPL": "Apple", "GOOGL": "Google", "MSFT": "Microsoft",
			"TSLA": "Tesla", "AMZN": "Amazon", "NVDA": "Nvidia",
			"META": "Meta", "NFLX": "Netflix",
		}

		type StockPrice struct {
			Symbol   string  `json:"symbol"`
			Name     string  `json:"name"`
			Price    float64 `json:"price"`
			Currency string  `json:"currency"`
		}

		var stockResults []StockPrice
		for _, symbol := range stocks {
			price, err := h.priceService.GetStockPrice(symbol, convertToIDR)
			if err == nil {
				name := symbol
				if n, ok := stockNames[symbol]; ok {
					name = n
				}
				stockResults = append(stockResults, StockPrice{
					Symbol:   symbol,
					Name:     name,
					Price:    price,
					Currency: currency,
				})
			}
		}

		c.JSON(200, gin.H{
			"success": true,
			"data": gin.H{
				"crypto": coins,
				"stocks": stockResults,
			},
		})
	}
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func splitSymbols(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else if c != ' ' {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
