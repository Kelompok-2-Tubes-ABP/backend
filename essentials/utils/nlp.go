package utils

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseIndonesianAmount converts Indonesian number format to float
// Examples: "1 juta" -> 1000000, "500 ribu" -> 500000, "100rb" -> 100000, "1000000" -> 1000000
func ParseIndonesianAmount(message string) float64 {
	msg := strings.ToLower(message)

	// Pattern for "X juta" or "X million" - CHECK THIS FIRST
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:juta|jt|million|m)`)
	matches := re.FindStringSubmatch(msg)
	if matches != nil {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val * 1000000
		}
	}

	// Pattern for "X ribu" or "X rb" or "X thousand"
	re = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:ribu|rb|thousand|k)`)
	matches = re.FindStringSubmatch(msg)
	if matches != nil {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val * 1000
		}
	}

	// Pattern for plain numbers last (e.g., "1000000")
	re = regexp.MustCompile(`(\d+)`)
	matches = re.FindStringSubmatch(msg)
	if matches != nil {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}

	return 0
}

// ExtractSavingsGoalName extracts the savings goal name from the message
func ExtractSavingsGoalName(message string) string {
	msg := strings.ToLower(message)

	// Common patterns: look for savings goal names in the message
	patterns := []string{
		`tabungan\s+(.+?)(?:\s+\d+|$)`,
		`nabung\s+(.+?)(?:\s+\d+|$)`,
		`target\s+(.+?)(?:\s+\d+|$)`,
		`untuk\s+(.+?)(?:\s+\d+|$)`,
		`buat\s+(.+?)(?:\s+\d+|$)`,
		`liburan\s+ke\s+(\w+)`,
		`(?:ke|jepang|milan|liburan|mobil|bayar|hp|laptop)\s*(\w+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(msg)
		if matches != nil && len(matches) > 1 {
			name := strings.TrimSpace(matches[1])
			name = strings.TrimSuffix(name, "nya")
			name = strings.TrimSuffix(name, "p")
			if len(name) > 1 {
				return name
			}
		}
	}

	knownGoals := []string{"jepang", "milan", "liburan", "mobil", "hp", "laptop", "rumah"}
	for _, goal := range knownGoals {
		if strings.Contains(msg, goal) {
			return goal
		}
	}

	return ""
}

// ExtractExpenseCategory extracts the expense category from the message
func ExtractExpenseCategory(message string) string {
	msg := strings.ToLower(message)
	categories := []struct {
		category string
		keywords []string
	}{
		{"food", []string{"makan", "food", "lunch", "dinner", "breakfast"}},
		{"transport", []string{"ojek", "taxi", "grab", "gojek", "bensin", "bbm", "parkir", "tol", "transport", "angkot", "bus", "kereta"}},
		{"entertainment", []string{"nonton", "film", "movie", "bioskop", "konser", "game", "streaming", "netflix"}},
		{"bills", []string{"listrik", "air", "internet", "wifi", "pulsa", "token", "tagihan", "bill"}},
		{"health", []string{"obat", "dokter", "rumah sakit", "rs", "apotek", "medical"}},
		{"education", []string{"buku", "kursus", "sekolah", "kuliah", "les", "study", "pelajaran"}},
		{"shopping", []string{"belanja", "pakaian", "baju", "sepatu", "tas", "beli", "shopping", "buy"}}, // 'beli' put here as fallback
		{"other", []string{"other", "lain", "lainnya"}},
	}

	for _, c := range categories {
		for _, kw := range c.keywords {
			// Using word boundaries to avoid partial matches would be better, but simple Contains is used originally
			if strings.Contains(msg, kw) {
				return c.category
			}
		}
	}
	return ""
}

// ExtractTransactionDescription extracts description from transaction message
func ExtractTransactionDescription(message string) string {
	msg := strings.ToLower(message)
	patterns := []string{
		`untuk\s+(.+)`,
		`ke\s+(.+)`,
		`desc\s*:\s*(.+)`,
		`catatan\s*:\s*(.+)`,
		`note\s*:\s*(.+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(msg)
		if matches != nil && len(matches) > 1 {
			desc := matches[1]
			words := strings.Fields(desc)
			if len(words) > 5 {
				desc = strings.Join(words[:5], " ") + "..."
			}
			return desc
		}
	}
	return "Via Chatbot"
}

// ExtractBillNameFromMessage extracts bill name
func ExtractBillNameFromMessage(msg string) string {
	removables := []string{"tagihan", "bill", "ada", "tambah", "catat", "buat", "punya", "baru", "jatuh", "tempo", "dalam", "bulan", "minggu", "hari", "sebesar", "nominalny", "nominal", "rp", "juta", "ribu", "rb", "jt"}
	cleaned := msg
	for _, r := range removables {
		cleaned = strings.ReplaceAll(cleaned, r, "")
	}
	re := regexp.MustCompile(`\d+`)
	cleaned = re.ReplaceAllString(cleaned, "")
	words := strings.Fields(cleaned)
	if len(words) > 0 {
		return strings.Join(words, " ")
	}
	return ""
}

// ExtractInterestRate extracts interest rate percentage
func ExtractInterestRate(msg string) float64 {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:%|persen|percent)`)
	matches := re.FindStringSubmatch(msg)
	if len(matches) > 1 {
		val, _ := strconv.ParseFloat(matches[1], 64)
		return val
	}
	return 0
}

// ExtractTenor extracts month/year duration
func ExtractTenor(msg string) int {
	reYear := regexp.MustCompile(`(\d+)\s*(?:tahun|year|thn|yr)`)
	matchesYear := reYear.FindStringSubmatch(msg)
	if len(matchesYear) > 1 {
		val, _ := strconv.Atoi(matchesYear[1])
		return val * 12
	}
	reMonth := regexp.MustCompile(`(\d+)\s*(?:bulan|month|bln|mo|tenor)`)
	matchesMonth := reMonth.FindStringSubmatch(msg)
	if len(matchesMonth) > 1 {
		val, _ := strconv.Atoi(matchesMonth[1])
		return val
	}
	return 0
}

// ExtractDebtNameFromMessage extracts debt asset name
func ExtractDebtNameFromMessage(msg string) string {
	removables := []string{
		"bayar", "bayarin", "cicil", "cicilan", "pake", "pakai", "pakek", "menggunakan",
		"jumlah", "untuk", "sebesar", "rp", "juta", "ribu", "rb", "jt", "nominalnya",
		"bayarkan", "ada", "baru", "tambah", "catat", "buat", "punya", "bunga",
		"persen", "percent", "tenor", "bulan", "tahun", "aku", "mau", "saya", "ingin",
		"hutang", "hutangku", "tagihan", "dong", "nih", "ya", "sip", "oke", "tolong",
	}

	cleaned := msg
	for _, r := range removables {
		re := regexp.MustCompile(`(?i)\b` + r + `\b`)
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	rePct := regexp.MustCompile(`\d+(?:\.\d+)?\s*%`)
	cleaned = rePct.ReplaceAllString(cleaned, "")

	words := strings.Fields(cleaned)
	if len(words) > 0 {
		limit := 3
		if len(words) < limit {
			limit = len(words)
		}
		return strings.Join(words[:limit], " ")
	}
	return ""
}

// ExtractAccountNameFromMessage extracts account name for payment
func ExtractAccountNameFromMessage(msg string) string {
	keywords := []string{"pake", "pakai", "pakek", "dari", "rekening", "akun"}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			parts := strings.Split(msg, kw)
			if len(parts) >= 2 {
				words := strings.Fields(parts[1])
				if len(words) > 0 {
					return words[0]
				}
			}
		}
	}
	return ""
}

// ExtractSymbolFromMessage extracts stock/crypto symbol
func ExtractSymbolFromMessage(msg string) string {
	removables := []string{"harga", "berapa", "price", "nilai", "saat", "ini", "sekarang", "cek", "dong", "saham", "crypto"}
	cleaned := msg
	for _, r := range removables {
		cleaned = strings.ReplaceAll(cleaned, r, "")
	}
	words := strings.Fields(cleaned)
	if len(words) > 0 {
		return words[len(words)-1]
	}
	return ""
}

// ExtractAmountString extracts raw amount string including formatting
func ExtractAmountString(msg string) string {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?\s*(?:juta|jt|million|m|ribu|rb|thousand|k))`)
	if match := re.FindString(msg); match != "" {
		return match
	}
	reNumeric := regexp.MustCompile(`(\d{4,})`)
	return reNumeric.FindString(msg)
}

// FormatNumber formats float to string gracefully
func FormatNumber(val float64) string {
	if val >= 1000 {
		return strconv.FormatFloat(val, 'f', 0, 64)
	}
	return strconv.FormatFloat(val, 'f', 2, 64)
}

// ContainsAny checks if any keyword exists in the string (case-insensitive)
func ContainsAny(s string, keywords []string) bool {
	s = strings.ToLower(s)
	for _, kw := range keywords {
		if strings.Contains(s, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// DetectLanguage detects if message is Indonesian or English
func DetectLanguage(message string) string {
	msg := strings.ToLower(message)
	indonesianKeywords := []string{"yang", "dan", "untuk", "dari", "dengan", "tidak", "aku", "kamu", "mau", "nya", "rp", "juta", "ribu", "bu", "buat", "lagi", "bisa", "ada", "sudah", "belum", "akan", "bayar", "pake", "gimana", " gimana", "berapa"}

	matchCount := 0
	for _, kw := range indonesianKeywords {
		if strings.Contains(msg, kw) {
			matchCount++
		}
	}

	// If more than 2 Indonesian keywords found, assume Indonesian
	if matchCount >= 2 {
		return "id"
	}
	return "en"
}

// ParseDateRange extracts date range from natural language message
// Returns startDate, endDate, periodName, and error
func ParseDateRange(message string) (time.Time, time.Time, string, error) {
	msg := strings.ToLower(message)
	now := time.Now()

	// Month name mapping (Indonesian + English abbreviations)
	monthNames := map[string]time.Month{
		"januari":   time.January,
		"februari":  time.February,
		"maret":     time.March,
		"mei":       time.May,
		"juni":      time.June,
		"juli":      time.July,
		"agustus":   time.August,
		"september": time.September,
		"oktober":   time.October,
		"november":  time.November,
		"desember":  time.December,
		"jan":       time.January,
		"feb":       time.February,
		"mar":       time.March,
		"apr":       time.April,
		"jun":       time.June,
		"jul":       time.July,
		"aug":       time.August,
		"sep":       time.September,
		"oct":       time.October,
		"nov":       time.November,
		"dec":       time.December,
	}

	// Check for "bulan ini" (this month)
	if strings.Contains(msg, "bulan ini") {
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0).Add(-time.Second)
		return start, end, "bulan ini", nil
	}

	// Check for "bulan lalu" (last month)
	if strings.Contains(msg, "bulan lalu") {
		start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0).Add(-time.Second)
		return start, end, "bulan lalu", nil
	}

	// Check for "kemarin" (yesterday)
	if strings.Contains(msg, "kemarin") || strings.Contains(msg, "yesterday") {
		start := now.AddDate(0, 0, -1)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end := start.AddDate(0, 0, 1).Add(-time.Second)
		return start, end, "kemarin", nil
	}

	// Check for "hari ini" (today)
	if strings.Contains(msg, "hari ini") || strings.Contains(msg, "today") {
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 0, 1).Add(-time.Second)
		return start, end, "hari ini", nil
	}

	// Check for "minggu ini" (this week)
	if strings.Contains(msg, "minggu ini") || strings.Contains(msg, "week ini") {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := now.AddDate(0, 0, -(weekday - 1))
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end := start.AddDate(0, 0, 7).Add(-time.Second)
		return start, end, "minggu ini", nil
	}

	// Check for "minggu lalu" (last week)
	if strings.Contains(msg, "minggu lalu") || strings.Contains(msg, "week lalu") {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := now.AddDate(0, 0, -(weekday - 1 + 7))
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end := start.AddDate(0, 0, 7).Add(-time.Second)
		return start, end, "minggu lalu", nil
	}

	// Check for "N bulan lalu" pattern
	re := regexp.MustCompile(`(\d+)\s*bulan\s+lalu`)
	if matches := re.FindStringSubmatch(msg); len(matches) > 1 {
		if months, err := strconv.Atoi(matches[1]); err == nil {
			start := time.Date(now.Year(), now.Month()-time.Month(months), 1, 0, 0, 0, 0, now.Location())
			end := start.AddDate(0, 1, 0).Add(-time.Second)
			return start, end, matches[1] + " bulan lalu", nil
		}
	}

	// Check for "N minggu lalu" pattern
	re = regexp.MustCompile(`(\d+)\s*minggu\s+lalu`)
	if matches := re.FindStringSubmatch(msg); len(matches) > 1 {
		if weeks, err := strconv.Atoi(matches[1]); err == nil {
			start := now.AddDate(0, 0, -(weeks*7))
			start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
			end := now
			return start, end, matches[1] + " minggu lalu", nil
		}
	}

	// Check for "N tahun lalu" pattern
	re = regexp.MustCompile(`(\d+)\s*tahun\s+lalu`)
	if matches := re.FindStringSubmatch(msg); len(matches) > 1 {
		if years, err := strconv.Atoi(matches[1]); err == nil {
			start := time.Date(now.Year()-years, now.Month(), 1, 0, 0, 0, 0, now.Location())
			end := start.AddDate(0, 1, 0).Add(-time.Second)
			return start, end, matches[1] + " tahun lalu", nil
		}
	}

	// Check for month name (e.g., "januari", "februari")
	for name, month := range monthNames {
		if strings.Contains(msg, name) {
			start := time.Date(now.Year(), month, 1, 0, 0, 0, 0, now.Location())
			end := start.AddDate(0, 1, 0).Add(-time.Second)
			return start, end, name, nil
		}
	}

	// Default to current month
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0).Add(-time.Second)
	return start, end, "bulan ini", nil
}
