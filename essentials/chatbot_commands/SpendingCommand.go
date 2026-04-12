package chatbot_commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/services"
)

type SpendingCommand struct {
	txService *services.TransactionService
}

func NewSpendingCommand(txService *services.TransactionService) *SpendingCommand {
	return &SpendingCommand{
		txService: txService,
	}
}

func (c *SpendingCommand) Handle(userID string, message string) string {
	msgLower := strings.ToLower(message)
	return c.handleSpendingAnalysis(userID, msgLower, message)
}

func (c *SpendingCommand) handleSpendingAnalysis(userID, msgLower, message string) string {
	now := time.Now()
	var startDate, endDate time.Time
	var periodName string

	numberMonthAgo := regexp.MustCompile(`(\d+)\s*(bulan|month)\s*(lalu|yang lalu|kemarin|ago)`)
	numberYearAgo := regexp.MustCompile(`(\d+)\s*(tahun|year)\s*(lalu|yang lalu|kemarin|ago)`)
	numberWeekAgo := regexp.MustCompile(`(\d+)\s*(minggu|week)\s*(lalu|yang lalu|kemarin|ago)`)

	if match := numberMonthAgo.FindStringSubmatch(msgLower); match != nil {
		if num, err := strconv.Atoi(match[1]); err == nil {
			targetMonth := now.AddDate(0, -num, 0)
			startDate = time.Date(targetMonth.Year(), targetMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
			endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
			periodName = fmt.Sprintf("%d Bulan Lalu", num)
		}
	} else if match := numberYearAgo.FindStringSubmatch(msgLower); match != nil {
		if num, err := strconv.Atoi(match[1]); err == nil {
			targetYear := now.Year() - num
			startDate = time.Date(targetYear, 1, 1, 0, 0, 0, 0, time.UTC)
			endDate = time.Date(targetYear, 12, 31, 23, 59, 59, 0, time.UTC)
			periodName = fmt.Sprintf("%d Tahun Lalu", num)
		}
	} else if match := numberWeekAgo.FindStringSubmatch(msgLower); match != nil {
		if num, err := strconv.Atoi(match[1]); err == nil {
			targetWeek := now.AddDate(0, 0, -num*7)
			weekday := int(targetWeek.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			startDate = targetWeek.AddDate(0, 0, -(weekday - 1))
			startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
			endDate = startDate.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			periodName = fmt.Sprintf("%d Minggu Lalu", num)
		}
	} else {
		monthNames := map[string]time.Month{
			"januari":   time.January,
			"februari":  time.February,
			"maret":     time.March,
			"april":     time.April,
			"mei":       time.May,
			"juni":      time.June,
			"juli":      time.July,
			"agustus":   time.August,
			"september": time.September,
			"oktober":   time.October,
			"november":  time.November,
			"desember":  time.December,
		}

		for monthName, monthVal := range monthNames {
			if strings.Contains(msgLower, "bulan "+monthName) || strings.Contains(msgLower, "bulan "+strconv.Itoa(int(monthVal))) {
				startDate = time.Date(now.Year(), monthVal, 1, 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
				periodName = startDate.Format("January 2006")
				break
			}
		}

		if startDate.IsZero() {
			switch {
			case containsAny(msgLower, []string{"bulan lalu", "last month"}):
				lastMonth := now.AddDate(0, -1, 0)
				startDate = time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
				periodName = lastMonth.Format("January 2006")
			case containsAny(msgLower, []string{"bulan ini", "this month", "bulan sekarang"}):
				startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
				periodName = now.Format("January 2006")
			case containsAny(msgLower, []string{"minggu ini", "this week"}):
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				startDate = now.AddDate(0, 0, -(weekday - 1))
				startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				periodName = "Minggu Ini"
			case containsAny(msgLower, []string{"minggu lalu", "last week"}):
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				lastWeekStart := now.AddDate(0, 0, -(weekday-1)-7)
				startDate = time.Date(lastWeekStart.Year(), lastWeekStart.Month(), lastWeekStart.Day(), 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				periodName = "Minggu Lalu"
			case containsAny(msgLower, []string{"tahun ini", "this year"}):
				startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
				endDate = time.Date(now.Year(), 12, 31, 23, 59, 59, 0, time.UTC)
				periodName = fmt.Sprintf("Tahun %d", now.Year())
			case containsAny(msgLower, []string{"tahun lalu", "last year"}):
				lastYear := now.Year() - 1
				startDate = time.Date(lastYear, 1, 1, 0, 0, 0, 0, time.UTC)
				endDate = time.Date(lastYear, 12, 31, 23, 59, 59, 0, time.UTC)
				periodName = fmt.Sprintf("Tahun %d", lastYear)
			case containsAny(msgLower, []string{"kemarin", "semalam", "yesterday"}):
				yesterday := now.AddDate(0, 0, -1)
				startDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
				endDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, time.UTC)
				periodName = "Kemarin"
			case containsAny(msgLower, []string{"hari ini", "today"}):
				startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
				endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)
				periodName = "Hari Ini"
			default:
				startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
				periodName = now.Format("January 2006")
			}
		}
	}

	filter := models.FilterTransaction{
		FromDate: startDate.Format("2006-01-02"),
		ToDate:   endDate.Format("2006-01-02"),
	}

	report, err := c.txService.GetReport(userID, filter)
	if err != nil {
		return "❌ Gagal mengambil data pengeluaran: " + err.Error()
	}

	outcome := report["outcome"]
	income := report["income"]
	net := report["net"]

	response := fmt.Sprintf("📊 Laporan Pengeluaran %s\n\n", periodName)
	response += fmt.Sprintf("💰 Pemasukan: Rp %.0f\n", income)
	response += fmt.Sprintf("💸 Pengeluaran: Rp %.0f\n", outcome)
	response += fmt.Sprintf("📈 Sisa: Rp %.0f\n\n", net)

	if outcome > 0 && income > 0 {
		savingsRate := ((income - outcome) / income) * 100
		if savingsRate > 20 {
			response += "✅ Kondisi keuangan baik! Tabungan > 20%%"
		} else if savingsRate > 0 {
			response += "⚠️ Coba lebih hemat! Tabungan < 20%%"
		} else {
			response += "❌ Pengeluaran melebihi pemasukan!"
		}
	} else if outcome == 0 {
		response += "💡 Belum ada pengeluaran"
	}

	return response
}
