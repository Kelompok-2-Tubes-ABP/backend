package handler

import (
	model "financeapi/essentials/models"
	service "financeapi/essentials/services"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateTransactionHandler(t *service.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tr model.Transaction
		timee := time.Now()
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, "Not Authorized!!!")
			return
		}
		if err := c.ShouldBindJSON(&tr); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		tr.User_id = userID.(string)
		tr.Date = timee

		// Normalize month to YYYY-MM format
		// Get year from Date
		year := timee.Year()

		if tr.Month != "" {
			// Convert month name to number, then format as YYYY-MM
			monthNum := monthNameToNumber(strings.ToLower(tr.Month))
			if monthNum > 0 && monthNum <= 12 {
				tr.Month = time.Date(year, time.Month(monthNum), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
			} else {
				// If month already in YYYY-MM format, keep it
				tr.Month = time.Date(year, time.Month(1), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
			}
		} else {
			// Default to current month
			tr.Month = timee.Format("2006-01")
		}

		transaction, err := t.CreateTransaction(tr)

		if err != nil {
			c.JSON(401, gin.H{"Error": "Transaction Not Found!!!"})
			return
		}
		c.JSON(200, gin.H{"Transaction added succesfully": transaction})
	}
}

func monthNameToNumber(month string) int {
	months := map[string]int{
		"january":   1, "jan": 1,
		"february":  2, "feb": 2,
		"march":     3, "mar": 3,
		"april":     4, "apr": 4,
		"may":       5,
		"june":      6, "jun": 6,
		"july":      7, "jul": 7,
		"august":    8, "aug": 8,
		"september": 9, "sep": 9, "sept": 9,
		"october":   10, "oct": 10,
		"november":  11, "nov": 11,
		"december":  12, "dec": 12,
	}
	return months[month]
}
