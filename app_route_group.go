package pilot

// RouteGroup represents a collection of related routes that can be mounted together
// under a common path prefix. This enables modular route organization and helps
// structure large applications with multiple feature areas or API versions.
//
// Route groups provide several organizational benefits:
//   - Logical grouping of related endpoints (e.g., user management, admin functions)
//   - Shared middleware application across multiple routes
//   - API versioning through prefixed mounting (e.g., "/v1", "/v2")
//   - Microservice-style organization within monolithic applications
//   - Easier testing and maintenance of related functionality
//
// Fields:
//   - Routes: Slice of grouped routes with their methods, paths, handlers, and middleware
//
// Example:
//
//	userRoutes := pilot.NewRouteGroup(
//	    pilot.GetRoute("/profile", getUserProfile, authMiddleware),
//	    pilot.PostRoute("/settings", updateSettings, authMiddleware, validationMiddleware),
//	    pilot.DeleteRoute("/account", deleteAccount, authMiddleware, adminMiddleware),
//	)
//	app.AddRouteGroup("/user", userRoutes)
type RouteGroup[RouteState any] struct {
	Routes []GroupedRoute[RouteState]
}

// NewRouteGroup creates a new route group from a variable number of grouped routes.
// This constructor provides a convenient way to define and organize multiple related
// routes that will be mounted together under a common prefix.
//
// Parameters:
//   - routes: Variable number of GroupedRoute instances created with helper functions
//
// Returns:
//   - *RouteGroup[RouteState]: New route group ready to be mounted with AddRouteGroup
//
// Example:
//
//	apiRoutes := pilot.NewRouteGroup(
//	    pilot.GetRoute("/health", healthCheck),
//	    pilot.GetRoute("/version", getVersion),
//	    pilot.PostRoute("/webhook", handleWebhook, validateWebhookMiddleware),
//	)
//	app.AddRouteGroup("/api", apiRoutes)
//	// Creates: /api/health, /api/version, /api/webhook
func NewRouteGroup[RouteState any](routes ...GroupedRoute[RouteState]) *RouteGroup[RouteState] {
	return &RouteGroup[RouteState]{
		Routes: routes,
	}
}

// GetRoute creates a GET route configuration for use in route groups.
// This convenience function creates a GroupedRoute configured for HTTP GET requests,
// which are typically used for data retrieval operations that should be safe and idempotent.
//
// Parameters:
//   - path: Route path relative to the group prefix (e.g., "/profile", "/:id")
//   - handler: Function to handle GET requests to this path
//   - middleware: Optional middleware functions to execute before the handler
//
// Returns:
//   - GroupedRoute[RouteState]: Route configuration for adding to a RouteGroup
//
// Example:
//
//	getUserRoute := pilot.GetRoute("/profile", func(req *pilot.RouteRequest[AppState]) *pilot.HttpResponse {
//	    return pilot.JsonResponse(req.State.User)
//	}, authMiddleware)
func GetRoute[RouteState RouteStateCompatible](path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) GroupedRoute[RouteState] {
	return GroupedRoute[RouteState]{
		Route:      path,
		Method:     Get,
		Handler:    handler,
		Middleware: middleware,
	}
}

// PostRoute creates a POST route configuration for use in route groups.
// This convenience function creates a GroupedRoute configured for HTTP POST requests,
// which are typically used for resource creation or non-idempotent operations.
//
// Parameters:
//   - path: Route path relative to the group prefix (e.g., "/", "/upload")
//   - handler: Function to handle POST requests to this path
//   - middleware: Optional middleware functions to execute before the handler
//
// Returns:
//   - GroupedRoute[RouteState]: Route configuration for adding to a RouteGroup
//
// Example:
//
//	createUserRoute := pilot.PostRoute("/", func(req *pilot.RouteRequest[AppState]) *pilot.HttpResponse {
//	    // Parse and create new user
//	    return pilot.CreatedResponse(newUser)
//	}, authMiddleware, validationMiddleware)
func PostRoute[RouteState RouteStateCompatible](path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) GroupedRoute[RouteState] {
	return GroupedRoute[RouteState]{
		Route:      path,
		Method:     Post,
		Handler:    handler,
		Middleware: middleware,
	}
}

// PutRoute creates a PUT route configuration for use in route groups.
// This convenience function creates a GroupedRoute configured for HTTP PUT requests,
// which are typically used for complete resource replacement or idempotent updates.
//
// Parameters:
//   - path: Route path relative to the group prefix (e.g., "/:id", "/settings")
//   - handler: Function to handle PUT requests to this path
//   - middleware: Optional middleware functions to execute before the handler
//
// Returns:
//   - GroupedRoute[RouteState]: Route configuration for adding to a RouteGroup
//
// Example:
//
//	updateUserRoute := pilot.PutRoute("/:id", func(req *pilot.RouteRequest[AppState]) *pilot.HttpResponse {
//	    userID := req.Request.GetParam("id")
//	    // Update user with complete replacement
//	    return pilot.JsonResponse(updatedUser)
//	}, authMiddleware, ownershipMiddleware)
func PutRoute[RouteState RouteStateCompatible](path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) GroupedRoute[RouteState] {
	return GroupedRoute[RouteState]{
		Route:      path,
		Method:     Put,
		Handler:    handler,
		Middleware: middleware,
	}
}

// PatchRoute creates a PATCH route configuration for use in route groups.
// This convenience function creates a GroupedRoute configured for HTTP PATCH requests,
// which are typically used for partial resource updates or modifications.
//
// Parameters:
//   - path: Route path relative to the group prefix (e.g., "/:id", "/status")
//   - handler: Function to handle PATCH requests to this path
//   - middleware: Optional middleware functions to execute before the handler
//
// Returns:
//   - GroupedRoute[RouteState]: Route configuration for adding to a RouteGroup
//
// Example:
//
//	patchUserRoute := pilot.PatchRoute("/:id", func(req *pilot.RouteRequest[AppState]) *pilot.HttpResponse {
//	    userID := req.Request.GetParam("id")
//	    // Apply partial updates to user
//	    return pilot.JsonResponse(updatedUser)
//	}, authMiddleware, validationMiddleware)
func PatchRoute[RouteState RouteStateCompatible](path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) GroupedRoute[RouteState] {
	return GroupedRoute[RouteState]{
		Route:      path,
		Method:     Patch,
		Handler:    handler,
		Middleware: middleware,
	}
}

// DeleteRoute creates a DELETE route configuration for use in route groups.
// This convenience function creates a GroupedRoute configured for HTTP DELETE requests,
// which are typically used for resource removal operations that should be idempotent.
//
// Parameters:
//   - path: Route path relative to the group prefix (e.g., "/:id", "/all")
//   - handler: Function to handle DELETE requests to this path
//   - middleware: Optional middleware functions to execute before the handler
//
// Returns:
//   - GroupedRoute[RouteState]: Route configuration for adding to a RouteGroup
//
// Example:
//
//	deleteUserRoute := pilot.DeleteRoute("/:id", func(req *pilot.RouteRequest[AppState]) *pilot.HttpResponse {
//	    userID := req.Request.GetParam("id")
//	    // Delete user and return confirmation
//	    return pilot.NoContentResponse()
//	}, authMiddleware, adminMiddleware)
func DeleteRoute[RouteState RouteStateCompatible](path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) GroupedRoute[RouteState] {
	return GroupedRoute[RouteState]{
		Route:      path,
		Method:     Delete,
		Handler:    handler,
		Middleware: middleware,
	}
}

// GroupedRoute represents a single route definition within a route group,
// containing all the information needed to register the route when the group is mounted.
// This struct packages together the route path, HTTP method, handler function,
// and any middleware that should be applied specifically to this route.
//
// GroupedRoute is typically created using the convenience functions like GetRoute,
// PostRoute, etc., rather than being constructed directly. This approach provides
// better type safety and cleaner syntax for route definitions.
//
// Fields:
//   - Route: The path for this route relative to the group prefix (e.g., "/:id", "/settings")
//   - Method: HTTP method this route handles (GET, POST, PUT, PATCH, DELETE)
//   - Handler: Main function that processes requests to this route
//   - Middleware: Slice of middleware functions applied before the handler
//
// When a RouteGroup is mounted with AddRouteGroup, each GroupedRoute is converted
// to a full route registration with the appropriate prefix path and middleware chain.
//
// Example:
//
//	// This GroupedRoute definition:
//	GetRoute("/:id", getUserHandler, authMiddleware)
//
//	// Becomes this when mounted at "/api/users":
//	// GET /api/users/:id with middleware: [authMiddleware] → getUserHandler
type GroupedRoute[RouteState any] struct {
	Route      string
	Method     HttpMethod
	Handler    RouteHandlerFn[RouteState]
	Middleware []MiddlewareFn[RouteState]
}
