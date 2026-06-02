package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"financeapi/essentials/config"
	"financeapi/essentials/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Println("🚀 Starting FinanceAPI Database Seeder...")

	// Connect to MongoDB
	client := config.ConnectDB()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer client.Disconnect(ctx)

	db := client.Database("mydb")

	fmt.Println("🧹 Clearing existing test data...")

	collections := []string{
		"users", "admins", "accounts", "Transaction",
		"monthly_budget", "category_budget", "investments",
		"debts", "savings_goals", "bill_reminders",
		"recurring_transactions", "notifications",
		"system_alerts", "audit_logs", "active_tokens",
	}

	for _, coll := range collections {
		if _, err := db.Collection(coll).DeleteMany(context.TODO(), bson.M{}); err != nil {
			fmt.Printf("⚠️  Warning: Could not clear %s: %v\n", coll, err)
		}
	}

	// Seed Admin Account
	seedAdmin(db)

	// Seed Test Users
	userIDs := seedUsers(db)

	// Seed Accounts for Users
	seedAccounts(db, userIDs)

	// Seed Transactions
	seedTransactions(db, userIDs)

	// Seed Budgets
	seedBudgets(db, userIDs)

	// Seed Investments
	seedInvestments(db, userIDs)

	// Seed Debts
	seedDebts(db, userIDs)

	// Seed Savings Goals
	seedSavingsGoals(db, userIDs)

	// Seed Bill Reminders
	seedBillReminders(db, userIDs)

	// Seed Recurring Transactions
	seedRecurringTransactions(db, userIDs)

	// Seed Notifications
	seedNotifications(db, userIDs)

	// Seed System Alerts
	seedSystemAlerts(db)

	fmt.Println("✅ Database seeding completed successfully!")
	fmt.Println("\n📋 Test Credentials:")
	fmt.Println("─────────────────────────────────────")
	fmt.Println("🔐 Admin Login:")
	fmt.Println("   Email:    admin@financeapi.com")
	fmt.Println("   Password: Admin123!")
	fmt.Println("─────────────────────────────────────")
	fmt.Println("👤 Test Users (any can login):")
	fmt.Println("   Email:    john@example.com  | Password: User123!")
	fmt.Println("   Email:    jane@example.com   | Password: User123!")
	fmt.Println("   Email:    bob@example.com    | Password: User123!")
	fmt.Println("─────────────────────────────────────")
}

func seedAdmin(db *mongo.Database) {
	fmt.Println("👤 Seeding admin account...")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Admin123!"), bcrypt.DefaultCost)

	admin := models.Admin{
		ID:        primitive.NewObjectID(),
		Username:  "admin",
		Email:     "admin@financeapi.com",
		Password:  string(hashedPassword),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := db.Collection("admins").InsertOne(context.TODO(), admin)
	if err != nil {
		log.Printf("⚠️  Admin seeding error (may already exist): %v", err)
	} else {
		fmt.Println("   ✅ Admin account created")
	}
}

func seedUsers(db *mongo.Database) []primitive.ObjectID {
	fmt.Println("👥 Seeding test users...")

	testUsers := []struct {
		username string
		email    string
		password string
	}{
		{"johndoe", "john@example.com", "User123!"},
		{"janedoe", "jane@example.com", "User123!"},
		{"bobsmith", "bob@example.com", "User123!"},
		{"alicewong", "alice@example.com", "User123!"},
		{"charlie", "charlie@example.com", "User123!"},
	}

	var userIDs []primitive.ObjectID

	for _, u := range testUsers {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)

		user := models.User{
			ID:                primitive.NewObjectID(),
			Username:          u.username,
			Email:             u.email,
			Password:          string(hashedPassword),
			Role:              "user",
			IsEmailVerified:   true,
			EmailVerifiedAt:   &[]time.Time{time.Now().AddDate(0, -1, 0)}[0],
			IsActive:          true,
			CreatedAt:         time.Now().AddDate(0, 0, -30), // Created 30 days ago
			UpdatedAt:         time.Now(),
			PasswordChangedAt: time.Now(),
		}

		_, err := db.Collection("users").InsertOne(context.TODO(), user)
		if err != nil {
			log.Printf("⚠️  User %s seeding error: %v", u.username, err)
			continue
		}

		userIDs = append(userIDs, user.ID)
		fmt.Printf("   ✅ User: %s (%s)\n", u.username, u.email)
	}

	return userIDs
}

func seedAccounts(db *mongo.Database, userIDs []primitive.ObjectID) {
	fmt.Println("🏦 Seeding accounts...")

	accountTypes := []struct {
		name        string
		accType     models.AccountType
		institution string
		balance     float64
	}{
		{"BCA Savings", models.AccountTypeBank, "BCA", 15000000},
		{"Mandiri Savings", models.AccountTypeBank, "Mandiri", 8500000},
		{"GoPay Wallet", models.AccountTypeWallet, "Gojek", 2500000},
		{"OVO Cash", models.AccountTypeWallet, "Grab", 500000},
		{"BNI Credit Card", models.AccountTypeCredit, "BNI", 2500000},
	}

	categories := []string{"food", "transport", "entertainment", "shopping", "bills", "salary", "investment"}
	_ = categories // Used for transaction templates

	for i, userID := range userIDs {
		for _, acc := range accountTypes {
			account := models.Account{
				ID:              primitive.NewObjectID(),
				UserID:          userID,
				Name:            acc.name,
				Type:            acc.accType,
				Institution:     acc.institution,
				AccountNumber:   fmt.Sprintf("%d%08d", i+1, int(acc.balance)),
				Currency:        "IDR",
				CurrentBalance:  acc.balance + float64(i*1000000),
				InitialBalance:  acc.balance,
				AvailableCredit: 10000000,
				InterestRate:    0.5,
				IsActive:        true,
				IsDefault:       acc.name == "BCA Savings",
				LastSynced:      time.Now(),
				Notes:           fmt.Sprintf("Account for user index %d", i),
				Icon:            "account_balance",
				Color:           "#4CAF50",
				CreatedAt:       time.Now().AddDate(0, 0, -20),
				UpdatedAt:       time.Now(),
			}

			db.Collection("accounts").InsertOne(context.TODO(), account)
		}

		// Create investment account
		investmentAccount := models.Account{
			ID:              primitive.NewObjectID(),
			UserID:          userID,
			Name:            "Investment Account",
			Type:            models.AccountTypeInvestment,
			Institution:     "Self-Managed",
			Currency:        "IDR",
			CurrentBalance:  5000000 + float64(i*500000),
			InitialBalance:  5000000,
			IsActive:        true,
			LastSynced:      time.Now(),
			Icon:            "trending_up",
			Color:           "#2196F3",
			CreatedAt:       time.Now().AddDate(0, 0, -15),
			UpdatedAt:       time.Now(),
		}
		db.Collection("accounts").InsertOne(context.TODO(), investmentAccount)

		fmt.Printf("   ✅ User index %d: %d accounts\n", i, len(accountTypes)+1)
	}
}

func seedTransactions(db *mongo.Database, userIDs []primitive.ObjectID) {
	fmt.Println("💸 Seeding transactions...")

	transactionTemplates := []struct {
		category    string
		amount      float64
		description string
		isExpense   bool
	}{
		// Expenses
		{"food", 45000, "Lunch at Nasi Padang", true},
		{"food", 120000, "Dinner at Restaurant", true},
		{"transport", 25000, "GoRide to Office", true},
		{"transport", 350000, "Monthly Gas", true},
		{"shopping", 500000, "New Shoes", true},
		{"entertainment", 150000, "Netflix Subscription", true},
		{"bills", 150000, "Internet Bill", true},
		{"bills", 200000, "Phone Bill", true},
		{"health", 100000, "Pharmacy", true},
		{"education", 300000, "Online Course", true},
		// Income
		{"salary", 15000000, "Monthly Salary", false},
		{"freelance", 2500000, "Freelance Project", false},
		{"investment", 500000, "Stock Dividend", false},
		{"gift", 1000000, "Bonus from Parent", false},
	}

	totalTx := 0
	for i, userID := range userIDs {
		numTransactions := 30 + (i * 10) // Each user has different number of transactions

		for j := 0; j < numTransactions; j++ {
			txTemplate := transactionTemplates[j%len(transactionTemplates)]
			amount := txTemplate.amount

			// Add some variance to amounts
			variance := float64(j%5) * 0.1 // 0-40% variance
			if txTemplate.isExpense {
				amount = amount * (1 + variance)
			}

			date := time.Now().AddDate(0, 0, -j-i*5)

			txType := "outcome"
			if !txTemplate.isExpense {
				txType = "income"
			}

			tx := models.Transaction{
				ID:          primitive.NewObjectID(),
				User_id:     userID.Hex(),
				Amount:      amount,
				Category:    txTemplate.category,
				Description: txTemplate.description,
				Date:        date,
				Month:       date.Format("2006-01"),
				Status:      "completed",
				Type:        txType,
			}

			db.Collection("Transaction").InsertOne(context.TODO(), tx)
			totalTx++
		}
		fmt.Printf("   ✅ User index %d: %d transactions\n", i, numTransactions)
	}
	fmt.Printf("   📊 Total transactions created: %d\n", totalTx)
}

func seedBudgets(db *mongo.Database, userIDs []primitive.ObjectID) {
	fmt.Println("📅 Seeding budgets...")

	month := time.Now().Format("2006-01")

	budgetTemplates := []struct {
		monthlyLimit float64
		categories   map[string]float64
	}{
		{10000000, map[string]float64{"food": 3000000, "transport": 1500000, "entertainment": 1000000, "shopping": 2000000}},
		{15000000, map[string]float64{"food": 4000000, "transport": 2000000, "entertainment": 1500000, "shopping": 3000000}},
		{8000000, map[string]float64{"food": 2500000, "transport": 1000000, "entertainment": 800000, "shopping": 1500000}},
		{12000000, map[string]float64{"food": 3500000, "transport": 1500000, "entertainment": 1200000, "shopping": 2500000}},
		{6000000, map[string]float64{"food": 2000000, "transport": 800000, "entertainment": 600000, "shopping": 1000000}},
	}

	for i, userID := range userIDs {
		template := budgetTemplates[i%len(budgetTemplates)]

		// Monthly budget
		monthlyBudget := models.MonthlyBudget{
			ID:     primitive.NewObjectID(),
			UserID: userID.Hex(),
			Month:  month,
			Limit:  template.monthlyLimit,
		}
		db.Collection("monthly_budget").InsertOne(context.TODO(), monthlyBudget)

		// Category budgets
		for category, limit := range template.categories {
			spent := limit * (0.6 + float64(i)*0.1) // Different spending levels

			catBudget := models.CategoryBudget{
				ID:        primitive.NewObjectID(),
				UserID:    userID.Hex(),
				Month:     month,
				Category:  category,
				Limit:     limit,
				Spent:     spent,
				CreatedAt: time.Now().AddDate(0, 0, -5),
				UpdatedAt: time.Now(),
			}
			db.Collection("category_budget").InsertOne(context.TODO(), catBudget)
		}

		fmt.Printf("   ✅ User index %d: Monthly (%s) + %d categories\n", i, month, len(template.categories))
	}
}

func seedInvestments(db *mongo.Database, userIDs []primitive.ObjectID) {
	fmt.Println("📈 Seeding investments...")

	investmentTemplates := []struct {
		name     string
		symbol   string
		invType  models.InvestmentType
		quantity float64
		avgCost  float64
	}{
		{"Bitcoin", "BTC", models.InvCrypto, 0.05, 650000000},    // IDR
		{"Ethereum", "ETH", models.InvCrypto, 0.5, 35000000},
		{"Solana", "SOL", models.InvCrypto, 10, 1500000},
		{"Apple Inc", "AAPL", models.InvStock, 10, 2500000},
		{"Google", "GOOGL", models.InvStock, 5, 2800000},
		{"Tesla", "TSLA", models.InvStock, 5, 3000000},
	}

	for i, userID := range userIDs {
		for j, inv := range investmentTemplates {
			if (i+j)%3 == 0 { // Not every user has every investment
				continue
			}

			currentPrice := inv.avgCost * (1 + float64(i+j%3)*0.05)
			totalCost := inv.quantity * inv.avgCost
			totalValue := inv.quantity * currentPrice

			investment := models.Investment{
				ID:            primitive.NewObjectID(),
				UserID:        userID,
				Name:          inv.name,
				Symbol:        inv.symbol,
				Type:          inv.invType,
				Quantity:      inv.quantity,
				AverageCost:   inv.avgCost,
				CurrentPrice:  currentPrice,
				Currency:      "IDR",
				TotalValue:    totalValue,
				TotalCost:     totalCost,
				GainLoss:      totalValue - totalCost,
				GainLossPercent: ((totalValue - totalCost) / totalCost) * 100,
				PurchaseDate:    time.Now().AddDate(0, -6, 0),
				LastPriceUpdate: time.Now(),
				IsActive:        true,
				Tags:            []string{"growth", "tech"},
				CreatedAt:       time.Now().AddDate(0, -6, 0),
				UpdatedAt:       time.Now(),
			}

			db.Collection("investments").InsertOne(context.TODO(), investment)
		}
		fmt.Printf("   ✅ User index %d: investments seeded\n", i)
	}
}

func seedDebts(db *mongo.Database, userIDs []primitive.ObjectID) {
	fmt.Println("🏛️ Seeding debts...")

	debtTemplates := []struct {
		name     string
		debtType models.DebtType
		creditor string
		original float64
		current  float64
		interest float64
		tenor    int
	}{
		{"KPR Rumah Bintaro", models.DebtMortgage, "Bank Mandiri", 500000000, 450000000, 8.5, 120},
		{"Kredit Mobil", models.DebtLoan, "BCA Finance", 200000000, 150000000, 7.0, 60},
		{"Credit Card BCA", models.DebtCreditCard, "Bank BCA", 5000000, 2500000, 2.5, 0},
		{"Pinjaman Personal", models.DebtPersonal, "CIMB Niaga", 10000000, 7500000, 12.0, 24},
	}

	for i, userID := range userIDs {
		for j, debt := range debtTemplates {
			if (i+j)%2 == 0 { // Not every user has every debt
				continue
			}

			paymentAmount := debt.current / float64(debt.tenor)
			if debt.tenor == 0 {
				paymentAmount = 500000
			}

			nextPayment := time.Now().AddDate(0, 0, 5+j*3)

			d := models.Debt{
				ID:                primitive.NewObjectID(),
				UserID:            userID,
				Name:              debt.name,
				Type:              debt.debtType,
				Creditor:          debt.creditor,
				Currency:          "IDR",
				OriginalAmount:    debt.original,
				CurrentBalance:    debt.current,
				InterestRate:      debt.interest,
				APRType:           "fixed",
				PaymentAmount:     paymentAmount,
				PaymentFrequency:  models.RepayMonthly,
				MinimumPayment:    paymentAmount * 0.8,
				NextPaymentDate:   nextPayment,
				StartDate:         time.Now().AddDate(0, -12, 0),
				IsActive:          true,
				IsPaidOff:         false,
				TotalPaid:         debt.original - debt.current,
				TotalInterest:     (debt.original - debt.current) * 0.15,
				RemainingPayments: debt.tenor - 12,
				TenorMonths:       debt.tenor,
				IsRecommended:     debt.interest < 10,
				Notes:             fmt.Sprintf("Debt for user index %d", i),
				CreatedAt:         time.Now().AddDate(0, -12, 0),
				UpdatedAt:         time.Now(),
			}

			db.Collection("debts").InsertOne(context.TODO(), d)
		}
		fmt.Printf("   ✅ User index %d: debts seeded\n", i)
	}
}

func seedSavingsGoals(db *mongo.Database, userIDs []primitive.ObjectID) {
	fmt.Println("🎯 Seeding savings goals...")

	goalTemplates := []struct {
		name        string
		description string
		target      float64
		current     float64
		category    string
		daysUntil   int
	}{
		{"Vacation to Japan", "Summer trip 2024", 15000000, 7500000, "travel", 180},
		{"Emergency Fund", "6 months expenses", 50000000, 30000000, "emergency", 365},
		{"New Laptop", "MacBook Pro", 25000000, 10000000, "electronics", 90},
		{"Down Payment Car", "Vehicle down payment", 40000000, 15000000, "vehicle", 365},
	}

	for i, userID := range userIDs {
		for j, goal := range goalTemplates {
			if (i+j)%2 == 0 {
				continue
			}

			targetDate := time.Now().AddDate(0, 0, goal.daysUntil+j*30)

			g := models.SavingsGoal{
				ID:            primitive.NewObjectID(),
				UserID:        userID,
				Name:          goal.name,
				Description:   goal.description,
				TargetAmount:  goal.target,
				CurrentAmount: goal.current + float64(i*1000000),
				StartDate:     time.Now().AddDate(0, -3, 0),
				TargetDate:    targetDate,
				Category:      goal.category,
				Priority:      (j % 3) + 1,
				Status:        "active",
				CreatedAt:     time.Now().AddDate(0, -3, 0),
				UpdatedAt:     time.Now(),
			}

			db.Collection("savings_goals").InsertOne(context.TODO(), g)
		}
		fmt.Printf("   ✅ User index %d: savings goals seeded\n", i)
	}
}

func seedBillReminders(db *mongo.Database, userIDs []primitive.ObjectID) {
	fmt.Println("📬 Seeding bill reminders...")

	billTemplates := []struct {
		name           string
		amount         float64
		payTo          string
		category       string
		billingCycle   string
		dayOfMonth     int
		remindDays     int
	}{
		{"Internet Home", 350000, "First Media", "bills", "monthly", 15, 3},
		{"Netflix", 186000, "Netflix Inc", "entertainment", "monthly", 20, 5},
		{"Spotify", 49000, "Spotify AB", "entertainment", "monthly", 10, 3},
		{"Electricity", 500000, "PLN", "utilities", "monthly", 5, 3},
		{"Water", 150000, "PDAM", "utilities", "monthly", 1, 5},
		{"Phone Plan", 200000, "Telkomsel", "bills", "monthly", 25, 3},
	}

	for i, userID := range userIDs {
		for j, bill := range billTemplates {
			nextDue := time.Now().AddDate(0, 0, bill.dayOfMonth-time.Now().Day()+j*5)

			b := models.BillReminder{
				ID:              primitive.NewObjectID(),
				UserID:          userID,
				Name:            bill.name,
				Description:     fmt.Sprintf("%s subscription", bill.name),
				Amount:          bill.amount + float64(i*10000),
				Currency:        "IDR",
				Category:        bill.category,
				PayTo:           bill.payTo,
				BillingCycle:   bill.billingCycle,
				DayOfMonth:      bill.dayOfMonth,
				RemindDaysBefore: bill.remindDays,
				ReminderDays:     []int{7, 3, 1},
				IsPaid:          time.Now().Day() > bill.dayOfMonth,
				LastPaidDate:    func() *time.Time { t := time.Now().AddDate(0, 0, -bill.dayOfMonth); return &t }(),
				NextDueDate:     nextDue,
				IsActive:        true,
				AutoPayEnabled:  i%2 == 0,
				Icon:            "receipt",
				Color:           "#FF9800",
				Tags:            []string{"recurring", bill.billingCycle},
				CreatedAt:       time.Now().AddDate(0, -6, 0),
				UpdatedAt:       time.Now(),
			}

			db.Collection("bill_reminders").InsertOne(context.TODO(), b)
		}
		fmt.Printf("   ✅ User index %d: %d bill reminders\n", i, len(billTemplates))
	}
}

func seedRecurringTransactions(db *mongo.Database, userIDs []primitive.ObjectID) {
	fmt.Println("🔁 Seeding recurring transactions...")

	recurringTemplates := []struct {
		name        string
		amount      float64
		category    string
		txType      string
		frequency   models.RecurrenceFrequency
		dayOfMonth  int
		isActive    bool
	}{
		{"Netflix Subscription", 186000, "entertainment", "expense", models.FreqMonthly, 20, true},
		{"Spotify Premium", 49000, "entertainment", "expense", models.FreqMonthly, 10, true},
		{"Gym Membership", 300000, "health", "expense", models.FreqMonthly, 1, true},
		{"Monthly Salary", 15000000, "salary", "income", models.FreqMonthly, 25, true},
		{"Electricity Auto-Pay", 350000, "utilities", "expense", models.FreqMonthly, 15, true},
		{"Weekly Groceries", 200000, "food", "expense", models.FreqWeekly, 0, true},
	}

	for i, userID := range userIDs {
		for j, rec := range recurringTemplates {
			startDate := time.Now().AddDate(0, -3, 0)
			nextRun := startDate.AddDate(0, 0, 7*(j+1))

			r := models.RecurringTransaction{
				ID:             primitive.NewObjectID(),
				UserID:         userID,
				Name:           rec.name,
				Description:    fmt.Sprintf("Automated %s", rec.name),
				Amount:         rec.amount + float64(i*10000),
				Currency:       "IDR",
				Category:       rec.category,
				Type:          rec.txType,
				AccountID:     primitive.NewObjectID(), // Would need real account ID in production
				Frequency:     rec.frequency,
				Interval:       1,
				StartDate:      startDate,
				NextRunDate:   nextRun,
				DayOfMonth:    rec.dayOfMonth,
				DayOfWeek:     1,
				IsActive:      rec.isActive,
				AutoCreate:    true,
				SkipWeekends:  false,
				CreatedAt:     time.Now().AddDate(0, -3, 0),
				UpdatedAt:     time.Now(),
			}

			db.Collection("recurring_transactions").InsertOne(context.TODO(), r)
		}
		fmt.Printf("   ✅ User index %d: %d recurring transactions\n", i, len(recurringTemplates))
	}
}

func seedNotifications(db *mongo.Database, userIDs []primitive.ObjectID) {
	fmt.Println("🔔 Seeding notifications...")

	notificationTypes := []struct {
		title       string
		message     string
		notifType   models.NotificationType
	}{
		{"Welcome to FinanceAPI!", "Thank you for joining. Start tracking your finances today.", models.NotifTypeSystem},
		{"Budget Alert", "You've reached 80% of your monthly food budget.", models.NotifTypeBudget},
		{"Large Transaction Detected", "A transaction of Rp 5,000,000 was recorded.", models.NotifTypeTransaction},
		{"Bill Reminder", "Netflix subscription is due in 3 days.", models.NotifTypeBill},
		{"Savings Goal Update", "You're 50% closer to your Vacation goal!", models.NotifTypeGoal},
	}

	for i, userID := range userIDs {
		for j, notif := range notificationTypes {
			n := models.UserNotification{
				ID:        primitive.NewObjectID(),
				UserID:    userID,
				Title:     notif.title,
				Message:   notif.message,
				Type:      notif.notifType,
				IsRead:    j%2 == 0,
				CreatedAt: time.Now().AddDate(0, 0, -j),
			}

			db.Collection("notifications").InsertOne(context.TODO(), n)
		}
		fmt.Printf("   ✅ User index %d: %d notifications\n", i, len(notificationTypes))
	}
}

func seedSystemAlerts(db *mongo.Database) {
	fmt.Println("🚨 Seeding system alerts...")

	alerts := []models.SystemAlert{
		{
			ID:          primitive.NewObjectID(),
			Title:       "High Error Rate Detected",
			Description: "Server error rate exceeded 1% threshold in the last hour",
			Severity:    "Warning",
			Type:        "system_alert",
			IsRead:      false,
			CanRetry:    false,
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			UpdatedAt:   time.Now().Add(-2 * time.Hour),
		},
		{
			ID:          primitive.NewObjectID(),
			Title:       "Failed Payment Processing",
			Description: "3 payment transactions failed due to gateway timeout",
			Severity:    "Error",
			Type:        "failed_action",
			IsRead:      false,
			CanRetry:    true,
			CreatedAt:   time.Now().Add(-30 * time.Minute),
			UpdatedAt:   time.Now().Add(-30 * time.Minute),
		},
		{
			ID:          primitive.NewObjectID(),
			Title:       "Database Backup Completed",
			Description: "Daily database backup completed successfully",
			Severity:    "Success",
			Type:        "system_alert",
			IsRead:      true,
			CanRetry:    false,
			CreatedAt:   time.Now().Add(-4 * time.Hour),
			UpdatedAt:   time.Now().Add(-4 * time.Hour),
		},
		{
			ID:          primitive.NewObjectID(),
			Title:       "API Rate Limit Warning",
			Description: "Some users approaching rate limit (80%)",
			Severity:    "Info",
			Type:        "system_alert",
			IsRead:      false,
			CanRetry:    false,
			CreatedAt:   time.Now().Add(-1 * time.Hour),
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		},
	}

	for _, alert := range alerts {
		db.Collection("system_alerts").InsertOne(context.TODO(), alert)
	}

	fmt.Println("   ✅ System alerts seeded")
}
