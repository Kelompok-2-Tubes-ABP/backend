package utils

import (
	"regexp"
	"strconv"
	"strings"
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
	categoryMap := map[string]string{
		"makan": "food", "food": "food", "lunch": "food", "dinner": "food", "breakfast": "food",
		"ojek": "transport", "taxi": "transport", "grab": "transport", "gojek": "transport", "bensin": "transport", "bbm": "transport", "parkir": "transport", "tol": "transport", "transport": "transport", "angkot": "transport", "bus": "transport", "kereta": "transport",
		"belanja": "shopping", "beli": "shopping", "shopping": "shopping", "buy": "shopping", "pakaian": "shopping", "baju": "shopping", "sepatu": "shopping", "tas": "shopping",
		"nonton": "entertainment", "film": "entertainment", "movie": "entertainment", "bioskop": "entertainment", "konser": "entertainment", "game": "entertainment", "streaming": "entertainment", "netflix": "entertainment",
		"listrik": "bills", "air": "bills", "internet": "bills", "wifi": "bills", "pulsa": "bills", "token": "bills", "tagihan": "bills", "bill": "bills",
		"obat": "health", "dokter": "health", "rumah sakit": "health", "rs": "health", "apotek": "health", "medical": "health",
		"buku": "education", "kursus": "education", "sekolah": "education", "kuliah": "education", "les": "education", "study": "education", "pelajaran": "education",
		"other": "other", "lain": "other", "lainnya": "other",
	}

	for keyword, category := range categoryMap {
		if strings.Contains(msg, keyword) {
			return category
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
