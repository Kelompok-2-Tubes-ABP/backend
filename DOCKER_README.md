# 🐳 FinanceAPI Docker Setup Guide

This guide helps frontend developers test all Admin API endpoints immediately using pre-populated test data.

## 🚀 Quick Start (One-Command Setup)

```bash
# Build and start everything (MongoDB + Seeder + API)
docker-compose up --build

# Or run in background
docker-compose up --build -d
```

The seeder will automatically run and populate the database with test data, then the API will start.

## 📍 API Endpoints

Base URL: `http://localhost:8080`

### Authentication

```bash
# Login as Admin
curl -X POST http://localhost:8080/admin/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@financeapi.com","password":"Admin123!"}'
```

## 👤 Test Credentials

| Role | Email | Password |
|------|-------|----------|
| **Admin** | admin@financeapi.com | Admin123! |
| **User** | john@example.com | User123! |
| **User** | jane@example.com | User123! |
| **User** | bob@example.com | User123! |

## 🧪 Complete Test Workflow

### 1. Admin Login → Get Token

```bash
# Login
curl -X POST http://localhost:8080/admin/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@financeapi.com","password":"Admin123!"}'
```

Response:
```json
{
  "status": "success",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 7200,
  "admin": {
    "id": "60d5f...",
    "username": "admin",
    "email": "admin@financeapi.com"
  }
}
```

### 2. Get Dashboard Stats

```bash
curl -X GET http://localhost:8080/admin/dashboard \
  -H "Authorization: Bearer <TOKEN>"
```

### 3. List All Users

```bash
curl -X GET "http://localhost:8080/admin/users?page=1&limit=10" \
  -H "Authorization: Bearer <TOKEN>"
```

### 4. Get User Details

```bash
curl -X GET http://localhost:8080/admin/users/{user_id} \
  -H "Authorization: Bearer <TOKEN>"
```

### 5. Disable User

```bash
curl -X PATCH http://localhost:8080/admin/users/{user_id}/disable \
  -H "Authorization: Bearer <TOKEN>"
```

### 6. Get Recent Transactions

```bash
curl -X GET "http://localhost:8080/admin/transactions/recent?limit=20" \
  -H "Authorization: Bearer <TOKEN>"
```

### 7. Update Transaction Status

```bash
curl -X PATCH http://localhost:8080/admin/transactions/{tx_id}/status \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"status":"completed"}'
```

Valid status values: `pending`, `completed`, `failed`

### 8. Get Alert Stats

```bash
curl -X GET http://localhost:8080/admin/alerts/stats \
  -H "Authorization: Bearer <TOKEN>"
```

### 9. Get Audit Logs

```bash
curl -X GET "http://localhost:8080/admin/audit-logs?page=1&limit=10" \
  -H "Authorization: Bearer <TOKEN>"
```

### 10. Export Audit Logs (CSV)

```bash
curl -X GET http://localhost:8080/admin/audit-logs/export \
  -H "Authorization: Bearer <TOKEN>" \
  --output audit_logs.csv
```

### 11. Get Investment Stats

```bash
curl -X GET http://localhost:8080/admin/investments/stats \
  -H "Authorization: Bearer <TOKEN>"
```

### 12. Export Investments (CSV)

```bash
curl -X GET http://localhost:8080/admin/investments/export \
  -H "Authorization: Bearer <TOKEN>" \
  --output investments.csv
```

## 📊 All Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/admin/login` | Admin login (public) |
| POST | `/admin/logout` | Admin logout |
| POST | `/admin/password` | Change admin password |
| GET | `/admin/dashboard` | Dashboard statistics |
| GET | `/admin/users` | List all users |
| GET | `/admin/users/:id` | Get user details |
| PATCH | `/admin/users/:id/disable` | Disable user |
| DELETE | `/admin/users/:id` | Delete user |
| GET | `/admin/transactions/recent` | Recent transactions |
| GET | `/admin/transactions/all` | All transactions (paginated) |
| PATCH | `/admin/transactions/:id/status` | Update transaction status |
| DELETE | `/admin/transactions/:id` | Delete transaction |
| GET | `/admin/alerts/stats` | Alert statistics |
| GET | `/admin/alerts` | List alerts |
| PATCH | `/admin/alerts/:id/read` | Mark alert as read |
| GET | `/admin/audit-logs/stats` | Audit log stats |
| GET | `/admin/audit-logs` | List audit logs |
| GET | `/admin/audit-logs/export` | Export audit logs (CSV) |
| GET | `/admin/budget-savings/stats` | Budget & savings stats |
| GET | `/admin/budgets` | List all user budgets |
| GET | `/admin/savings-goals` | List all savings goals |
| GET | `/admin/investments/stats` | Investment statistics |
| GET | `/admin/investments` | List all investments |
| GET | `/admin/investments/export` | Export investments (CSV) |
| GET | `/admin/analytics` | Analytics & reports |
| GET | `/admin/analytics/export` | Export analytics report (CSV) |

## 📁 Project Structure

```
financeapi/
├── Dockerfile              # Main API Dockerfile
├── Dockerfile.seed         # Database seeder Dockerfile
├── docker-compose.yml      # Docker Compose configuration
├── cmd/
│   └── seed/
│       └── main.go         # Database seeder source
├── scripts/
│   ├── api_test_examples.json  # API test examples (for Postman/Insomnia)
│   └── mongo-init.js       # MongoDB initialization (optional)
└── essentials/
    ├── handler/             # HTTP handlers
    ├── services/           # Business logic
    └── routes/             # Route definitions
```

## 🔧 Useful Commands

```bash
# View running containers
docker-compose ps

# View logs
docker-compose logs -f api

# View seeder logs (run seeder manually if needed)
docker-compose run --rm seeder

# Stop everything
docker-compose down

# Rebuild and restart
docker-compose up --build --force-recreate

# Reseed database (delete containers + rebuild)
docker-compose down -v
docker-compose up --build
```

## 🗄️ Seeded Test Data

The database seeder creates:

| Data Type | Amount |
|-----------|--------|
| Admin accounts | 1 |
| User accounts | 5 |
| Accounts (per user) | 6 |
| Transactions (per user) | 30-70 |
| Budgets (monthly) | 5 |
| Category budgets | 20 |
| Investments | Various |
| Debts | Various |
| Savings goals | Various |
| Bill reminders | Various |
| Recurring transactions | Various |
| Notifications | 25 |
| System alerts | 4 |

## 🌐 Network Configuration

The Docker setup uses:
- **API**: `http://localhost:8080`
- **MongoDB**: `mongodb://mongodb:27017` (internal Docker network)

To access host machine's Ollama from Docker:
```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

## 📝 Environment Variables

In `docker-compose.yml`:
```yaml
environment:
  - PORT=8080
  - JWT_SECRET=dev-secret-change-in-production-32chars
  - MONGO_URI=mongodb://mongodb:27017
  - OLLAMA_URL=http://host.docker.internal:11434
```

## ⚠️ Important Notes

1. **Token Expiry**: JWT tokens expire after 2 hours
2. **Seeded Data**: The seeder clears ALL existing data on each run
3. **Docker Network**: MongoDB is only accessible within Docker network
4. **Non-Root User**: Docker runs as non-root user for security
5. **Health Checks**: API has health check on `/auth/login` endpoint

## 🐛 Troubleshooting

```bash
# MongoDB connection issues
docker-compose logs mongodb

# API not starting
docker-compose logs api

# Seeder failed
docker-compose logs seeder

# Check if ports are in use
lsof -i :8080
lsof -i :27017

# Rebuild without cache
docker-compose build --no-cache
```
