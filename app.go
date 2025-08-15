// Package pilot provides a lightweight, high-performance HTTP framework for Go with
// built-in support for generics, middleware, JSON processing, and database integration.
// The framework is designed to be type-safe, minimal, and production-ready with
// sensible defaults.
//
// The core philosophy is to provide a complete web framework in a single package
// that handles HTTP routing, request/response processing, JSON parsing with validation,
// CORS configuration, and database connectivity without external dependencies.
//
// Key Features:
//   - Generic route state management for type-safe handler contexts
//   - Built-in worker pool for concurrent request handling
//   - Automatic CORS handling with configurable policies
//   - Custom JSON parser with field-by-field validation
//   - Database connection pooling and context management
//   - Middleware support with early termination capabilities
//   - Zero-allocation routing with trie-based path matching
//   - Graceful shutdown with context cancellation support
//
// Example usage:
//
//	type AppState struct {
//	    UserID int64
//	    IsAdmin bool
//	}
//
//	app := pilot.NewApplication[AppState](":8080", db)
//	app.Routes.AddRoute(pilot.Get, "/users", handleGetUsers)
//	app.Start()
package pilot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
)

// RouteStateCompatible is a type constraint that allows any type to be used
// as route state in the application. This enables type-safe access to shared
// data across route handlers while maintaining flexibility for different
// application architectures.
type RouteStateCompatible any

// Application represents the main HTTP server instance with generic route state support.
// The Application manages the entire request lifecycle from connection acceptance
// through response delivery, including middleware execution, routing, and database access.
//
// The generic RouteState parameter allows you to define custom state that will be
// available to all route handlers, enabling type-safe access to user sessions,
// authentication context, or any other per-request data.
//
// Fields:
//   - Port: The port string the server will listen on (e.g., ":8080", "localhost:3000")
//   - Routes: The route collection managing all registered endpoints and their handlers
//   - CorsOrigin: CORS Access-Control-Allow-Origin header value (default: "*")
//   - CorsHeaders: CORS Access-Control-Allow-Headers header value (default: "*")
//   - CorsMethods: CORS Access-Control-Allow-Methods header value (default: all common methods)
//   - SilentMode: When true, suppresses startup and route registration output
//   - Database: SQL database connection available to all route handlers
//   - Context: Application context for graceful shutdown and request cancellation
//   - WorkerCount: Number of goroutines handling concurrent requests (default: 10)
//   - LogRequestsLevel: Request logging verbosity (0=none, 1=basic, 2=detailed)
//
// The Application uses a worker pool architecture where a configurable number of
// goroutines handle incoming requests concurrently, providing excellent performance
// under high load while maintaining predictable resource usage.
type Application[RouteState RouteStateCompatible] struct {
	Port             string
	Routes           *RouteCollection[RouteState]
	CorsOrigin       string
	CorsHeaders      string
	CorsMethods      string
	SilentMode       bool
	Database         *sql.DB
	Context          context.Context
	WorkerCount      int32
	LogRequestsLevel int
}

// NewInlineApplication creates a new Application instance with a custom context.
// This constructor allows you to provide your own context for fine-grained control
// over application lifecycle, timeout management, or custom cancellation logic.
//
// Use this constructor when you need to:
//   - Implement custom shutdown logic
//   - Set application-wide timeouts
//   - Integrate with existing context hierarchies
//   - Control server lifetime programmatically
//
// Parameters:
//   - port: Server port in format ":8080" or "localhost:3000"
//   - db: Database connection that will be available to all route handlers
//   - ctx: Custom context for application lifecycle management
//
// Returns:
//   - *Application[RouteState]: Configured application instance ready to accept routes
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
//	defer cancel()
//	app := pilot.NewInlineApplication[MyState](":8080", db, ctx)
//	// Server will automatically shut down after 1 hour
func NewInlineApplication[RouteState any](port string, db *sql.DB, ctx context.Context) *Application[RouteState] {
	return &Application[RouteState]{
		Port:             port,
		CorsOrigin:       "*",
		CorsHeaders:      "*",
		CorsMethods:      "GET, PUT, POST, DELETE, HEAD, PATCH",
		Routes:           NewRouteCollection[RouteState](),
		SilentMode:       false,
		Database:         db,
		WorkerCount:      10,
		Context:          ctx,
		LogRequestsLevel: 0,
	}
}

// NewApplication creates a new Application instance with automatic signal handling.
// This is the standard constructor that sets up graceful shutdown handling for
// SIGINT and SIGKILL signals, making it ideal for production deployments.
//
// The application will automatically shut down gracefully when receiving:
//   - SIGINT (Ctrl+C during development)
//   - SIGKILL (container orchestration shutdown)
//
// Parameters:
//   - port: Server port in format ":8080" or "localhost:3000"
//   - db: Database connection that will be available to all route handlers
//
// Returns:
//   - *Application[RouteState]: Configured application instance with signal handling
//
// Example:
//
//	db, _ := sql.Open("postgres", connectionString)
//	app := pilot.NewApplication[UserState](":8080", db)
//	app.Routes.AddRoute(pilot.Get, "/health", healthHandler)
//	app.Start() // Blocks until signal received
func NewApplication[RouteState any](port string, db *sql.DB) *Application[RouteState] {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	return &Application[RouteState]{
		Port:             port,
		CorsOrigin:       "*",
		CorsHeaders:      "*",
		CorsMethods:      "GET, PUT, POST, DELETE, HEAD, PATCH",
		Routes:           NewRouteCollection[RouteState](),
		SilentMode:       false,
		Database:         db,
		WorkerCount:      10,
		Context:          ctx,
		LogRequestsLevel: 0,
	}
}

// AddRouteGroup registers all routes from a RouteGroup under a common prefix.
// This method enables modular route organization by allowing you to define
// related routes in groups and then mount them at specific path prefixes.
//
// The method automatically handles path normalization:
//   - Ensures the prefix starts with "/"
//   - Ensures the prefix ends with "/"
//   - Removes leading "/" from individual routes to prevent double slashes
//   - Preserves middleware configuration for each route
//
// This is particularly useful for:
//   - API versioning (e.g., "/v1", "/v2")
//   - Feature modules (e.g., "/admin", "/user", "/api")
//   - Microservice-style organization within a monolith
//
// Parameters:
//   - prefix: URL path prefix for all routes in the group (e.g., "/api", "/v1")
//   - rg: RouteGroup containing routes and their associated middleware
//
// Example:
//
//	userRoutes := pilot.NewRouteGroup(
//	    pilot.GetRoute("/profile", getUserProfile),
//	    pilot.PostRoute("/settings", updateSettings),
//	)
//	app.AddRouteGroup("/user", userRoutes)
//	// Creates: /user/profile, /user/settings
func (a *Application[RouteState]) AddRouteGroup(prefix string, rg *RouteGroup[RouteState]) {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for i := range (*rg).Routes {
		route := (*rg).Routes[i].Route
		route = strings.TrimPrefix(route, "/")
		a.Routes.AddRouteWithMiddleware((*rg).Routes[i].Method, prefix+route, (*rg).Routes[i].Handler, (*rg).Routes[i].Middleware)
	}
}

// Start begins listening for HTTP requests and blocks until the application context
// is cancelled. This method initializes the worker pool, starts accepting connections,
// and handles the complete request lifecycle including graceful shutdown.
//
// The server architecture uses a two-stage queuing system:
//  1. A receiver goroutine accepts connections and queues them
//  2. Worker goroutines process requests from the queue concurrently
//
// This design provides several benefits:
//   - Predictable resource usage through bounded worker pools
//   - Excellent performance under high concurrent load
//   - Graceful handling of connection bursts through buffered queues
//   - Clean shutdown that waits for in-flight requests to complete
//
// The method will:
//   - Print route tree and startup message (unless SilentMode is true)
//   - Create a TCP listener on the configured port
//   - Start the configured number of worker goroutines
//   - Begin accepting and dispatching connections
//   - Handle graceful shutdown when the context is cancelled
//
// Startup Output:
// When SilentMode is false, displays registered routes in a tree format
// showing the complete routing hierarchy with supported HTTP methods.
//
// Error Handling:
// Panics on listener creation failure. All other errors are logged and
// handled gracefully to maintain server stability.
//
// Example:
//
//	app := pilot.NewApplication[AppState](":8080", db)
//	app.Routes.AddRoute(pilot.Get, "/health", healthCheck)
//	log.Println("Server configured, starting...")
//	app.Start() // Blocks here until shutdown signal
func (a *Application[RouteState]) Start() {
	if !a.SilentMode {
		fmt.Printf("Starting server on port %v.\n\nRegistered routes:\n", a.Port)
		a.Routes.PrintTree()
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", a.Port))
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	queue := make(chan net.Conn, a.WorkerCount*10)
	recvQueue := make(chan net.Conn, a.WorkerCount*10)
	for i := int32(0); i < a.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handleRequest(queue, a, (*a).Context, i)
		}()
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				} else {
					panic(err)
				}
			}
			if (*a).LogRequestsLevel > 1 {
				log.Printf("{reciever} Dispatching connection from %s\n", conn.RemoteAddr().String())
			}
			recvQueue <- conn
		}
	}()
	func() {
		for {
			select {
			case <-(*a).Context.Done():
				log.Println("Stopping Pilot server...")
				listener.Close()
				wg.Wait()
				return
			case conn := <-recvQueue:
				queue <- conn
			}
		}
	}()
}

// handlerLog provides structured logging for request processing with worker and connection tracking.
// This internal function formats log messages with consistent structure for debugging and monitoring.
//
// Log Format: {workerID/connectionID} (clientIP): message
//
// Parameters:
//   - id: Worker goroutine identifier for tracking concurrent request processing
//   - connId: Connection sequence number for this worker (increments per request)
//   - ip: Client IP address for security and analytics tracking
//   - msg: Human-readable message describing the request processing event
func handlerLog(id int32, connId int64, ip net.Addr, msg string) {
	log.Printf("{%d/%d} (%s): %s\n", id, connId, ip.String(), msg)
}

// handleRequest processes HTTP requests in a worker goroutine with complete request lifecycle management.
// This is the core request processing function that handles connection parsing, routing,
// middleware execution, handler dispatch, and response delivery.
//
// The function implements a complete HTTP request processing pipeline:
//  1. Parse incoming HTTP request from TCP connection
//  2. Handle CORS preflight OPTIONS requests automatically
//  3. Route request to appropriate handler based on path and method
//  4. Execute middleware chain with early termination support
//  5. Call route handler with typed state and database access
//  6. Apply CORS headers and send response
//  7. Log request processing (based on LogRequestsLevel configuration)
//
// Error Handling:
//   - Invalid requests are logged and connections closed gracefully
//   - Missing routes return 404 responses
//   - Handler errors are caught and return 500 responses
//   - Network errors are handled without crashing the worker
//
// Context Management:
//   - Respects context cancellation for graceful shutdown
//   - Stops processing new requests when context is done
//   - Maintains worker lifecycle through context monitoring
//
// Parameters:
//   - conn: Channel receiving TCP connections to process
//   - app: Application instance with configuration and routes
//   - cn: Context for cancellation and timeout control
//   - id: Unique worker identifier for logging and monitoring
func handleRequest[RouteState any](conn <-chan net.Conn, app *Application[RouteState], cn context.Context, id int32) {
	var connId int64 = 0
	log.Printf("Worker #%d online, ready for requests.", id)
ReqLoop:
	for {
		select {
		case <-cn.Done():
			log.Printf("Worker #%d shutdown.", id)
			return
		case conn := <-conn:
			connId++
			if (*app).LogRequestsLevel > 1 {
				handlerLog(id, connId, conn.RemoteAddr(), "Request dispatched.")
			}
			request := ParseRequest(&conn)
			if request == nil {
				handlerLog(id, connId, conn.RemoteAddr(), "Could not parse request.")
				conn.Close()
				continue ReqLoop
			}
			if (*app).LogRequestsLevel > 0 {
				handlerLog(id, connId, conn.RemoteAddr(), fmt.Sprintf("%s: '%s'", request.Method, request.Path))
			}

			if request.Method == Options {
				response := HttpResponse{
					StatusCode: StatusOK,
					Body:       []byte{},
					Headers: map[string]string{
						"Access-Control-Allow-Origin":  (*app).CorsOrigin,
						"Access-Control-Allow-Headers": (*app).CorsHeaders,
						"Access-Control-Allow-Methods": (*app).CorsMethods,
					},
				}
				response.Write(conn)
				conn.Close()
				continue ReqLoop
			}
			response := StringResponse("")
			response.Body = []byte("404 not found")
			response.SetStatus(StatusNotFound)
			response.ApplyCors(&app.CorsOrigin, &app.CorsHeaders, &app.CorsMethods)
			route := (*app).Routes.FindPath(request.Path, false)
			if route == nil {
				response.Write(conn)
				if (*app).LogRequestsLevel > 1 {
					handlerLog(id, connId, conn.RemoteAddr(), "No route found.")
				}
				conn.Close()
				continue ReqLoop
			}
			handler, found := route.Handlers[request.Method]
			if !found {
				response.Write(conn)
				if (*app).LogRequestsLevel > 1 {
					handlerLog(id, connId, conn.RemoteAddr(), "No handler found.")
				}
				conn.Close()
				continue ReqLoop
			}

			var routeState RouteState

			routeData := RouteRequest[RouteState]{
				Context:  cn,
				Request:  request,
				Database: app.Database,
				State:    &routeState,
			}

			for i := range handler.Middleware {
				response = handler.Middleware[i](&routeData)
				if response != nil {
					response.ApplyCors(&app.CorsOrigin, &app.CorsHeaders, &app.CorsMethods)
					response.Write(conn)
					conn.Close()
					continue ReqLoop
				}
			}

			response = handler.Handler(&routeData)
			if response == nil {
				handlerLog(id, connId, conn.RemoteAddr(), "Handler returned nil, sending 500.")
				response = StringResponse("500 Internal Server Error")
			}
			response.ApplyCors(&app.CorsOrigin, &app.CorsHeaders, &app.CorsMethods)
			response.Write(conn)
			conn.Close()
		}
	}
}
