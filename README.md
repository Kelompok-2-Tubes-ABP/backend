# 💰 FINANCEAPI: Advanced Financial AI & Asset Tracking 🚀

Sistem manajemen keuangan cerdas berbasis Golang yang mengintegrasikan pelacakan aset real-time, manajemen hutang otomatis, dan chatbot berbasis AI (NLP).

---

## 📌 API Information
- **Base URL**: `http://localhost:8080`
- **Default Port**: `8080`
- **Authentication**: Bearer Token (JWT) pada header `Authorization`.

---

## 📑 Daftar Isi
1. [Authentication](#-1-authentication)
2. [User Profile](#-2-user-profile)
3. [Financial Management](#-3-financial-management)
    - [Transactions](#-transactions)
    - [Budgeting](#-budgeting)
    - [Accounts & Transfers](#-accounts--transfers)
    - [Debt Management](#-debt-management)
    - [Recurring Transactions](#-recurring-transactions)
    - [Bill Reminders](#-bill-reminders)
4. [Investment & Market Prices](#-4-investment--market-prices)
    - [Portfolio Tracking](#-portfolio-tracking)
    - [Real-time Market Prices](#-real-time-market-prices)
5. [AI & Insights](#-5-ai--insights)
    - [Chatbot NLP](#-chatbot-nlp)
    - [Analytics & Health Score](#-analytics--health-score)

---

## 🔐 1. Authentication

### **Register User**
Mendaftarkan akun baru dan mengirimkan email verifikasi.
- **Method**: `POST`
- **URL**: `/auth/register`
- **Auth Status**: `Public`
- **Input Detail (JSON Body)**:
```json
{
  "username": "arsyadmaulana",
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```
- **Output Success**:
```json
{
  "message": "User registered successfully. Please check your email for verification code.",
  "user_id": "65f8a...",
  "username": "arsyadmaulana",
  "email": "user@example.com",
  "email_sent": true
}
```

### **Login**
Autentikasi user dan mendapatkan JWT Token.
- **Method**: `POST`
- **URL**: `/auth/login`
- **Auth Status**: `Public`
- **Input Detail (JSON Body)**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```
- **Output Success**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "status": "success"
}
```

### **Verify Email (Token Link)**
Verifikasi email melalui link yang dikirim ke email.
- **Method**: `GET`
- **URL**: `/auth/verify/:token`
- **Auth Status**: `Public`
- **Output Success**: `Redirect to Success Page or JSON message.`

### **Verify Code**
Verifikasi akun menggunakan kode 6 digit.
- **Method**: `POST`
- **URL**: `/auth/verify/code`
- **Auth Status**: `Public`
- **Input Detail (JSON Body)**:
```json
{
  "email": "user@example.com",
  "code": "123456"
}
```

### **Reset Password Request**
Mengirimkan link reset password ke email.
- **Method**: `POST`
- **URL**: `/auth/reset/request`
- **Auth Status**: `Public`
- **Input Detail (JSON Body)**:
```json
{
  "email": "user@example.com"
}
```

### **Logout**
Menonaktifkan sesi token saat ini.
- **Method**: `POST`
- **URL**: `/auth/logout`
- **Auth Status**: `Protected (Bearer Token)`

---

## 👤 2. User Profile

### **Get Profile**
Melihat informasi detail user yang sedang login.
- **Method**: `GET`
- **URL**: `/profile/`
- **Auth Status**: `Protected (Bearer Token)`
- **Output Success**:
```json
{
  "id": "65f8a...",
  "username": "arsyadmaulana",
  "email": "user@example.com",
  "is_active": true,
  "created_at": "2024-03-01T10:00:00Z"
}
```

---

## 💰 3. Financial Management

### 💸 **Transactions**

#### **Create Transaction**
Mencatat pemasukan atau pengeluaran baru.
- **Method**: `POST`
- **URL**: `/transaction/new`
- **Auth Status**: `Protected (Bearer Token)`
- **Input Detail (JSON Body)**:
```json
{
  "amount": 50000,
  "category": "lifestyle",
  "description": "Beli Kopi Starbucks",
  "type": "expense",
  "account_id": "65f8acc..."
}
```

#### **Get Transactions history**
Melihat semua riwayat transaksi.
- **Method**: `GET`
- **URL**: `/transaction/`
- **Auth Status**: `Protected (Bearer Token)`

---

### 📅 **Budgeting**

#### **Create Monthly Budget**
Menentukan limit pengeluaran bulanan (Global).
- **Method**: `POST`
- **URL**: `/budget/`
- **Auth Status**: `Protected (Bearer Token)`
- **Input Detail (JSON Body)**:
```json
{
  "amount": 5000000,
  "month": "2024-04"
}
```

#### **Create Category Budget**
Menentukan limit pengeluaran per kategori.
- **Method**: `POST`
- **URL**: `/budget/category`
- **Auth Status**: `Protected (Bearer Token)`
- **Input Detail (JSON Body)**:
```json
{
  "category": "food",
  "limit": 2000000,
  "month": "2024-04"
}
```

---

### 🏦 **Accounts & Transfers**

#### **Create Account**
Mendaftarkan Bank, E-Wallet, atau Cash.
- **Method**: `POST`
- **URL**: `/account/`
- **Auth Status**: `Protected (Bearer Token)`
- **Input Detail (JSON Body)**:
```json
{
  "name": "BCA Personal",
  "type": "bank",
  "institution": "BCA",
  "current_balance": 15000000,
  "currency": "IDR",
  "is_default": true
}
```

#### **Inter-Account Transfer**
Transfer saldo antar rekening sendiri.
- **Method**: `POST`
- **URL**: `/account/transfer`
- **Auth Status**: `Protected (Bearer Token)`
- **Input Detail (JSON Body)**:
```json
{
  "from_account_id": "65f8acc...1",
  "to_account_id": "65f8acc...2",
  "amount": 1000000,
  "note": "Top up GoPay"
}
```

---

### 🏛️ **Debt Management**

#### **Create Debt**
Mencatat hutang atau cicilan baru.
- **Method**: `POST`
- **URL**: `/debt/`
- **Auth Status**: `Protected (Bearer Token)`
- **Input Detail (JSON Body)**:
```json
{
  "name": "KPR Rumah Bintaro",
  "type": "mortgage",
  "creditor": "Bank Mandiri",
  "original_amount": 500000000,
  "interest_rate": 8.5,
  "tenor_months": 120,
  "payment_amount": 6500000,
  "start_date": "2024-01-01T00:00:00Z"
}
```

#### **Pay Debt**
Membayar cicilan hutang secara atomic (mengurangi saldo akun & update sisa hutang).
- **Method**: `POST`
- **URL**: `/debt/:id/pay`
- **Auth Status**: `Protected (Bearer Token)`
- **Input Detail (JSON Body)**:
```json
{
  "amount": 6500000,
  "account_id": "65f8acc..."
}
```

---

### 🔁 **Recurring Transactions**

#### **Create Subscription**
Mencatat transaksi otomatis (Netflix, Spotify, dll).
- **Method**: `POST`
- **URL**: `/recurring/`
- **Auth Status**: `Protected (Bearer Token)`
- **Input Detail (JSON Body)**:
```json
{
  "name": "Netflix Premium",
  "amount": 186000,
  "category": "entertainment",
  "frequency": "monthly",
  "start_date": "2024-04-12T00:00:00Z",
  "account_id": "65f8acc..."
}
```

---

## 💹 4. Investment & Market Prices

### 📈 **Portfolio Tracking**

#### **Create Investment**
Menambahkan aset investasi (Saham/Crypto).
- **Method**: `POST`
- **URL**: `/investment/`
- **Auth Status**: `Protected (Bearer Token)`
- **Input Detail (JSON Body)**:
```json
{
  "name": "Bitcoin",
  "symbol": "BTC",
  "type": "crypto",
  "quantity": 0.05,
  "average_cost": 65000,
  "currency": "USD"
}
```

#### **Portfolio with Live Prices**
Melihat semua aset beserta harga real-time (API terintegrasi).
- **Method**: `GET`
- **URL**: `/investment/portfolio?currency=idr`
- **Auth Status**: `Protected (Bearer Token)`

---

### 🌍 **Real-time Market Prices**

#### **Get All Live Prices**
Melihat harga pasar terkini untuk daftar pantauan (Stocks & Crypto).
- **Method**: `GET`
- **URL**: `/prices/all`
- **Auth Status**: `Public`

---

## 🤖 5. AI & Insights

### 💬 **Chatbot NLP**

#### **AI Message**
Interaksi dengan chatbot untuk manajemen keuangan via bahasa natural.
- **Method**: `POST`
- **URL**: `/chatbot/message`
- **Auth Status**: `Protected (Bearer Token)`
- **Input Detail (JSON Body)**:
```json
{
  "message": "Berapa total kekayaan bersih saya hari ini?"
}
```
- **Output Success**:
```json
{
  "response": "Total kekayaan bersih Anda saat ini adalah Rp145.200.000 (Aset: Rp160.200.000 - Hutang: Rp15.000.000).",
  "session_id": "8b2..."
}
```

---

### � **Analytics & Health Score**

#### **Full Analytics**
Laporan komprehensif kesehatan keuangan.
- **Method**: `GET`
- **URL**: `/analytics/`
- **Auth Status**: `Protected (Bearer Token)`

#### **Get Health Score**
Mendapatkan skor 1-100 tentang kondisi finansial user.
- **Method**: `GET`
- **URL**: `/insights/health`
- **Auth Status**: `Protected (Bearer Token)`

---
