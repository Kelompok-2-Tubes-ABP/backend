package constants

// Transaction Types
const (
	TrxIncome  = "income"
	TrxOutcome = "outcome"
)

// Transaction Categories for easier checks (optional but useful)
// Can be extended for standard expense categories

// Account Types
const (
	AccWallet     = "wallet"
	AccCash       = "cash"
	AccCredit     = "credit"
	AccSavings    = "savings"
	AccInvestment = "investment"
)

// Investment Types
const (
	InvCrypto     = "crypto"
	InvStock      = "stock"
	InvBond       = "bond"
	InvRealEstate = "real_estate"
)

// Bill Statuses
const (
	BillOverdue  = "overdue"
	BillDueSoon  = "due_soon"
	BillUpcoming = "upcoming"
)
