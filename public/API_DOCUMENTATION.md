# FinanceAPI - Complete API Documentation

## 📋 Table of Contents
- [Authentication](#authentication)
- [User Endpoints](#user-endpoints)
- [Admin Endpoints](#admin-endpoints)

---

## 🔐 Authentication

All protected endpoints require a Bearer token in the Authorization header:
```
Authorization: Bearer <your_jwt_token>
```

---

## 👤 User Endpoints

### Authentication & Profile

#### POST /auth/login
**Request:**
```json
{
  "email": "john@example.com",
  "password": "User123!"
}
```
**Response (200):**
```json
{
  "message": "Login successful",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "name": "John Doe",
    "email": "john@example.com",
    "username": "johndoe"
  }
}
```

#### POST /auth/register
**Request:**
```json
{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "password": "User123!",
  "username": "janedoe"
}
```
**Response (201):**
```json
{
  "message": "User registered successfully",
  "user": {
    "id": "507f1f77bcf86cd799439012",
    "name": "Jane Doe",
    "email": "jane@example.com",
    "username": "janedoe",
    "is_verified": false
  }
}
```

#### GET /profile/
**Response (200):**
```json
{
  "id": "507f1f77bcf86cd799439011",
  "name": "John Doe",
  "email": "john@example.com",
  "username": "johndoe",
  "is_verified": true,
  "created_at": "2024-01-15T10:30:00Z"
}
```

#### POST /auth/password/change
**Request:**
```json
{
  "current_password": "User123!",
  "new_password": "NewUser123!"
}
```
**Response (200):**
```json
{
  "message": "Password changed successfully"
}
```

---

### Bank & Asset Accounts

#### POST /account/
**Request:**
```json
{
  "name": "BCA Savings",
  "type": "savings",
  "balance": 15000000
}
```
**Response (201):**
```json
{
  "message": "Account created successfully",
  "account": {
    "id": "60d5f484f1b2c72b8c8e4f1a",
    "name": "BCA Savings",
    "type": "savings",
    "balance": 15000000,
    "currency": "IDR",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

#### GET /account/
**Response (200):**
```json
{
  "accounts": [
    {
      "id": "60d5f484f1b2c72b8c8e4f1a",
      "name": "BCA Savings",
      "type": "savings",
      "balance": 15000000,
      "currency": "IDR"
    }
  ],
  "total": 3
}
```

#### GET /account/summary
**Response (200):**
```json
{
  "total_balance": 45000000,
  "accounts_count": 3,
  "by_type": {
    "savings": 25000000,
    "checking": 15000000,
    "investment": 5000000
  }
}
```

#### PATCH /account/:id/balance
**Request:**
```json
{
  "balance": 18500000
}
```
**Response (200):**
```json
{
  "message": "Balance updated successfully",
  "account": {
    "id": "60d5f484f1b2c72b8c8e4f1a",
    "balance": 18500000
  }
}
```

#### POST /account/transfer
**Request:**
```json
{
  "from_account_id": "60d5f484f1b2c72b8c8e4f1a",
  "to_account_id": "60d5f484f1b2c72b8c8e4f1b",
  "amount": 250000,
  "description": "Rent share payment"
}
```
**Response (200):**
```json
{
  "message": "Transfer successful",
  "transfer": {
    "id": "60d5f484f1b2c72b8c8e4f1c",
    "from_account_id": "60d5f484f1b2c72b8c8e4f1a",
    "to_account_id": "60d5f484f1b2c72b8c8e4f1b",
    "amount": 250000,
    "description": "Rent share payment",
    "created_at": "2024-01-15T14:30:00Z"
  }
}
```

---

### Income & Expenses

#### POST /transaction/new
**Request:**
```json
{
  "amount": 75000,
  "category": "food",
  "description": "Dinner at local diner",
  "account_id": "60d5f484f1b2c72b8c8e4f1a",
  "type": "expense",
  "date": "2024-01-15"
}
```
**Response (201):**
```json
{
  "message": "Transaction created successfully",
  "transaction": {
    "id": "60d5f484f1b2c72b8c8e4f1b",
    "amount": 75000,
    "category": "food",
    "description": "Dinner at local diner",
    "type": "expense",
    "account_id": "60d5f484f1b2c72b8c8e4f1a",
    "date": "2024-01-15T00:00:00Z",
    "created_at": "2024-01-15T19:30:00Z"
  }
}
```

#### GET /transaction/
**Response (200):**
```json
{
  "transactions": [
    {
      "id": "60d5f484f1b2c72b8c8e4f1b",
      "amount": 75000,
      "category": "food",
      "description": "Dinner at local diner",
      "type": "expense",
      "date": "2024-01-15T00:00:00Z"
    }
  ],
  "total": 150,
  "page": 1,
  "per_page": 20
}
```

#### PATCH /transaction/update/:id
**Request:**
```json
{
  "amount": 80000,
  "description": "Dinner with coffee"
}
```
**Response (200):**
```json
{
  "message": "Transaction updated successfully",
  "transaction": {
    "id": "60d5f484f1b2c72b8c8e4f1b",
    "amount": 80000,
    "description": "Dinner with coffee"
  }
}
```

#### GET /transaction/getMonthly
**Query Params:** `?month=2024-01`
**Response (200):**
```json
{
  "month": "2024-01",
  "total_income": 5000000,
  "total_expense": 3500000,
  "net": 1500000,
  "transactions_count": 45
}
```

---

### Budget Management

#### POST /budget/
**Request:**
```json
{
  "month": "2024-01",
  "amount": 5000000,
  "category": "monthly"
}
```
**Response (201):**
```json
{
  "message": "Budget created successfully",
  "budget": {
    "id": "60d5f484f1b2c72b8c8e4f1c",
    "month": "2024-01",
    "amount": 5000000,
    "spent": 0,
    "remaining": 5000000,
    "category": "monthly"
  }
}
```

#### GET /budget/summary
**Response (200):**
```json
{
  "current_month": "2024-01",
  "total_budget": 5000000,
  "total_spent": 3200000,
  "remaining": 1800000,
  "percentage_used": 64,
  "status": "on_track"
}
```

---

### Savings Goals

#### POST /savings_goal/
**Request:**
```json
{
  "name": "Emergency Fund",
  "target_amount": 50000000,
  "current_amount": 10000000,
  "deadline": "2024-12-31"
}
```
**Response (201):**
```json
{
  "message": "Savings goal created successfully",
  "savings_goal": {
    "id": "60d5f484f1b2c72b8c8e4f1d",
    "name": "Emergency Fund",
    "target_amount": 50000000,
    "current_amount": 10000000,
    "deadline": "2024-12-31T00:00:00Z",
    "progress_percentage": 20,
    "status": "in_progress"
  }
}
```

#### POST /savings_goal/:id/contribute
**Request:**
```json
{
  "amount": 1000000,
  "description": "Monthly contribution"
}
```
**Response (200):**
```json
{
  "message": "Contribution added successfully",
  "savings_goal": {
    "id": "60d5f484f1b2c72b8c8e4f1d",
    "current_amount": 11000000,
    "progress_percentage": 22
  }
}
```

---

### Investments & Portfolio

#### POST /investment/
**Request:**
```json
{
  "symbol": "BTC",
  "name": "Bitcoin",
  "type": "crypto",
  "quantity": 0.5,
  "purchase_price": 600000000,
  "purchase_date": "2024-01-15"
}
```
**Response (201):**
```json
{
  "message": "Investment created successfully",
  "investment": {
    "id": "60d5f484f1b2c72b8c8e4f1e",
    "symbol": "BTC",
    "name": "Bitcoin",
    "type": "crypto",
    "quantity": 0.5,
    "purchase_price": 600000000,
    "current_price": 650000000,
    "total_value": 325000000,
    "profit_loss": 25000000,
    "profit_loss_percentage": 8.33
  }
}
```

#### GET /investment/portfolio
**Response (200):**
```json
{
  "total_value": 325000000,
  "total_invested": 300000000,
  "total_profit_loss": 25000000,
  "profit_loss_percentage": 8.33,
  "investments": [
    {
      "id": "60d5f484f1b2c72b8c8e4f1e",
      "symbol": "BTC",
      "quantity": 0.5,
      "current_price": 650000000,
      "total_value": 325000000
    }
  ]
}
```

#### GET /prices/crypto
**Response (200):**
```json
{
  "BTC": {
    "price": 650000000,
    "change_24h": 2.5,
    "last_updated": "2024-01-15T20:00:00Z"
  },
  "ETH": {
    "price": 35000000,
    "change_24h": 1.8,
    "last_updated": "2024-01-15T20:00:00Z"
  }
}
```

---

### Debt Management

#### POST /debt/
**Request:**
```json
{
  "name": "Car Loan",
  "total_amount": 150000000,
  "remaining_amount": 120000000,
  "interest_rate": 8.5,
  "monthly_payment": 5000000,
  "due_date": "2026-12-31"
}
```
**Response (201):**
```json
{
  "message": "Debt created successfully",
  "debt": {
    "id": "60d5f484f1b2c72b8c8e4f1f",
    "name": "Car Loan",
    "total_amount": 150000000,
    "remaining_amount": 120000000,
    "interest_rate": 8.5,
    "monthly_payment": 5000000,
    "due_date": "2026-12-31T00:00:00Z"
  }
}
```

#### POST /debt/:id/pay
**Request:**
```json
{
  "amount": 5000000,
  "payment_date": "2024-01-15"
}
```
**Response (200):**
```json
{
  "message": "Payment recorded successfully",
  "debt": {
    "id": "60d5f484f1b2c72b8c8e4f1f",
    "remaining_amount": 115000000
  },
  "payment": {
    "id": "60d5f484f1b2c72b8c8e4f20",
    "amount": 5000000,
    "payment_date": "2024-01-15T00:00:00Z"
  }
}
```

---

### Analytics & Insights

#### GET /analytics/
**Response (200):**
```json
{
  "net_worth": 75000000,
  "total_assets": 100000000,
  "total_liabilities": 25000000,
  "monthly_income": 8000000,
  "monthly_expenses": 5500000,
  "savings_rate": 31.25,
  "top_expense_categories": [
    {"category": "food", "amount": 1500000},
    {"category": "transport", "amount": 1200000}
  ]
}
```

#### GET /insights/health
**Response (200):**
```json
{
  "health_score": 75,
  "rating": "Good",
  "factors": {
    "savings_rate": 85,
    "debt_to_income": 65,
    "emergency_fund": 70,
    "budget_adherence": 80
  },
  "recommendations": [
    "Increase emergency fund to 6 months expenses",
    "Consider paying off high-interest debt first"
  ]
}
```

---

## 🔧 Admin Endpoints

### Authentication

#### POST /admin/login
**Request:**
```json
{
  "email": "admin@financeapi.com",
  "password": "Admin123!"
}
```
**Response (200):**
```json
{
  "message": "Login successful",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "email": "admin@financeapi.com",
    "role": "admin"
  }
}
```

---

### Dashboard & Statistics

#### GET /admin/dashboard
**Response (200):**
```json
{
  "total_users": 1250,
  "active_users": 980,
  "total_transactions": 45678,
  "total_volume": 12500000.50,
  "new_users_today": 15,
  "transactions_today": 234,
  "revenue_this_month": 5600000
}
```

---

### User Management

#### GET /admin/users
**Response (200):**
```json
{
  "users": [
    {
      "id": "507f1f77bcf86cd799439011",
      "name": "John Doe",
      "email": "john@example.com",
      "username": "johndoe",
      "is_verified": true,
      "is_active": true,
      "created_at": "2024-01-15T10:30:00Z",
      "last_login": "2024-01-20T15:45:00Z"
    }
  ],
  "total": 1250,
  "page": 1,
  "per_page": 20
}
```

#### PATCH /admin/users/:id/disable
**Request:**
```json
{
  "is_active": false,
  "reason": "Suspicious activity"
}
```
**Response (200):**
```json
{
  "message": "User disabled successfully",
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "is_active": false
  }
}
```

---

## 📝 Notes

- All timestamps are in ISO 8601 format (UTC)
- Currency amounts are in the smallest unit (e.g., IDR cents)
- Pagination: Use `?page=1&per_page=20` query parameters
- Filtering: Use `?category=food&type=expense` for filtering
- Date ranges: Use `?start_date=2024-01-01&end_date=2024-01-31`

---

## 🚀 Quick Start

1. Register a new user: `POST /auth/register`
2. Login: `POST /auth/login`
3. Save the token from response
4. Use token in Authorization header for all protected endpoints
5. Start creating accounts, transactions, budgets, etc.

---

## 📞 Support

For issues or questions, please contact the development team.
