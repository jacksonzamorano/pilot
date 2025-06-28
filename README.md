# Pilot

Pilot is a batteries-included, high-performance HTTP framework for Go that provides everything you need to build robust web applications and APIs. The framework is designed around four core packages that work seamlessly together to handle common backend development tasks with minimal boilerplate and maximum type safety.

A Repack blueprint file is provided, if you already use [Repack](https://github.com/jacksonzamorano/repack), my codegen tool.

## 🚀 Key Features

- **Type-Safe Database Operations**: Fluent PostgreSQL query builder with generics support
- **High-Performance HTTP Server**: Custom HTTP server with connection pooling and middleware support
- **Secure Authentication**: AES-encrypted stateless authentication tokens
- **Advanced JSON Processing**: Custom JSON parser with field-by-field validation
- **Zero-Config Setup**: Sensible defaults that work out of the box
- **Production Ready**: Built-in CORS, request logging, and error handling

## 📦 Package Overview

### `pilot_http` - HTTP Server & Routing
A high-performance HTTP server with a clean, intuitive API for building REST APIs. Features include:
- Type-safe route handlers with middleware support
- Built-in CORS configuration
- Connection pooling with configurable worker threads
- Request/response utilities with automatic error handling
- Integration with PostgreSQL connection pools

### `pilot_db` - PostgreSQL Query Builder
A fluent, type-safe query builder that eliminates SQL injection vulnerabilities and offers an ORM-like parsing experience while still allowing you to write SQL.
- Generic query builder for SELECT, INSERT, UPDATE, DELETE operations
- Support for complex joins
- Automatic parameter binding and type conversion
- Transaction support with bulk operations
- Custom error types with HTTP response integration

### `pilot_json` - JSON Parser & Validator
A custom JSON parsing library that provides granular control, individual field parsing, and better error handling than the standard library:
- Field-by-field validation with detailed error messages
- Support for nested objects and arrays
- Custom error types with field path tracking
- Type-safe conversion methods for all common Go types

### `pilot_exchange` - Secure Authentication
Stateless authentication system using AES encryption for secure token management:
- AES-256-CBC encryption with random initialization vectors
- JSON-based payload structure with automatic marshaling
- Built-in expiration handling for time-limited tokens
- Environment-based secret key configuration
- No server-side session storage required

## 🛠️ Quick Start

### Installation

```bash
go mod init your-project
go get github.com/jacksonzamorano/pilot
```

### Basic Setup

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/jacksonzamorano/pilot/pilot-http"
    "github.com/jacksonzamorano/pilot/pilot-db"
    "github.com/jacksonzamorano/pilot/pilot-exchange"
)

// Define your data structures
type User struct {
    ID       int64     `json:"id"`
    Name     string    `json:"name"`
    Email    string    `json:"email"`
    Created  time.Time `json:"created_at"`
}

// Row conversion function for the query builder
func userFromRow(row pgx.Rows) (*User, error) {
    var user User
    err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Created)
    return &user, err
}

// Authentication middleware
func authMiddleware(req *pilot_http.HttpRequest) *AuthState {
    // Extract and validate authentication token
    token := req.GetHeader("Authorization")
    if token == "" {
        return &AuthState{Authenticated: false}
    }
    
    payload := pilot_exchange.DecodeJson[pilot_exchange.AuthPayload](token)
    if payload == nil || payload.Expiration.Before(time.Now()) {
        return &AuthState{Authenticated: false}
    }
    
    return &AuthState{
        Authenticated: true,
        UserID:       payload.AccountId,
    }
}

type AuthState struct {
    Authenticated bool
    UserID        int64
}

func main() {
    // Database configuration
    dbConfig := pilot_http.DatabaseConfiguration{
        Host:     "localhost",
        Port:     5432,
        Database: "myapp",
        Username: "postgres",
        Password: "password",
    }
    
    // Create application
    app := pilot_http.NewApplication(":8080", dbConfig, authMiddleware)
    
    // Add routes
    app.Routes.Get("/users", getUsers)
    app.Routes.Post("/users", createUser)
    app.Routes.Get("/users/:id", getUser)
    
    // Start server
    log.Println("Starting server on :8080")
    app.Start()
}

// Route handlers
func getUsers(req *pilot_http.HttpRequest, db *pgxpool.Conn, auth *AuthState) *pilot_http.HttpResponse {
    if !auth.Authenticated {
        return pilot_http.UnauthorizedResponse("Authentication required")
    }
    
    query := pilot_db.Select("users", userFromRow).
        Select("id").Select("name").Select("email").Select("created_at").
        SortDesc("created_at").
        Limit(50)
    
    users, err := query.QueryMany(context.Background(), db)
    if err != nil {
        return err.Response()
    }
    
    return pilot_http.JsonResponse(users)
}

func createUser(req *pilot_http.HttpRequest, db *pgxpool.Conn, auth *AuthState) *pilot_http.HttpResponse {
    if !auth.Authenticated {
        return pilot_http.UnauthorizedResponse("Authentication required")
    }
    
    // Parse request body
    body := req.GetBody()
    obj := pilot_json.NewJsonObject()
    err := obj.Parse(&body)
    if err != nil {
        return pilot_http.BadRequestResponse("Invalid JSON")
    }
    
    name, err := obj.GetString("name")
    if err != nil {
        return pilot_http.BadRequestResponse("Name is required")
    }
    
    email, err := obj.GetString("email")
    if err != nil {
        return pilot_http.BadRequestResponse("Email is required")
    }
    
    // Insert new user
    query := pilot_db.Insert("users", userFromRow).
        Set("name", *name).
        Set("email", *email).
        Set("created_at", time.Now())
    
    newUser, err := query.QueryOneExpected(context.Background(), db)
    if err != nil {
        return err.Response()
    }
    
    return pilot_http.JsonResponse(newUser)
}

func getUser(req *pilot_http.HttpRequest, db *pgxpool.Conn, auth *AuthState) *pilot_http.HttpResponse {
    if !auth.Authenticated {
        return pilot_http.UnauthorizedResponse("Authentication required")
    }
    
    userID := req.GetParam("id")
    
    query := pilot_db.Select("users", userFromRow).
        Select("id").Select("name").Select("email").Select("created_at").
        WhereEq("id", userID)
    
    user, err := query.QueryOne(context.Background(), db)
    if err != nil {
        return err.Response()
    }
    
    if user == nil {
        return pilot_http.NotFoundResponse("User not found")
    }
    
    return pilot_http.JsonResponse(user)
}
```

## 📚 Advanced Examples

### Complex Database Queries

```go
// Join multiple tables with filtering and sorting
// Note you can also use helper functions WhereEq, WhereLt,
// etc., instead of just Where
query := pilot_db.Select("users", userWithProfileFromRow).
    Select("id").Select("name").Select("email").
    InnerJoin("profiles", "id", "user_id").
    Select("bio").Select("avatar_url").
    InnerJoin("companies", "company_id", "id").
    SelectAs("name", "company_name").
    Where("active", "= $", true).
    Where("created_at", "> $", time.Now().AddDate(-1, 0, 0)).
    SortDesc("created_at").
    Limit(25)

users, err := query.QueryMany(ctx, conn)
```

### Bulk Database Operations

```go
// Bulk insert multiple records
query := pilot_db.Insert("products", productFromRow).
    Set("name", "Product 1").Set("price", 19.99).Set("category", "electronics").
    Set("name", "Product 2").Set("price", 29.99).Set("category", "electronics").
    Set("name", "Product 3").Set("price", 39.99).Set("category", "electronics")

err := query.QueryInTransaction(ctx, tx)
```

### Custom Middleware

```go
func loggingMiddleware(req *pilot_http.HttpRequest, db *pgxpool.Conn, auth *AuthState) *pilot_http.HttpResponse {
    start := time.Now()
    log.Printf("Request: %s %s from %s", req.Method, req.Path, req.GetHeader("User-Agent"))
    
    // Continue to next middleware/handler by returning nil
    return nil
}

func rateLimitMiddleware(req *pilot_http.HttpRequest, db *pgxpool.Conn, auth *AuthState) *pilot_http.HttpResponse {
    // Check rate limit for this IP
    clientIP := req.GetClientIP()
    if isRateLimited(clientIP) {
        return pilot_http.ErrorResponse("Rate limit exceeded", 429)
    }
    
    return nil // Continue to next middleware
}

// Add middleware to specific routes
app.Routes.GetWithMiddleware("/api/data", getData, []Middleware{
    loggingMiddleware,
    rateLimitMiddleware,
})
```

### Secure Token Management

```go
// Create authentication token
payload := pilot_exchange.AuthPayload{
    AccountId:  user.ID,
    Expiration: time.Now().Add(24 * time.Hour),
}
token := pilot_exchange.EncodeJson(payload)

// Send token to client
response := pilot_http.JsonResponse(map[string]string{"token": token})
response.SetHeader("Authorization", "Bearer " + token)

// Later, validate token
func validateToken(tokenString string) (*pilot_exchange.AuthPayload, bool) {
    payload := pilot_exchange.DecodeJson[pilot_exchange.AuthPayload](tokenString)
    if payload == nil {
        return nil, false
    }
    
    if payload.Expiration.Before(time.Now()) {
        return nil, false // Token expired
    }
    
    return payload, true
}
```

## 🔧 Configuration

### Environment Variables

```bash
# Database connection
DATABASE_URL=postgres://user:password@localhost:5432/dbname

# Authentication
SIGNING_KEY=your-32-byte-encryption-key-here

# Server configuration  
PORT=8080
CORS_ORIGIN=https://yourdomain.com
LOG_LEVEL=info
```

### Database Configuration

```go
config := pilot_http.DatabaseConfiguration{
    Host:            "localhost",
    Port:            5432,
    Database:        "myapp",
    Username:        "postgres", 
    Password:        "password",
    MaxConnections:  25,
    SSLMode:         "disable",
}
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
