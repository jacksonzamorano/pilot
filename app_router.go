package pilot

import (
	"fmt"
	"strings"
)

// String returns the string representation of an HttpMethod for logging and debugging.
// This method enables HttpMethod to be used with fmt.Printf and other string formatting functions.
func (m HttpMethod) String() string {
	return string(m)
}

// RouteCollection manages the complete routing tree for an application using a trie-based
// data structure for efficient path matching. The collection supports dynamic route registration,
// middleware attachment, and hierarchical route organization.
//
// The trie structure enables:
//   - O(path_length) route lookup time regardless of route count
//   - Support for path parameters (e.g., "/users/:id")
//   - Hierarchical route organization matching URL structure
//   - Efficient memory usage through shared path prefixes
//
// Type Parameters:
//   - RouteState: The generic state type available to all route handlers
//
// Fields:
//   - Routes: Root-level route nodes forming the base of the routing trie
type RouteCollection[RouteState RouteStateCompatible] struct {
	Routes []*Route[RouteState]
}

// NewRouteCollection creates an empty route collection ready for route registration.
// This constructor initializes the routing trie structure that will efficiently
// match incoming requests to their appropriate handlers.
//
// Returns:
//   - *RouteCollection[RouteState]: Empty route collection ready for use
//
// Example:
//
//	routes := pilot.NewRouteCollection[AppState]()
//	routes.AddRoute(pilot.Get, "/users", getUsersHandler)
//	routes.AddRoute(pilot.Post, "/users", createUserHandler)
func NewRouteCollection[RouteState RouteStateCompatible]() *RouteCollection[RouteState] {
	return &RouteCollection[RouteState]{
		Routes: []*Route[RouteState]{},
	}
}

// PrintTree outputs the complete routing hierarchy to stdout for debugging and documentation.
// This method displays the routing trie structure showing all registered routes,
// their HTTP methods, and hierarchical relationships in a tree format.
//
// Output Format:
// Each route is displayed with its path component and supported HTTP methods:
//
//	/users [GET, POST]
//	 /profile [GET, PUT]
//	 /:id [GET, DELETE]
//
// This is automatically called during application startup unless SilentMode is enabled,
// providing immediate feedback about registered routes and helping identify routing conflicts.
func (self *RouteCollection[RouteState]) PrintTree() {
	for i := range self.Routes {
		self.Routes[i].PrintTree(0)
	}
}

// FindPath locates or optionally creates a route node in the routing trie for the given path.
// This method implements the core routing lookup algorithm that powers request dispatch.
// It traverses the trie structure matching path components and can dynamically create
// missing nodes when registering new routes.
//
// The algorithm:
//  1. Splits the path into components (e.g., "/users/profile" -> ["users", "profile"])
//  2. Traverses the trie matching each component against existing nodes
//  3. Returns the exact matching node or nil if not found (when create=false)
//  4. Creates missing nodes along the path when create=true
//
// Path Parameter Support:
// Supports path parameters using colon syntax (e.g., "/users/:id") where ":id"
// becomes a wildcard component that matches any value in that path segment.
//
// Parameters:
//   - path: URL path to find or create (e.g., "/api/users/profile")
//   - create: Whether to create missing route nodes along the path
//
// Returns:
//   - *Route[RouteState]: The route node for the path, or nil if not found and create=false
//
// Example:
//
//	// Find existing route
//	route := routes.FindPath("/users/profile", false)
//	if route != nil {
//	    // Route exists, can add handler
//	}
//
//	// Create route if missing
//	route = routes.FindPath("/users/settings", true)
//	// route is guaranteed to be non-nil
func (self *RouteCollection[RouteState]) FindPath(path string, create bool) *Route[RouteState] {
	comps := PathListFromString(path)
	var node *Route[RouteState] = nil
	for i := range self.Routes {
		if self.Routes[i].PathComponent == comps[0] {
			node = self.Routes[i]
			break
		}
	}
	if node == nil {
		if create {
			newNode := NewEmptyRoute[RouteState](comps[0])
			self.Routes = append(self.Routes, &newNode)
			node = &newNode
		} else {
			return nil
		}
	}
	i := 1
	for i < len(comps) {
		foundAtDepth := false
		for childIdx := range node.Children {
			if node.Children[childIdx].PathComponent == comps[i] {
				node = node.Children[childIdx]
				foundAtDepth = true
				break
			}
		}
		if !foundAtDepth && create {
			newRoute := NewEmptyRoute[RouteState](comps[i])
			node.Children = append(node.Children, &newRoute)
			node = &newRoute
		} else if !foundAtDepth {
			return nil
		}
		i++
	}
	return node
}

// AddRoute registers a route handler for the specified HTTP method and path without middleware.
// This is the simplest way to register routes and is equivalent to calling AddRouteWithMiddleware
// with an empty middleware slice.
//
// The method automatically creates the route path in the trie structure if it doesn't exist,
// then associates the handler with the specified HTTP method for that path.
//
// Parameters:
//   - method: HTTP method this handler responds to (Get, Post, Put, etc.)
//   - path: URL path pattern (e.g., "/users", "/users/:id", "/api/posts")
//   - fn: Handler function that processes requests to this endpoint
//
// Example:
//
//	routes.AddRoute(pilot.Get, "/users", func(req *pilot.RouteRequest[AppState]) *pilot.HttpResponse {
//	    return pilot.JsonResponse([]User{})
//	})
func (self *RouteCollection[RouteState]) AddRoute(method HttpMethod, path string, fn RouteHandlerFn[RouteState]) {
	self.FindPath(path, true).Handlers[method] = RouteHandler[RouteState]{
		Handler:    fn,
		Middleware: []MiddlewareFn[RouteState]{},
	}
}

// AddRouteWithMiddleware registers a route handler with associated middleware functions.
// Middleware functions are executed in order before the main handler, providing a powerful
// way to implement cross-cutting concerns like authentication, logging, and rate limiting.
//
// Middleware Execution:
//   - Middleware functions are called in the order provided in the slice
//   - If any middleware returns a non-nil HttpResponse, execution stops immediately
//   - The response from middleware is sent to the client without calling subsequent middleware or the handler
//   - If all middleware returns nil, the main handler is executed
//
// This pattern enables:
//   - Authentication/authorization checks that can terminate early
//   - Request validation and transformation
//   - Rate limiting with immediate error responses
//   - Logging and monitoring of request processing
//
// Parameters:
//   - method: HTTP method this handler responds to
//   - path: URL path pattern supporting parameters (e.g., "/users/:id")
//   - fn: Main handler function executed after all middleware passes
//   - middleware: Slice of middleware functions executed before the handler
//
// Example:
//
//	authMiddleware := func(req *pilot.RouteRequest[AppState]) *pilot.HttpResponse {
//	    if req.Request.GetHeader("Authorization") == "" {
//	        return pilot.UnauthorizedResponse("Missing auth token")
//	    }
//	    return nil // Continue to handler
//	}
//
//	routes.AddRouteWithMiddleware(pilot.Get, "/admin/users", adminHandler, []MiddlewareFn[AppState]{authMiddleware})
func (self *RouteCollection[RouteState]) AddRouteWithMiddleware(method HttpMethod, path string, fn RouteHandlerFn[RouteState], middleware []MiddlewareFn[RouteState]) {
	self.FindPath(path, true).Handlers[method] = RouteHandler[RouteState]{
		Handler:    fn,
		Middleware: middleware,
	}
}

// RouteHandlerFn defines the signature for route handler functions that process HTTP requests.
// Handlers receive a RouteRequest containing the HTTP request, database connection,
// application context, and typed route state, then return an HttpResponse.
//
// The RouteRequest provides access to:
//   - HTTP request details (headers, body, query parameters, path parameters)
//   - Database connection for data operations
//   - Application context for timeout and cancellation handling
//   - Typed route state for request-scoped data sharing
//
// Handler functions should be pure and stateless when possible, relying on the
// provided context and database connection rather than global state.
//
// Function signature:
//
//	func(req *RouteRequest[RouteState]) *HttpResponse
type RouteHandlerFn[RouteState RouteStateCompatible] func(*RouteRequest[RouteState]) *HttpResponse

// MiddlewareFn defines the signature for middleware functions that can intercept and modify
// request processing. Middleware has the same signature as handlers but serves a different
// purpose in the request processing pipeline.
//
// Middleware Return Values:
//   - nil: Continue processing to the next middleware or handler
//   - *HttpResponse: Immediately return this response and stop processing
//
// This allows middleware to:
//   - Validate requests and return errors early
//   - Implement authentication and authorization
//   - Add request/response logging
//   - Modify request state before handler execution
//   - Implement rate limiting and request throttling
//
// Function signature:
//
//	func(req *RouteRequest[RouteState]) *HttpResponse
type MiddlewareFn[RouteState RouteStateCompatible] func(*RouteRequest[RouteState]) *HttpResponse

// Route represents a single node in the routing trie, corresponding to one path component.
// Each route can handle multiple HTTP methods and contain child routes for deeper paths.
// The trie structure enables efficient O(path_length) lookups regardless of total route count.
//
// Trie Structure Example:
//
//	Root
//	├── api (Route)
//	│   ├── users (Route with GET/POST handlers)
//	│   │   └── :id (Route with GET/PUT/DELETE handlers)
//	│   └── posts (Route with GET/POST handlers)
//	└── admin (Route)
//	    └── dashboard (Route with GET handler)
//
// Fields:
//   - PathComponent: The URL segment this route matches (e.g., "users", ":id", "profile")
//   - Handlers: Map of HTTP methods to their corresponding handler and middleware
//   - Children: Child route nodes for deeper path segments
//
// Path Parameters:
// Components starting with ":" are treated as parameters that match any value.
// The matched value becomes available in the request context.
type Route[RouteState RouteStateCompatible] struct {
	PathComponent string
	Handlers      map[HttpMethod]RouteHandler[RouteState] `json:"-"`
	Children      []*Route[RouteState]
}

// RouteHandler combines a handler function with its associated middleware pipeline.
// This structure represents the complete processing chain for a specific HTTP method
// on a specific route, enabling complex request processing workflows.
//
// Processing Order:
//  1. Execute middleware functions in sequence
//  2. If any middleware returns non-nil response, stop and return that response
//  3. If all middleware returns nil, execute the main handler
//  4. Return the handler's response
//
// Fields:
//   - Handler: The main function that processes the request after middleware
//   - Middleware: Slice of functions executed before the handler, in order
type RouteHandler[RouteState RouteStateCompatible] struct {
	Handler    RouteHandlerFn[RouteState]
	Middleware []MiddlewareFn[RouteState]
}

// PrintTree recursively prints this route and all child routes in a hierarchical tree format.
// This method is used to visualize the complete routing structure during application startup
// or for debugging routing configurations.
//
// Output Format:
// The tree uses indentation to show hierarchy, with each route showing its path component
// and the HTTP methods it supports:
//
//	/api [GET]
//	 /users [GET, POST]
//	  /:id [GET, PUT, DELETE]
//	  /profile [GET, PUT]
//
// Parameters:
//   - level: Current indentation level for proper tree formatting (0 for root)
//
// The method automatically handles indentation and formats HTTP methods as a
// comma-separated list for readability.
func (self *Route[RouteState]) PrintTree(level int) {
	methods := make([]string, 0, len(self.Handlers))
	for k := range self.Handlers {
		methods = append(methods, string(k))
	}
	for range level {
		fmt.Print(" ")
	}
	fmt.Printf("/%v [%v]\n", self.PathComponent, strings.Join(methods, ", "))
	for i := range self.Children {
		self.Children[i].PrintTree(level + 1)
	}
}

// NewEmptyRoute creates a new route node with no handlers or children for the specified path component.
// This constructor is used internally by the routing system when building the trie structure.
// The created route is ready to have handlers and child routes added to it.
//
// Parameters:
//   - path: The path component this route will match (e.g., "users", ":id", "admin")
//
// Returns:
//   - Route[RouteState]: Empty route node ready for handler registration
//
// Example usage (typically internal):
//
//	route := pilot.NewEmptyRoute[AppState]("users")
//	// Route can now have handlers added:
//	// route.Handlers[pilot.Get] = RouteHandler{...}
func NewEmptyRoute[RouteState RouteStateCompatible](path string) Route[RouteState] {
	return Route[RouteState]{
		PathComponent: path,
		Handlers:      map[HttpMethod]RouteHandler[RouteState]{},
		Children:      []*Route[RouteState]{},
	}
}
