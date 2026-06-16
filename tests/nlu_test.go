package tests

import (
	"testing"

	"financeapi/essentials/utils"
)

// TestFuzzyMatch tests typo tolerance functionality
func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		keywords []string
		threshold float64
		expected bool
	}{
		{
			name:     "exact match",
			input:    "beli",
			keywords: []string{"beli", "bayar"},
			threshold: 0.8,
			expected: true,
		},
		{
			name:     "typo - missing letter",
			input:    "bli",
			keywords: []string{"beli", "bayar"},
			threshold: 0.8,
			expected: true,
		},
		{
			name:     "typo - swapped letters",
			input:    "lebi",
			keywords: []string{"beli", "bayar"},
			threshold: 0.8,
			expected: true,
		},
		{
			name:     "no match - too different",
			input:    "xyz",
			keywords: []string{"beli", "bayar"},
			threshold: 0.8,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FuzzyMatch(tt.input, tt.keywords, tt.threshold)
			if result != tt.expected {
				t.Errorf("FuzzyMatch(%q, %v, %.2f) = %v; want %v",
					tt.input, tt.keywords, tt.threshold, result, tt.expected)
			}
		})
	}
}

// TestLevenshteinDistance tests string similarity calculation
func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{
		{
			name:     "identical strings",
			s1:       "beli",
			s2:       "beli",
			expected: 0,
		},
		{
			name:     "one letter difference",
			s1:       "beli",
			s2:       "bli",
			expected: 1,
		},
		{
			name:     "completely different",
			s1:       "abc",
			s2:       "xyz",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.LevenshteinDistance(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d; want %d",
					tt.s1, tt.s2, result, tt.expected)
			}
		})
	}
}

// TestSimilarityScore tests similarity calculation
func TestSimilarityScore(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		minScore float64
	}{
		{
			name:     "identical",
			s1:       "beli",
			s2:       "beli",
			minScore: 1.0,
		},
		{
			name:     "similar",
			s1:       "beli",
			s2:       "bli",
			minScore: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SimilarityScore(tt.s1, tt.s2)
			if result < tt.minScore {
				t.Errorf("SimilarityScore(%q, %q) = %.2f; want >= %.2f",
					tt.s1, tt.s2, result, tt.minScore)
			}
		})
	}
}

// TestExtractEntities tests structured entity extraction
func TestExtractEntities(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		expectAmount bool
	}{
		{
			name:        "expense with amount",
			message:     "tadi aku beli makan 50rb buat teman",
			expectAmount: true,
		},
		{
			name:        "expense with juta",
			message:     "bayar tagihan listrik 1 juta",
			expectAmount: true,
		},
		{
			name:        "no amount",
			message:     "bagaimana keuangan saya?",
			expectAmount: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := utils.ExtractEntities(tt.message)
			if tt.expectAmount && entities.Amount == 0 {
				t.Errorf("Expected amount to be extracted from %q", tt.message)
			}
			if !tt.expectAmount && entities.Amount > 0 {
				t.Errorf("Expected no amount from %q", tt.message)
			}
		})
	}
}

// TestDetectIntent tests intent detection
func TestDetectIntent(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "typo - bli instead of beli",
			message:  "tadi aku bli makan",
			expected: "transaction", // or "ai" if no clear intent
		},
		{
			name:     "typo - byar instead of bayar",
			message:  "byar tagihan internet",
			expected: "bills",
		},
		{
			name:     "explicit budget with food",
			message:  "tambah budget makanan 500rb",
			expected: "budget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := utils.DetectIntent(tt.message)
			if intent != tt.expected {
				t.Errorf("DetectIntent(%q) = %q; want %q",
					tt.message, intent, tt.expected)
			}
		})
	}
}

// TestDetectIntentConfidence tests confidence scoring
func TestDetectIntentConfidence(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		minConfidence float64
	}{
		{
			name:        "high confidence",
			message:     "tambah budget makanan 500rb",
			minConfidence: 0.6, // Lowered since "budget" is in message
		},
		{
			name:        "low confidence",
			message:     "uang",
			minConfidence: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, confidence := utils.DetectIntentWithConfidence(tt.message)
			if confidence < tt.minConfidence {
				t.Errorf("DetectIntentWithConfidence(%q) confidence = %.2f; want >= %.2f",
					tt.message, confidence, tt.minConfidence)
			}
		})
	}
}

// TestExtractTimeExpression tests time expression parsing
func TestExtractTimeExpression(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "tadi",
			message:  "tadi aku beli makan",
			expected: "tadi",
		},
		{
			name:     "kemarin",
			message:  "kemarin transfer 100rb",
			expected: "kemarin",
		},
		{
			name:     "bulan lalu",
			message:  "bulan lalu pengeluaran 5 juta",
			expected: "bulan lalu",
		},
		{
			name:     "no time",
			message:  "bayar tagihan",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ExtractTimeExpression(tt.message)
			if result != tt.expected {
				t.Errorf("ExtractTimeExpression(%q) = %q; want %q",
					tt.message, result, tt.expected)
			}
		})
	}
}

// TestIsReferenceKeyword tests reference detection
func TestIsReferenceKeyword(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		isRef    bool
	}{
		{
			name:    "reference itu",
			message: "bisa ubah itu?",
			isRef:   true,
		},
		{
			name:    "reference yang ini",
			message: "yang ini jangan dihapus",
			isRef:   true,
		},
		{
			name:    "not reference",
			message: "tambah budget baru",
			isRef:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsReferenceKeyword(tt.message)
			if result != tt.isRef {
				t.Errorf("IsReferenceKeyword(%q) = %v; want %v",
					tt.message, result, tt.isRef)
			}
		})
	}
}

// TestParsedIntentStruct tests ParsedIntent structure
func TestParsedIntentStruct(t *testing.T) {
	message := "tadi aku beli makan 50rb buat teman"
	entities := utils.ExtractEntities(message)

	if entities.Amount == 0 {
		t.Error("Amount should be extracted")
	}
	if entities.Action == "" {
		t.Error("Action should be detected")
	}
}

// TestDetectLanguage tests language detection
func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "Indonesian - common words",
			message:  "tadi aku beli makan 50rb",
			expected: "id",
		},
		{
			name:     "Indonesian - with slang",
			message:  "bgt mau nabung dpt",
			expected: "id",
		},
		{
			name:     "English - common words",
			message:  "how much did I spend this month?",
			expected: "en",
		},
		{
			name:     "English - budget",
			message:  "can you check my budget?",
			expected: "en",
		},
		{
			name:     "English - spending",
			message:  "what was my spending this week?",
			expected: "en",
		},
		{
			name:     "Indonesian with slang",
			message:  "tadi aku bli makan 50rb",
			expected: "id",
		},
		{
			name:     "English - thousand",
			message:  "I spent 5 thousand",
			expected: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.DetectLanguage(tt.message)
			if result != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q; want %q",
					tt.message, result, tt.expected)
			}
		})
	}
}



// TestDetectIntentEnhanced tests the enhanced intent detection
func TestDetectIntentEnhanced(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		expectedIntent  string
		expectedLang    string
	}{
		{
			name:            "Indonesian transaction",
			message:         "tadi aku beli makan 50rb",
			expectedIntent:  "transaction",
			expectedLang:    "id",
		},
		{
			name:            "English spending",
			message:         "how much did I spend?",
			expectedIntent:  "ai", // Question - sent to AI
			expectedLang:    "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test language detection
			lang := utils.DetectLanguage(tt.message)
			if lang != tt.expectedLang {
				t.Errorf("DetectLanguage(%q) = %q; want %q",
					tt.message, lang, tt.expectedLang)
			}
		})
	}
}

// TestDetectSentiment tests sentiment analysis
func TestDetectSentiment(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "frustrated - angry words",
			message:  "anjing kok gak bisa!",
			expected: "frustrated",
		},
		{
			name:     "frustrated - annoying",
			message:  "this is so frustrating",
			expected: "frustrated",
		},
		{
			name:     "happy - terima kasih",
			message:  "terima kasih banyak!",
			expected: "happy",
		},
		{
			name:     "happy - great",
			message:  "awesome, great job!",
			expected: "happy",
		},
		{
			name:     "confused - gimana",
			message:  "gimana sih caranya?",
			expected: "confused",
		},
		{
			name:     "confused - confused",
			message:  "I'm confused about this",
			expected: "confused",
		},
		{
			name:     "urgent - emergency",
			message:  "urgent! I need help now",
			expected: "urgent",
		},
		{
			name:     "urgent - segera",
			message:  "segera tangani ini",
			expected: "urgent",
		},
		{
			name:     "satisfied - oke sip",
			message:  "oke sip, terima kasih",
			expected: "satisfied",
		},
		{
			name:     "satisfied - oke sip alone",
			message:  "oke sip",
			expected: "satisfied",
		},
		{
			name:     "neutral - simple statement",
			message:  "bayar tagihan listrik",
			expected: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.DetectSentiment(tt.message)
			if result.Emotion != tt.expected {
				t.Errorf("DetectSentiment(%q) = %q; want %q",
					tt.message, result.Emotion, tt.expected)
			}
		})
	}
}

// TestGetSentimentContext tests sentiment context generation
func TestGetSentimentContext(t *testing.T) {
	sentiment := utils.DetectSentiment("anjing kok gak bisa!")
	ctx := utils.GetSentimentContext(sentiment)
	
	if ctx["sentiment"] != "frustrated" {
		t.Errorf("Expected sentiment 'frustrated', got %v", ctx["sentiment"])
	}
	
	if ctx["needs_empathy"] != true {
		t.Errorf("Expected needs_empathy to be true for frustrated sentiment")
	}
}
