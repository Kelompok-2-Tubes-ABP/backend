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
