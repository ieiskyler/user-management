# Go User Management API

A RESTful User Management API built with Go, Gin, GORM, and PostgreSQL.

The API supports:

- User registration
- User login
- Password hashing with bcrypt
- JWT authentication
- Protected user listing
- Unit and integration tests

## Technology Stack

- Go 1.26+
- Gin
- GORM
- PostgreSQL
- JWT
- bcrypt
- SQLite for integration tests

## Project Structure

```text
user-management/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── auth.go
│   │   └── database.go
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── auth_handler_test.go
│   │   ├── user_handler.go
│   │   └── user_handler_test.go
│   ├── middleware/
│   │   ├── auth_middleware.go
│   │   └── auth_middleware_test.go
│   ├── models/
│   │   ├── user.go
│   │   └── user_test.go
│   ├── repository/
│   │   ├── user_repository.go
│   │   └── user_repository_test.go
│   ├── server/
│   │   ├── router.go
│   │   └── router_test.go
│   └── service/
│       ├── auth_service.go
│       ├── auth_service_test.go
│       ├── user_service.go
│       └── user_service_test.go
├── .env
├── go.mod
└── README.md
```

## Environment Configuration

Create a .env file in the project root:

```env
DB_HOST=localhost
DB_USER=your-postgres-user
DB_PASSWORD=your-postgres-password
DB_NAME=user_management
DB_PORT=5432
JWT_SECRET=replace-this-with-a-long-random-secret
```

## Prerequisites

Before running the API, make sure you have:

- Go 1.26 or later
- PostgreSQL running locally
- A PostgreSQL database named `user_management`

Create the database with:

```sql
CREATE DATABASE user_management;
```

## Installation

Clone the repository and enter the project directory:

```bash
git clone https://github.com/ieiskyler/user-management.git
cd user-management
```

Download the Go dependencies:

```bash
go mod download
```

## Running the API

Start the API with:

```bash
go run ./cmd/api
```

The server starts on:
```text
http://localhost:8080
```

The API base URL is:
```text
http://localhost:8080/api/v1
```

The application loads configuration from `.env`, connects to PostgreSQL, and automatically creates or updates the `users` table using GORM.

## API Endpoints

The API base URL is:

```text
http://localhost:8080/api/v1
```

### Register User

Creates a new user account.

```http
POST /api/v1/register
Content-Type: application/json
```

Request body:

```json
{
	"username": "johndoe",
	"email": "john@example.com",
	"password": "securepass123"
}
```

The username and email are required. The email must be valid and the password must contain at least 8 characters.

Successful response: `201 Created`

```json
{
	"id": "123e4567-e89b-12d3-a456-426614174000",
	"username": "johndoe",
	"email": "john@example.com",
	"created_at": "2026-09-04T00:00:00Z"
}
```

Duplicate username or email: `409 Conflict`

```json
{
	"error": "user already exists"
}
```

Example:

```bash
curl -X POST http://localhost:8080/api/v1/register \
	-H "Content-Type: application/json" \
	-d '{
		"username": "johndoe",
		"email": "john@example.com",
		"password": "securepass123"
	}'
```

### Login

Authenticates an existing user and returns a JWT.

```http
POST /api/v1/login
Content-Type: application/json
```

Request body:

```json
{
	"username": "johndoe",
	"password": "securepass123"
}
```

Successful response: `200 OK`

```json
{
	"token": "eyJhbGciOiJIUzI1NiIs..."
}
```

Invalid credentials: `401 Unauthorized`

```json
{
	"error": "invalid credentials"
}
```

Example:

```bash
curl -X POST http://localhost:8080/api/v1/login \
	-H "Content-Type: application/json" \
	-d '{
		"username": "johndoe",
		"password": "securepass123"
	}'
```

### List Users

Returns all registered users. This endpoint requires a valid JWT.

```http
GET /api/v1/users
Authorization: Bearer <token>
```

Successful response: `200 OK`

```json
{
	"users": [
		{
			"id": "123e4567-e89b-12d3-a456-426614174000",
			"username": "johndoe",
			"email": "john@example.com",
			"created_at": "2026-09-04T00:00:00Z"
		}
	]
}
```

Missing or invalid token: `401 Unauthorized`

```json
{
	"error": "Invalid or expired token"
}
```

Example:

```bash
curl http://localhost:8080/api/v1/users \
	-H "Authorization: Bearer <token>"
```

## Testing

Run all unit and integration tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

The project targets at least 85% statement coverage. Integration tests use an in-memory SQLite database and do not require PostgreSQL.

Run static analysis:

```bash
go vet ./...
```
