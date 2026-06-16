package utils

import (
	"regexp"
	"strings"
)

// ParsedIntent represents extracted entities from user message
type ParsedIntent struct {
	Action    string  // "expense", "income", "query", "transfer", etc.
	Amount    float64 // Extracted amount
	Category  string  // "food", "transport", etc.
	Note      string  // Description/note
	Time      string  // "tadi", "kemarin", "bulan lalu", etc.
	RawMessage string // Original message preserved
}

// FuzzyMatch checks if any keyword matches with typo tolerance
func FuzzyMatch(input string, keywords []string, threshold float64) bool {
	input = strings.ToLower(input)
	words := strings.Fields(input)
	
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		// Exact match (including partial like "bli" in "bli makan")
		if strings.Contains(input, kwLower) {
			return true
		}
		// Check each word in input against keyword
		for _, word := range words {
			// Word to keyword similarity
			score := SimilarityScore(word, kwLower)
			// For very short typos (3-4 chars), be more lenient
			if len(word) <= 4 && len(kwLower) <= 4 {
				// Allow 1 char difference for short words
				if LevenshteinDistance(word, kwLower) <= 1 {
					return true
				}
			}
			if score >= threshold {
				return true
			}
		}
		// Also check if input is similar to keyword
		if SimilarityScore(input, kwLower) >= threshold {
			return true
		}
	}
	return false
}

// LevenshteinDistance calculates edit distance between two strings
func LevenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	r1 := []rune(s1)
	r2 := []rune(s2)

	cost := 0
	if r1[len(r1)-1] != r2[len(r2)-1] {
		cost = 1
	}

	min := func(a, b, c int) int {
		if a < b {
			if a < c {
				return a
			}
			return c
		}
		if b < c {
			return b
		}
		return c
	}

	// Create matrix
	matrix := make([][]int, len(r1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(r2)+1)
	}

	// Initialize
	for i := 0; i <= len(r1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(r2); j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(r1); i++ {
		for j := 1; j <= len(r2); j++ {
			if r1[i-1] == r2[j-1] {
				matrix[i][j] = matrix[i-1][j-1]
			} else {
				matrix[i][j] = min(
					matrix[i-1][j]+1,      // deletion
					matrix[i][j-1]+1,        // insertion
					matrix[i-1][j-1]+cost,   // substitution
				)
			}
		}
	}

	return matrix[len(r1)][len(r2)]
}

// SimilarityScore calculates similarity between 0 and 1
func SimilarityScore(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	distance := LevenshteinDistance(s1, s2)
	return 1.0 - (float64(distance) / float64(maxLen))
}

// ExtractEntities extracts structured data from natural language
func ExtractEntities(message string) ParsedIntent {
	msg := strings.ToLower(message)

	intent := ParsedIntent{
		RawMessage: message,
	}

	// Extract amount
	intent.Amount = ParseIndonesianAmount(message)

	// Extract category
	intent.Category = ExtractExpenseCategory(message)

	// Extract time expression
	intent.Time = ExtractTimeExpression(message)

	// Detect action type
	intent.Action = detectActionType(msg)

	// Extract note/description
	intent.Note = ExtractTransactionDescription(message)

	return intent
}

// detectActionType determines the action type from message
func detectActionType(msg string) string {
	// Expense keywords
	expenseKeywords := []string{"beli", "bayar", "bayarin", "belanja", "makan", "lunch", "dinner", "purchase", "ngeluarin", "keluar", "habis", "spent", "checkout", "pembelian"}
	if FuzzyMatch(msg, expenseKeywords, 0.8) {
		return "expense"
	}

	// Income keywords
	incomeKeywords := []string{"gaji", "gajian", "income", "pemasukan", "dapet", "dapat", "uang masuk", "salary", "earned", "terima"}
	if FuzzyMatch(msg, incomeKeywords, 0.8) {
		return "income"
	}

	// Transfer keywords
	transferKeywords := []string{"transfer", "kirim", "kirim uang", "transfer ke"}
	if FuzzyMatch(msg, transferKeywords, 0.8) {
		return "transfer"
	}

	// Query keywords
	queryKeywords := []string{"?", "bagaimana", "apa", "kenapa", "berapa", "cek", "lihat", "show", "tolong"}
	if FuzzyMatch(msg, queryKeywords, 0.8) {
		return "query"
	}

	// Create keywords
	createKeywords := []string{"tambah", "buat", "bikin", "daftar", "create", "new"}
	if FuzzyMatch(msg, createKeywords, 0.8) {
		return "create"
	}

	// Delete keywords
	deleteKeywords := []string{"hapus", "delete", "batal", "cancel", "remove"}
	if FuzzyMatch(msg, deleteKeywords, 0.8) {
		return "delete"
	}

	// Update keywords
	updateKeywords := []string{"edit", "ubah", "update", "ganti", "change", "modify"}
	if FuzzyMatch(msg, updateKeywords, 0.8) {
		return "update"
	}

	return "unknown"
}

// DetectIntent detects the primary intent from message
func DetectIntent(message string) string {
	msg := strings.ToLower(message)

	// === PRIORITY: Action + Object pattern ===
	// "tambah/buat" + "budget" → budget
	if FuzzyMatch(msg, []string{"tambah", "buat", "bikin", "buatkan", "ingin", "butuh"}, 0.8) && 
	   strings.Contains(msg, "budget") {
		return "budget"
	}
	// "edit/ubah" + "budget" → budget
	if FuzzyMatch(msg, []string{"edit", "ubah", "update", "ganti"}, 0.8) && 
	   strings.Contains(msg, "budget") {
		return "budget"
	}
	// "hapus" + "budget" → budget
	if FuzzyMatch(msg, []string{"hapus", "delete"}, 0.8) && 
	   strings.Contains(msg, "budget") {
		return "budget"
	}

	// === Standalone object keywords ===
	// Budget alone
	if strings.Contains(msg, "budget") || strings.Contains(msg, "anggaran") {
		return "budget"
	}

	// Bills - "tagihan" should trigger bills
	if strings.Contains(msg, "tagihan") {
		return "bills"
	}
	if strings.Contains(msg, "bill") || strings.Contains(msg, "jatuh tempo") {
		return "bills"
	}

	// Transaction expense
	expenseWords := []string{"beli", "bayar", "bayarin", "purchase", "transaksi", "pengeluaran", 
		"spent", "keluar", "keluarkan", "habis", "makan"}
	if FuzzyMatch(msg, expenseWords, 0.8) {
		return "transaction"
	}
	// Transaction income
	incomeWords := []string{"gaji", "gajian", "income", "pemasukan", "dapet", "uang masuk", "salary", "terima"}
	if FuzzyMatch(msg, incomeWords, 0.8) {
		return "transaction"
	}

	// Account
	accountWords := []string{"saldo", "rekening", "e-wallet", "topup", "tarik"}
	for _, w := range accountWords {
		if strings.Contains(msg, w) {
			return "account"
		}
	}

	// Savings
	savingsWords := []string{"tabungan", "menabung", "nabung", "savings"}
	for _, w := range savingsWords {
		if strings.Contains(msg, w) {
			return "savings"
		}
	}

	// Investment
	investWords := []string{"crypto", "bitcoin", "saham", "invest", "trading"}
	for _, w := range investWords {
		if strings.Contains(msg, w) {
			return "investment"
		}
	}

	// Debt
	debtWords := []string{"hutang", "cicilan", "pinjaman", "kredit"}
	for _, w := range debtWords {
		if strings.Contains(msg, w) {
			return "debt"
		}
	}

	// Recurring
	if strings.Contains(msg, "langganan") || strings.Contains(msg, "subscription") || 
	   strings.Contains(msg, "recurring") {
		return "recurring"
	}

	// Spending analysis
	if strings.Contains(msg, "analisa") || strings.Contains(msg, "pola") || 
	   strings.Contains(msg, "spending") || strings.Contains(msg, "bulan ini") {
		return "spending"
	}

	// Health
	if strings.Contains(msg, "keuangan") || strings.Contains(msg, "summary") {
		return "health"
	}

	// Default → AI
	return "ai"
}

// DetectIntentWithConfidence returns intent and confidence score
func DetectIntentWithConfidence(message string) (string, float64) {
	msg := strings.ToLower(message)
	intent := DetectIntent(message)

	// Calculate confidence based on keyword strength
	confidence := 0.5 // Base confidence

	// Strong keywords (exact match or very similar)
	strongKeywords := map[string][]string{
		"budget":      {"budget", "anggaran", "limit budget"},
		"bills":       {"bill", "tagihan", "jatuh tempo"},
		"transaction": {"transaksi", "pengeluaran", "pemasukan", "gajian"},
		"investment":  {"crypto", "bitcoin", "saham", "stock"},
		"debt":        {"hutang", "cicilan", "pinjaman"},
		"savings":     {"tabungan", "savings goal", "target tabungan"},
	}

	if keywords, ok := strongKeywords[intent]; ok {
		for _, kw := range keywords {
			if strings.Contains(msg, kw) {
				confidence = 0.9
				break
			}
			if SimilarityScore(msg, kw) >= 0.85 {
				confidence = 0.8
				break
			}
		}
	}

	// Medium confidence for fuzzy matches
	if confidence == 0.5 {
		confidence = 0.6
	}

	// Low confidence for vague messages
	if len(msg) < 5 {
		confidence = 0.3
	}

	return intent, confidence
}

// ExtractTimeExpression extracts time references from message
func ExtractTimeExpression(message string) string {
	msg := strings.ToLower(message)

	timeExpressions := map[string]string{
		"tadi":       "tadi",
		"kemarin":    "kemarin",
		"kemarin kemarin": "kemarin",
		"hari ini":   "hari ini",
		"today":      "hari ini",
		"minggu ini": "minggu ini",
		"minggu lalu": "minggu lalu",
		"bulan ini":  "bulan ini",
		"bulan lalu": "bulan lalu",
		"tahun ini":  "tahun ini",
		"tahun lalu": "tahun lalu",
	}

	for expr, normalized := range timeExpressions {
		if strings.Contains(msg, expr) {
			return normalized
		}
	}

	// Check for "N waktu lalu" pattern
	re := regexp.MustCompile(`(\d+)\s*(?:menit|jam|hari|minggu|bulan|tahun)\s+lalu`)
	if matches := re.FindStringSubmatch(msg); len(matches) > 1 {
		return matches[0]
	}

	return ""
}

// IsReferenceKeyword checks if message contains reference words
func IsReferenceKeyword(message string) bool {
	msg := strings.ToLower(message)

	referenceKeywords := []string{"itu", "yang ini", "yang itu", "dengan itu", "ke itu", "dari itu", "tersebut"}

	for _, kw := range referenceKeywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}

	return false
}

// DetectIntentEnhanced detects intent with language awareness
func DetectIntentEnhanced(message string) (string, string) {
	lang := DetectLanguage(message)
	return DetectIntentByLanguage(message, lang), lang
}

// DetectIntentByLanguage detects intent using language-specific keywords
func DetectIntentByLanguage(message string, lang string) string {
	if lang == "en" {
		return detectEnglishIntent(message)
	}
	return DetectIntent(message)
}

// detectEnglishIntent detects intent for English messages
func detectEnglishIntent(msg string) string {
	// Transaction - expense/income
	expenseWords := []string{"bought", "paid", "spent", "purchase", "buy", 
		"checkout", "payment", "pay", "expense", "outgoing", 
		"withdraw", "withdrawal", "deduct", "charged", "got"}
	incomeWords := []string{"income", "salary", "earned", "received", "got paid", "payday",
		"deposit", "incoming", "revenue", "gain", "profit", "refund", "cashback"}
	
	if fuzzyMatchWords(msg, expenseWords) {
		return "transaction"
	}
	if fuzzyMatchWords(msg, incomeWords) {
		return "transaction"
	}
	
	// Budget
	budgetWords := []string{"budget", "budgeting", "allocation", "limit", "ceiling", "cap",
		"set budget", "create budget", "edit budget", "update budget", "check budget",
		"remaining budget", "budget left", "over budget", "within budget"}
	if fuzzyMatchWords(msg, budgetWords) {
		return "budget"
	}
	
	// Bills
	billWords := []string{"bill", "bills", "payment", "due date", "reminder",
		"utility", "utilities", "electricity", "internet bill", "phone bill",
		"subscription", "monthly bill", "pay bill", "outstanding"}
	if fuzzyMatchWords(msg, billWords) {
		return "bills"
	}
	
	// Account
	accountWords := []string{"balance", "account", "bank", "wallet", "e-wallet", 
		"top up", "topup", "withdraw", "withdrawal", "deposit", "transfer",
		"send money", "receive money", "cash", "funds"}
	if fuzzyMatchWords(msg, accountWords) {
		return "account"
	}
	
	// Savings
	savingsWords := []string{"save", "saving", "savings", "goal", "target", "target amount",
		"save up", "saving up", "set aside", "emergency fund", "put away", "stashing", "nest egg"}
	if fuzzyMatchWords(msg, savingsWords) {
		return "savings"
	}
	
	// Investment
	investWords := []string{"invest", "investing", "investment", "stock", "stocks",
		"shares", "portfolio", "trading", "trades", "crypto", "bitcoin", "ethereum",
		"dividend", "returns", "market", "buy stock", "sell stock", "buy crypto"}
	if fuzzyMatchWords(msg, investWords) {
		return "investment"
	}
	
	// Debt
	debtWords := []string{"debt", "loan", "loans", "borrow", "borrowed", "lend", "lending",
		"credit", "owe", "owes", "repay", "repayment", "installment", "interest rate",
		"mortgage", "car loan", "personal loan", "credit card"}
	if fuzzyMatchWords(msg, debtWords) {
		return "debt"
	}
	
	// Recurring
	recurringWords := []string{"recurring", "subscription", "monthly", "annual", "yearly",
		"auto", "automatic", "automated", "standing order", "direct debit",
		"regular payment", "periodic", "repeat", "membership"}
	if fuzzyMatchWords(msg, recurringWords) {
		return "recurring"
	}
	
	// Spending analysis
	spendingWords := []string{"spending", "expense", "expenses", "analysis", "analyze",
		"track", "tracking", "report", "summary", "breakdown", "categories",
		"monthly", "weekly", "daily", "this month", "last month", "trend",
		"compare", "comparison", "over time", "history"}
	if fuzzyMatchWords(msg, spendingWords) {
		return "spending"
	}
	
	// Health/Financial health
	healthWords := []string{"health", "financial health", "overview", "summary",
		"net worth", "assets", "liabilities", "financial status", "status",
		"how am i doing", "hows my finances", "financial report"}
	if fuzzyMatchWords(msg, healthWords) {
		return "health"
	}
	
	// Questions - send to AI
	questionWords := []string{"how", "what", "why", "when", "can i", "could i",
		"should i", "would you", "is it possible", "tell me", "explain",
		"tip", "tips", "suggest", "suggestion", "recommend", "advice", "?"}
	if fuzzyMatchWords(msg, questionWords) {
		return "ai"
	}
	
	// Greetings
	greetingWords := []string{"hi", "hello", "hey", "morning", "afternoon", "evening",
		"good morning", "good afternoon", "good evening", "howdy", "yo", "sup",
		"thanks", "thank you", "please", "sorry"}
	if fuzzyMatchWords(msg, greetingWords) && len(strings.Fields(msg)) <= 4 {
		return "ai"
	}
	
	return "ai"
}

// fuzzyMatchWords checks if any word matches with typo tolerance
func fuzzyMatchWords(msg string, keywords []string) bool {
	msg = strings.ToLower(msg)
	words := strings.Fields(msg)
	
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		// Exact substring match
		if strings.Contains(msg, kwLower) {
			return true
		}
		// Word-level fuzzy match
		for _, word := range words {
			if SimilarityScore(word, kwLower) >= 0.8 {
				return true
			}
		}
	}
	return false
}

// Sentiment represents the emotional state of a message
type Sentiment struct {
	Emotion  string   // "frustrated", "happy", "confused", "urgent", "satisfied", "neutral"
	Score    float64  // -1.0 (negative) to 1.0 (positive)
	Keywords []string // matched keywords
}

// DetectSentiment analyzes the emotional tone of a message
func DetectSentiment(message string) Sentiment {
	msg := strings.ToLower(message)
	
	// Keyword categories
	frustratedKeywords := []string{
		"anjing", "bangsat", "kesel", "kesal", "gila", "gile", "fuck", "shit", "damn", "wtf", "seriously",
		"frustrating", "annoying", "annoyed", "angry", "mad", "capek", "lelah", "pusing", "stress", "muak", "jengkel",
		"bodoh", "stupid", "dumb", "parah", "gak bisa", "cannot", "cant", "not working",
	}
	
	confusedKeywords := []string{
		"gimana", "gimana sih", "bingung", "confused", "dont understand", "don't understand",
		"not sure", "uncertain", "what do you mean", "huh", "apaan", "how do i", "how to", "caranya gimana",
	}
	
	happyKeywords := []string{
		"terima kasih", "thanks", "thank you", "makasih", "bagus", "good", "great", "mantap", "keren", "cool", "awesome", "amazing", "perfect",
		"luar biasa", "hebat", "nice", "best", "seneng", "senang", "happy", "glad", "yay", "yeay", "hore", "success", "sukses",
	}
	
	urgentKeywords := []string{
		"segera", "urgent", "emergency", "buru-buru", "buru", "asap", "now", "immediately", "darurat", "penting", "critical", "quick", "fast",
	}
	
	satisfiedKeywords := []string{
		"oke", "ok", "sip", "sep", "sounds good", "got it", "understood", "i see", "clear", "mantap", "okee", "beres", "selesai", "done",
	}
	
	// Count matches
	frustratedScore := 0
	confusedScore := 0
	happyScore := 0
	urgentScore := 0
	satisfiedScore := 0
	
	frustratedMatches := []string{}
	confusedMatches := []string{}
	happyMatches := []string{}
	urgentMatches := []string{}
	satisfiedMatches := []string{}
	
	for _, kw := range frustratedKeywords {
		if strings.Contains(msg, kw) {
			frustratedScore++
			frustratedMatches = append(frustratedMatches, kw)
		}
	}
	
	for _, kw := range confusedKeywords {
		if strings.Contains(msg, kw) {
			confusedScore++
			confusedMatches = append(confusedMatches, kw)
		}
	}
	
	for _, kw := range happyKeywords {
		if strings.Contains(msg, kw) {
			happyScore++
			happyMatches = append(happyMatches, kw)
		}
	}
	
	for _, kw := range urgentKeywords {
		if strings.Contains(msg, kw) {
			urgentScore++
			urgentMatches = append(urgentMatches, kw)
		}
	}
	
	for _, kw := range satisfiedKeywords {
		if strings.Contains(msg, kw) {
			satisfiedScore++
			satisfiedMatches = append(satisfiedMatches, kw)
		}
	}
	
	// Intensifiers
	if strings.Contains(msg, "banget") || strings.Contains(msg, "bgt") || strings.Contains(msg, "very") || strings.Contains(msg, "so much") || strings.Contains(msg, "really") {
		frustratedScore++
		happyScore++
	}
	
	// Determine dominant sentiment
	maxScore := frustratedScore
	emotion := "frustrated"
	
	if confusedScore > maxScore {
		maxScore = confusedScore
		emotion = "confused"
	}
	if happyScore > maxScore {
		maxScore = happyScore
		emotion = "happy"
	}
	if urgentScore > maxScore {
		maxScore = urgentScore
		emotion = "urgent"
	}
	if satisfiedScore > maxScore {
		maxScore = satisfiedScore
		emotion = "satisfied"
	}
	
	if maxScore == 0 {
		emotion = "neutral"
	}
	
	// Calculate score (-1 to 1)
	if maxScore > 0 {
		if happyScore > frustratedScore+confusedScore {
			return Sentiment{Emotion: emotion, Score: 0.5, Keywords: happyMatches}
		} else if frustratedScore > happyScore {
			return Sentiment{Emotion: emotion, Score: -0.5, Keywords: frustratedMatches}
		}
	}
	
	// Combine all matches
	allMatches := append(frustratedMatches, confusedMatches...)
	allMatches = append(allMatches, happyMatches...)
	allMatches = append(allMatches, urgentMatches...)
	allMatches = append(allMatches, satisfiedMatches...)
	
	return Sentiment{Emotion: emotion, Score: 0, Keywords: allMatches}
}

// GetSentimentResponse returns an empathetic response based on sentiment
func GetSentimentResponse(sentiment Sentiment) string {
	switch sentiment.Emotion {
	case "frustrated":
		return "Maaf bikin kamu kesal. Saya bantu sebisa mungkin ya! 🙏"
	case "confused":
		return "Saya bantu jelasin lebih detail ya. Ada yang kurang jelas?"
	case "happy":
		return "Senang bisa membantu! 😊"
	case "urgent":
		return "Saya tangani segera! ⚡"
	case "satisfied":
		return "👍"
	default:
		return ""
	}
}

// GetSentimentContext returns context info for AI to generate empathetic responses
func GetSentimentContext(sentiment Sentiment) map[string]interface{} {
	return map[string]interface{}{
		"sentiment":       sentiment.Emotion,
		"sentiment_score": sentiment.Score,
		"needs_empathy":   sentiment.Emotion == "frustrated" || sentiment.Emotion == "confused",
		"needs_urgency":   sentiment.Emotion == "urgent",
	}
}

// GetIntentKeywords returns enhanced keyword list for an intent
func GetIntentKeywords(intent string) []string {
	// Comprehensive keyword lists with Indonesian slang, abbreviations, and typos
	keywordMap := map[string][]string{
		"transaction": {
			// Original keywords
			"beli", "bayar", "purchase", "transaksi", "pengeluaran", "pemasukan", "gajian", "income", "spent", "keluar", "keluarkan", "transfer",
			// Typo variations
			"bli", "byar", "bayarin", "byr", "bliin", "beliin", "bayarin",
			// Slang and abbreviations
			"makan", "lunch", "dinner", "mkn", "makanm", "lunchh",
			"habis", "spent", "checkout", "checkout", "bought", "bayar ke",
			"ngeluarin", "ngeluarin", "keluarin", "keluarin", "keluarinn",
			// More slang
			"belanja", "blanja", "blanjaa", "shopping", "shop", "shoping",
			"dapet", "dapat", "dpt", "dapatt", "dptt", "gajian", "gjain",
			"gaji", "salary", "income", "pemasukan", "pmasukan", "uang masuk",
			"uang keluar", "duit keluar", "duit masuk", "duit", "duid",
			// Transaction types
			"taransaksi", "transksi", "transaki", "tranksaksi",
			// Food related
			"sarapan", "srg", "makan siang", "mkn siang", "makan malam", "mkn mlm",
			"breakfast", "brunch", "brunchh",
			// Common phrases
			"udah habis", "udh abis", "sudah habis", "habis nih",
		},
		"budget": {
			// Original keywords
			"budget", "anggaran", "limit budget", "planning budget",
			// Action variations
			"buat budget", "buatkan budget", "mau budget", "ingin budget", "butuh budget",
			"edit budget", "ubah budget", "update budget", "ganti budget", "hapus budget", "delete budget",
			// Typo/slang variations
			"bgt", "bgtu", "budgeting", "anggran", "anggarann",
			"buatin budget", "buate budget", "bikin budget", "bikn budget",
			"edit budget", "edt budget", "ubah budget", "ubh budget", "update budget", "updt budget",
			"ganti budget", "genti budget", "hapus budget", "hpus budget",
			// More variations
			"planning", "plan", "pln", "limit", "lmt", "ceiling",
			"masukkan budget", "tambah budget", "kurangi budget", "atur budget",
		},
		"bills": {
			// Original keywords
			"bill", "tagihan", "reminder", "jatuh tempo", "pembayaran", "bayar tagihan",
			// Typo/slang variations
			"tagihan", "tghn", "tgihan", "taghian", "bill", "bil",
			"reminder", "remindeer", "reminderr", "ranking",
			"jatuh tempo", "jth tempo", "jatu tempo", "due date", "duedate",
			// Bill types
			"listrik", "listrikn", "pln", "pulsa", "pulsaa", "token", "token listrik",
			"air", "air pdam", "pdam", "internett", "internete", "wifi", "wfii",
			"bpjs", "premi", "asuransi", "cicilan", "angsuran",
			// More variations
			"byr tagihan", "bayar bill", "bayar tgihan", "bayar tghn",
			"pembayaran", "pembyaran", "byar", "byrr",
		},
		"account": {
			// Original keywords
			"saldo", "bank", "e-wallet", "kartu debit", "kartu kredit", "akun", "rekening",
			// Typo/slang variations
			"saldoo", "salddo", "sldo", "sld", "balance", "balace",
			"bank", "bnk", "bngk", "banking",
			"ewallet", "ewalet", "ewallett", "dana", "gopay", "ovo", "shopeepay",
			"kartu", "kartoo", "krt", "debit", "dbt", "kredit", "krdit",
			// Action variations
			"topup", "top up", "tpup", "tupup", "isi saldo", "isi sldo",
			"tarik", "tarik uang", "withdraw", "wd", "tarik tunai",
			"deposit", "dposit", "transfer masuk", "tf masuk",
			// More variations
			"akun", "akunn", "akunnn", "account", "acc", "accnt",
			"rekening", "rekeningg", "rkg", "rek",
		},
		"savings": {
			// Original keywords
			"tabungan", "savings", "goal", "target", "menabung", "nabung", "save", "saved",
			// Typo/slang variations
			"tabungan", "tbungan", "tabngn", "tabngun", "tbn",
			"savings", "saving", "svng", "svngs",
			"goal", "gol", "gaol", "target", "trgt", "tgt",
			"menabung", "mnabung", "nabung", "nbing", "nbng", "tabung",
			"save", "sv", "sve", "saved", "svd",
			// More variations
			"target tabungan", "goal tabungan", "target nabung",
			"tabungan baru", "goal baru", "target baru", "mulai nabung",
			"nabung", "nbng", "nbing", "masi nabung", "lagi nabung",
		},
		"investment": {
			// Original keywords
			"crypto", "bitcoin", "ethereum", "invest", "portfolio", "investasi", "trading",
			"saham", "stock", "stocks",
			// Typo/slang variations
			"crypto", "kripto", "cryipt", "bitcoin", "btc", "btcc",
			"ethereum", "eth", "ethh",
			"invest", "investing", "invst", "investasi", "investsi", "nvst",
			"trading", "trde", "trding", "trad", "td",
			"saham", "sham", "shm", "stock", "stok", "stck",
			// Stock tickers
			"aapl", "googl", "msft", "tsla", "amzn", "fb", "meta",
			// More variations
			"portfolio", "portofolio", "portofolyo",
			"beli saham", "jual saham", "trading saham", "investasikan",
			"crypto trading", "trading crypto", "beli btc", "jual eth",
		},
		"debt": {
			// Original keywords
			"hutang", "debt", "pinjaman", "kredit", "cicilan", "loan",
			// Typo/slang variations
			"hutang", "utang", "hutng", "utng", "ht",
			"debt", "dbt", "dpt",
			"pinjaman", "pnjm", "pinjm", "pinjman", "pjmn",
			"kredit", "krdit", "krdt", "kreedit",
			"cicilan", "ccln", "cicln", "cicilannn",
			"loan", "ln", "loann", "pinjaman online",
			// More variations
			"bayarin cicilan", "bayar cicilan", "lunas", "lnas",
			"angsuran", "angsrn", "bayar angsuran",
		},
		"recurring": {
			// Original keywords
			"recurring", "berulang", "auto debit", "otomatis", "langganan", "subscription",
			// Typo/slang variations
			"recurring", "recurrng", "rcurring", "reccuring",
			"berulang", "brulang", "ulang", "ulang2", "rutin",
			"auto debit", "autodebit", "autodbt", "otodbt",
			"otomatis", "otmtis", "otomats", "auto",
			"langganan", "lngganan", "langanan", "lggnan",
			"subscription", "subscript", "subs", "member",
			// More variations
			"langganan baru", "renew langganan", "perpanjang langganan",
			"auto payment", "auto bayaran", "payment otomatis",
		},
		"spending": {
			// Original keywords
			"analisa", "analysis", "spending", "pola", "total", "cek", "lihat", "bulanan",
			// Typo/slang variations
			"analisa", "analisy", "anlisa", "anls", "analysis", "anlys",
			"pola", "polaa", "pattern", "tren", "trend",
			"total", "ttl", "jumlah", "jumlahny",
			"spending", "spnding", "spendingg", "pengeluaran",
			"bulanan", "blnan", "blnnan", "per bulan", "tiap bulan",
			// More variations
			"bulan ini", "bulan lalu", "minggu ini", "minggu lalu",
			"cek pengeluaran", "lihat pola", "analisa spending",
			"total pengeluaran", "summary", "ringkasan",
		},
		"health": {
			// Original keywords
			"health", "kesehatan", "keuangan", "summary", "ringkasan",
			// Typo/slang variations
			"health", "hlth", "helth", "helt",
			"kesehatan", "keshatan", "ksehtan", "shtn",
			"keuangan", "keu", "keuangn", "finansial", "financial",
			"summary", "smry", "summarry", "ringkasan", "ringkasn",
			// More variations
			"overview", "ovrvw", "dashboard", "dasbor",
			"kondisi keuangan", "status keuangan", "keadaan keuangan",
		},
	}

	if keywords, ok := keywordMap[intent]; ok {
		return keywords
	}
	return []string{}
}

