package handler

import (
	"encoding/csv"
	"financeapi/essentials/services"
	"fmt"

	"github.com/gin-gonic/gin"
)

func ExportCSVHandler(t *services.TransactionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"error": "Not Authorized"})
			return
		}

		transactions, err := t.ShowTransaction(userID.(string))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=transactions.csv")

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		writer.Write([]string{"Date", "Category", "Amount", "Description"})

		for _, tr := range transactions {
			formattedDate := tr.Date.Format("2006-01-02")
			record := []string{
				formattedDate,
				tr.Category,
				fmt.Sprintf("%.2f", tr.Amount),
				tr.Description,
			}
			writer.Write(record)
		}
	}
}
