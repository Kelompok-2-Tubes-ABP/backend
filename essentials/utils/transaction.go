package utils

import "strings"

// IsIncome check if a category is an income category
func IsIncome(category string) bool {
	categoryLower := strings.ToLower(category)
	incomeCategories := map[string]bool{
		"income":     true,
		"gaji":       true,
		"salary":     true,
		"pendapatan": true,
		"revenue":    true,
	}

	return incomeCategories[categoryLower]
}

// IsOutcome check if a category is an outcome category
func IsOutcome(category string) bool {
	// Everything that is not income is considered outcome in this system
	return !IsIncome(category)
}

func IsValidMonthFormat(month string) bool {
	if len(month) != 7 {
		return false
	}
	if month[4] != '-' {
		return false
	}
	for i, char := range month {
		if i == 4 {
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
