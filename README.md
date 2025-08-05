# Pilot

Pilot is a lightweight, high-performance HTTP framework for Go that provides essential tools for building robust web applications and APIs. The framework is designed as a single, cohesive package that combines HTTP routing, JSON processing, and database utilities with minimal boilerplate and maximum type safety.

A Repack blueprint file is provided, if you already use [Repack](https://github.com/jacksonzamorano/repack), my codegen tool.

## 🚀 Key Features

- **Type-Safe HTTP Server**: Generic HTTP server with middleware support and flexible routing
- **Advanced JSON Processing**: Custom JSON parser with field-by-field validation and error tracking
- **Database Integration**: Built-in support for database connections with context management
- **Zero-Config Setup**: Sensible defaults that work out of the box
- **Production Ready**: Built-in CORS, request logging, and error handling
- **Generics Support**: Full Go generics support for type-safe route state management

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
    "database/sql"
    "log"
    "time"
    
    "github.com/jacksonzamorano/pilot"
    _ "github.com/lib/pq" // PostgreSQL driver
)

// Define your route state (can be any type)
type AppState struct {
    UserID        int64
    Authenticated bool
}

// Define your data structures
type User struct {
    ID      int64     `json:"id"`
    Name    string    `json:"name"`
    Email   string    `json:"email"`
    Created time.Time `json:"created_at"`
}

func main() {
    // Database connection
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/dbname?sslmode=disable")
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.Close()
    
    // Create application with typed route state
    app := pilot.NewApplication[AppState](":8080", db)
    
    // Configure CORS
    app.CorsOrigin = "*"
    app.CorsHeaders = "Content-Type, Authorization"
    app.CorsMethods = "GET, POST, PUT, DELETE, OPTIONS"
    
    // Add routes
    app.Routes.Get("/users", getUsers)
    app.Routes.Post("/users", createUser)
    app.Routes.Get("/users/:id", getUser)
    
    // Start server
    log.Println("Starting server on :8080")
    app.Start()
}

// Route handlers with typed state
func getUsers(req *pilot.HttpRequest, db *sql.DB, state AppState) *pilot.HttpResponse {
    if !state.Authenticated {
        return pilot.UnauthorizedResponse("Authentication required")
    }
    
    rows, err := db.Query("SELECT id, name, email, created_at FROM users ORDER BY created_at DESC LIMIT 50")
    if err != nil {
        return pilot.InternalServerErrorResponse("Database error")
    }
    defer rows.Close()
    
    var users []User
    for rows.Next() {
        var user User
        if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Created); err != nil {
            continue
        }
        users = append(users, user)
    }
    
    return pilot.JsonResponse(users)
}

func createUser(req *pilot.HttpRequest, db *sql.DB, state AppState) *pilot.HttpResponse {
    if !state.Authenticated {
        return pilot.UnauthorizedResponse("Authentication required")
    }
    
    // Parse JSON request body
    body := req.GetBody()
    obj := pilot.NewJsonObject()
    if err := obj.Parse(&body); err != nil {
        return pilot.BadRequestResponse("Invalid JSON")
    }
    
    name, err := obj.GetString("name")
    if err != nil {
        return pilot.BadRequestResponse("Name is required")
    }
    
    email, err := obj.GetString("email")
    if err != nil {
        return pilot.BadRequestResponse("Email is required")
    }
    
    // Insert new user
    var userID int64
    err = db.QueryRow(
        "INSERT INTO users (name, email, created_at) VALUES ($1, $2, $3) RETURNING id",
        *name, *email, time.Now(),
    ).Scan(&userID)
    
    if err != nil {
        return pilot.InternalServerErrorResponse("Failed to create user")
    }
    
    user := User{
        ID:      userID,
        Name:    *name,
        Email:   *email,
        Created: time.Now(),
    }
    
    return pilot.JsonResponse(user)
}

func getUser(req *pilot.HttpRequest, db *sql.DB, state AppState) *pilot.HttpResponse {
    if !state.Authenticated {
        return pilot.UnauthorizedResponse("Authentication required")
    }
    
    userID := req.GetParam("id")
    
    var user User
    err := db.QueryRow(
        "SELECT id, name, email, created_at FROM users WHERE id = $1",
        userID,
    ).Scan(&user.ID, &user.Name, &user.Email, &user.Created)
    
    if err == sql.ErrNoRows {
        return pilot.NotFoundResponse("User not found")
    } else if err != nil {
        return pilot.InternalServerErrorResponse("Database error")
    }
    
    return pilot.JsonResponse(user)
}
```

## 📚 Advanced Examples

### Working with JSON Arrays

```go
func handleBulkData(req *pilot.HttpRequest, db *sql.DB, state AppState) *pilot.HttpResponse {
    body := req.GetBody()
    obj := pilot.NewJsonObject()
    if err := obj.Parse(&body); err != nil {
        return pilot.BadRequestResponse("Invalid JSON")
    }
    
    // Get JSON array
    items, err := obj.GetArray("items")
    if err != nil {
        return pilot.BadRequestResponse("Items array required")
    }
    
    var results []map[string]interface{}
    for i := 0; i < items.Length(); i++ {
        itemData, err := items.GetData(i)
        if err != nil {
            continue
        }
        
        itemObj := pilot.NewJsonObject()
        if err := itemObj.Parse(itemData); err != nil {
            continue
        }
        
        name, _ := itemObj.GetString("name")
        value, _ := itemObj.GetInt64("value")
        
        if name != nil && value != nil {
            result := map[string]interface{}{
                "name":  *name,
                "value": *value,
            }
            results = append(results, result)
        }
    }
    
    return pilot.JsonResponse(results)
}
```

### Middleware Pattern

```go
// Middleware functions can be chained by checking the request/state
func authMiddleware(req *pilot.HttpRequest, db *sql.DB, state AppState) *pilot.HttpResponse {
    token := req.GetHeader("Authorization")
    if token == "" {
        return pilot.UnauthorizedResponse("No token provided")
    }
    
    // Validate token logic here...
    // For this example, assume token validation passes
    
    return nil // Continue to next handler
}

func rateLimitMiddleware(req *pilot.HttpRequest, db *sql.DB, state AppState) *pilot.HttpResponse {
    clientIP := req.GetClientIP()
    
    // Check rate limit (simplified example)
    var count int
    db.QueryRow("SELECT COUNT(*) FROM requests WHERE ip = $1 AND created_at > NOW() - INTERVAL '1 minute'", clientIP).Scan(&count)
    
    if count > 60 {
        return pilot.ErrorResponse("Rate limit exceeded", 429)
    }
    
    // Log the request
    db.Exec("INSERT INTO requests (ip, created_at) VALUES ($1, NOW())", clientIP)
    
    return nil // Continue to next handler
}
```

### Advanced JSON Processing

```go
func processComplexData(req *pilot.HttpRequest, db *sql.DB, state AppState) *pilot.HttpResponse {
    body := req.GetBody()
    obj := pilot.NewJsonObject()
    if err := obj.Parse(&body); err != nil {
        return pilot.BadRequestResponse("Invalid JSON")
    }
    
    // Extract nested data
    userObj, err := obj.GetObject("user")
    if err != nil {
        return pilot.BadRequestResponse("User object required")
    }
    
    name, err := userObj.GetString("name")
    if err != nil {
        return pilot.BadRequestResponse("User name required")
    }
    
    age, err := userObj.GetInt64("age")
    if err != nil {
        return pilot.BadRequestResponse("User age required")
    }
    
    // Optional fields with defaults
    email, _ := userObj.GetString("email")
    emailValue := ""
    if email != nil {
        emailValue = *email
    }
    
    // Process the data
    result := map[string]interface{}{
        "processed_name": strings.ToUpper(*name),
        "age_category":   getAgeCategory(*age),
        "email":          emailValue,
        "timestamp":      time.Now(),
    }
    
    return pilot.JsonResponse(result)
}

func getAgeCategory(age int64) string {
    if age < 18 {
        return "minor"
    } else if age < 65 {
        return "adult"
    }
    return "senior"
}
```

## 🔧 Configuration

### Application Configuration

```go
// Create application with custom settings
app := pilot.NewApplication[YourStateType](":8080", db)

// CORS Configuration
app.CorsOrigin = "https://yourdomain.com"  // or "*" for all origins
app.CorsHeaders = "Content-Type, Authorization, X-Requested-With"
app.CorsMethods = "GET, POST, PUT, DELETE, OPTIONS"

// Server Configuration
app.WorkerCount = 20        // Number of worker goroutines (default: 10)
app.SilentMode = false      // Enable/disable request logging
app.LogRequestsLevel = 1    // 0=none, 1=basic, 2=detailed

// Start the server
app.Start()
```

### Context-Aware Applications

```go
// Create application with custom context
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

app := pilot.NewInlineApplication[AppState](":8080", db, ctx)

// The application will respect context cancellation
go func() {
    time.Sleep(30 * time.Second)
    cancel() // This will gracefully shutdown the server
}()

app.Start()
```

### Database Integration

```go
import (
    "database/sql"
    _ "github.com/lib/pq"           // PostgreSQL
    _ "github.com/go-sql-driver/mysql" // MySQL
    _ "github.com/mattn/go-sqlite3"    // SQLite
)

// PostgreSQL
db, err := sql.Open("postgres", "postgres://user:pass@localhost/dbname?sslmode=disable")

// MySQL
db, err := sql.Open("mysql", "user:pass@tcp(localhost:3306)/dbname")

// SQLite
db, err := sql.Open("sqlite3", "./database.db")

if err != nil {
    log.Fatal(err)
}

app := pilot.NewApplication[AppState](":8080", db)
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
