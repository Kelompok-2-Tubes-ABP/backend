package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PriceService struct {
	coinGeckoURL    string
	coinGeckoAPIKey string

	finnhubURL    string
	finnhubAPIKey string

	currencyService *CurrencyService
}

type CoinGeckoResponse map[string]map[string]float64

type FinnhubQuoteResponse struct {
	C  float64 `json:"c"`
	D  float64 `json:"d"`
	DP float64 `json:"dp"`
	H  float64 `json:"h"`
	L  float64 `json:"l"`
	O  float64 `json:"o"`
	PC float64 `json:"pc"`
	TS int64   `json:"t"`
}

type CoinGeckoMarketItem struct {
	ID             string  `json:"id"`
	Symbol         string  `json:"symbol"`
	Name           string  `json:"name"`
	CurrentPrice   float64 `json:"current_price"`
	PriceChange24H float64 `json:"price_change_24h"`
	PriceChangePct float64 `json:"price_change_percentage_24h"`
	MarketCap      float64 `json:"market_cap"`
	Volume         float64 `json:"total_volume"`
	Image          string  `json:"image"`
}

type CoinGeckoMarketResponse []CoinGeckoMarketItem

var CryptoSymbolMapping = map[string]string{
	"btc":   "bitcoin",
	"eth":   "ethereum",
	"sol":   "solana",
	"ada":   "cardano",
	"dot":   "polkadot",
	"doge":  "dogecoin",
	"xrp":   "ripple",
	"avax":  "avalanche-2",
	"matic": "matic-network",
	"link":  "chainlink",
	"uni":   "uniswap",
	"ltc":   "litecoin",
	"bch":   "bitcoin-cash",
	"xlm":   "stellar",
	"atom":  "cosmos",
	"trx":   "tron",
	"etc":   "ethereum-classic",
	"xmr":   "monero",
	"shib":  "shiba-inu",
	"pepe":  "pepe",
	"ar":    "arweave",
	"inj":   "injective-protocol",
	"apt":   "aptos",
	"arb":   "arbitrum",
	"op":    "optimism",
	"sand":  "the-sandbox",
	"mana":  "decentraland",
}

func NewPriceService(currencyService *CurrencyService) *PriceService {
	return &PriceService{
		coinGeckoURL:    "https://api.coingecko.com/api/v3",
		coinGeckoAPIKey: os.Getenv("GECKO_API_KEY"),

		finnhubURL:    "https://finnhub.io/api/v1",
		finnhubAPIKey: os.Getenv("FINNHUB_API_KEY"),

		currencyService: currencyService,
	}
}

func (s *PriceService) GetCryptoPrices(symbols []string, currency string) (map[string]float64, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("symbols cannot be empty")
	}

	lowerSymbols := make([]string, len(symbols))
	for i, sym := range symbols {
		lowerSymbols[i] = strings.ToLower(sym)
	}

	coinIDs := make([]string, len(lowerSymbols))
	for i, sym := range lowerSymbols {
		if id, ok := CryptoSymbolMapping[sym]; ok {
			coinIDs[i] = id
		} else {
			coinIDs[i] = sym
		}
	}

	ids := strings.Join(coinIDs, ",")
	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=%s",
		s.coinGeckoURL, ids, currency)

	if s.coinGeckoAPIKey != "" {
		url += "&x_cg_demo_api_key=" + s.coinGeckoAPIKey
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prices: %w", err)
	}
	defer resp.Body.Close()

	var result CoinGeckoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	prices := make(map[string]float64)

	for _, symbol := range lowerSymbols {
		coinID := symbol
		if id, ok := CryptoSymbolMapping[symbol]; ok {
			coinID = id
		}

		if coinData, ok := result[coinID]; ok {
			if price, ok := coinData[currency]; ok {
				prices[symbol] = price
			}
		}
	}

	return prices, nil
}

func (s *PriceService) GetCryptoPrice(symbol string, currency string) (float64, error) {
	prices, err := s.GetCryptoPrices([]string{symbol}, currency)
	if err != nil {
		return 0, err
	}

	price, ok := prices[strings.ToLower(symbol)]
	if !ok {
		return 0, fmt.Errorf("price not found for symbol: %s", symbol)
	}

	return price, nil
}

func (s *PriceService) GetStockPrice(symbol string, convertToIDR bool) (float64, error) {
	if symbol == "" {
		return 0, fmt.Errorf("symbol cannot be empty")
	}

	url := fmt.Sprintf("%s/quote?symbol=%s&token=%s",
		s.finnhubURL, strings.ToUpper(symbol), s.finnhubAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch stock price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result FinnhubQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if convertToIDR {
		rate, err := s.currencyService.GetExchangeRate("USD", "IDR")
		if err != nil {
			fmt.Printf("Warning: Failed to get real-time USD/IDR rate: %v. Using fallback.\n", err)
		}

		return result.C * rate.Rate, nil
	}

	return result.C, nil
}

func (s *PriceService) GetStockPrices(symbols []string, convertToIDR bool) (map[string]float64, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("symbols cannot be empty")
	}

	prices := make(map[string]float64)
	priceChan := make(chan struct {
		symbol string
		price  float64
		err    error
	}, len(symbols))

	// Launch parallel requests
	for _, symbol := range symbols {
		go func(sym string) {
			price, err := s.GetStockPrice(sym, convertToIDR)
			priceChan <- struct {
				symbol string
				price  float64
				err    error
			}{sym, price, err}
		}(symbol)
	}

	var errors []string
	for i := 0; i < len(symbols); i++ {
		res := <-priceChan
		if res.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", res.symbol, res.err))
		} else {
			prices[strings.ToUpper(res.symbol)] = res.price
		}
	}

	if len(prices) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to fetch any prices: %s", strings.Join(errors, "; "))
	}

	return prices, nil
}

// ============================================================
// UNIFIED METHOD - Untuk Semua Tipe Investasi
// ============================================================

// GetPrice - Ambil harga berdasarkan tipe investasi
//
// params:
//   - symbol: symbol investasi (BTC, AAPL, dll)
//   - investmentType: tipe investasi (crypto, stock, dll)
//   - currency: mata uang target
//
// return:
func (s *PriceService) GetPrice(symbol string, investmentType string, currency string) (float64, error) {
	symbolLower := strings.ToLower(symbol)

	switch investmentType {
	case "crypto":
		return s.GetCryptoPrice(symbolLower, currency)

	case "stock":
		convertToIDR := (strings.ToUpper(currency) == "IDR")
		return s.GetStockPrice(symbol, convertToIDR)

	default:
		return 0, fmt.Errorf("unsupported investment type: %s", investmentType)
	}
}

type InvestmentWithPrice struct {
	ID              primitive.ObjectID `json:"id"`
	Name            string             `json:"name"`
	Symbol          string             `json:"symbol"`
	Type            string             `json:"type"`
	Quantity        float64            `json:"quantity"`
	AverageCost     float64            `json:"average_cost"`
	CurrentPrice    float64            `json:"current_price"`
	TotalCost       float64            `json:"total_cost"`
	TotalValue      float64            `json:"total_value"`
	GainLoss        float64            `json:"gain_loss"`
	GainLossPercent float64            `json:"gain_loss_percent"`
	Currency        string             `json:"currency"`
	Exchange        string             `json:"exchange"`
	LastPriceUpdate time.Time          `json:"last_price_update"`
}

func (s *PriceService) GetUserPortfolioWithPrices(userID primitive.ObjectID, currency string) ([]InvestmentWithPrice, error) {
	return []InvestmentWithPrice{}, nil
}

func (s *PriceService) RefreshInvestmentPrices(userID primitive.ObjectID, investmentIDs []primitive.ObjectID) (int, error) {
	return 0, nil
}

func (s *PriceService) GetCoinGeckoID(symbol string) (string, bool) {
	symbolLower := strings.ToLower(symbol)
	id, ok := CryptoSymbolMapping[symbolLower]
	return id, ok
}

func (s *PriceService) IsSupportedCrypto(symbol string) bool {
	_, ok := CryptoSymbolMapping[strings.ToLower(symbol)]
	return ok
}

func (s *PriceService) GetTrendingCoins(currency string, perPage, page int) ([]CoinGeckoMarketItem, error) {
	if perPage > 250 {
		perPage = 250
	}
	if perPage < 1 {
		perPage = 50
	}
	if page < 1 {
		page = 1
	}

	url := fmt.Sprintf("%s/coins/markets?vs_currency=%s&order=market_cap_desc&per_page=%d&page=%d&sparkline=false&price_change_percentage=24h",
		s.coinGeckoURL, currency, perPage, page)

	// Tambahkan API key jika ada
	if s.coinGeckoAPIKey != "" {
		url += "&x_cg_demo_api_key=" + s.coinGeckoAPIKey
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trending coins: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result CoinGeckoMarketResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}
