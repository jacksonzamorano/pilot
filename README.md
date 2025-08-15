# Pilot

Pilot is a lightweight, high-performance HTTP framework for Go that provides essential tools for building robust web applications and APIs. The framework is designed as a single, cohesive package that combines HTTP routing, JSON processing, and database utilities with minimal boilerplate and maximum type safety.

## 🚀 Key Features

- **Type-Safe HTTP Server**: Generic HTTP server with middleware support and flexible routing
- **Advanced JSON Processing**: Custom JSON parser with field-by-field validation and error tracking
- **Database Integration**: Built-in support for database connections with context management
- **Zero-Config Setup**: Sensible defaults that work out of the box
- **Production Ready**: Built-in CORS, request logging, and error handling
- **Generics Support**: Full Go generics support for type-safe route state management
- **Worker Pool Architecture**: Configurable concurrent request processing
- **Middleware System**: Flexible middleware chain with early termination support
- **Comprehensive Error Handling**: Structured error responses with proper HTTP status codes

## 🛠️ Installation

```bash
go mod init your-project
go get github.com/jacksonzamorano/pilot
```

## 🚀 Quick Start

### Basic HTTP Server

```go
package main

import (
    "database/sql"
    "log"
    
    "github.com/jacksonzamorano/pilot"
    _ "github.com/lib/pq" // PostgreSQL driver
)

// Define your application state
type AppState struct {
    UserID        int64
    Authenticated bool
    IsAdmin       bool
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
    app.Routes.AddRoute(pilot.Get, "/health", healthCheck)
    app.Routes.AddRoute(pilot.Get, "/users", getUsers)
    app.Routes.AddRoute(pilot.Post, "/users", createUser)
    app.Routes.AddRoute(pilot.Get, "/users/:id", getUser)
    
    // Start server
    log.Println("Starting server on :8080")
    app.Start()
}

func healthCheck(req *pilot.RouteRequest[AppState]) *pilot.HttpResponse {
    return pilot.StringResponse("OK")
}
```
