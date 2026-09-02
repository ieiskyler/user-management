# Go User Management API

A robust, production-ready User Management RESTful API built with **Go (Golang)**, **Gin**, **GORM**, and **PostgreSQL**, featuring secure password hashing (`bcrypt`) and stateless authentication via **JSON Web Tokens (JWT)**.

---

## Tech Stack

* **Language:** Go (Golang)
* **Web Framework:** Gin
* **ORM:** GORM
* **Database:** PostgreSQL
* **Security & Auth:** `bcrypt` (password hashing), `golang-jwt/jwt/v5` (JWT authentication)

---

## Structures

```text
user-management/
├── config/
│   └── database.go    # PostgreSQL connection and auto-migration
├── controllers/
│   └── auth.go        # Register, Login, and Profile logic
├── middlewares/
│   └── auth.go        # JWT verification and route protection
├── models/
│   └── user.go        # GORM User model schema
├── .env               # Local environment variables
├── go.mod
├── go.sum
└── main.go            # Entry point and route definitions