package pilot

// HttpMethod represents the HTTP request method/verb used for routing and handler dispatch.
// The framework supports all standard HTTP methods with type safety to prevent routing errors.
type HttpMethod string

// RequestHandler defines the legacy function signature for route handlers.
// This type is deprecated in favor of RouteHandlerFn which provides better integration
// with the middleware system and database access patterns.
//
// Deprecated: Use RouteHandlerFn[RouteState] instead for new applications.
type RequestHandler[RouteState any] func(RouteState, *HttpRequest) *HttpResponse

// HTTP method constants representing all supported request verbs.
// These constants provide type safety and prevent string-based routing errors.
//
// Supported methods:
//   - Get: Retrieve data, should be idempotent and safe
//   - Post: Create new resources, non-idempotent
//   - Put: Update/replace entire resources, idempotent
//   - Patch: Partial resource updates, may or may not be idempotent
//   - Delete: Remove resources, idempotent
//   - Options: CORS preflight and resource introspection, handled automatically
//   - None: Internal placeholder, not used for actual routing
const (
	Get     HttpMethod = "GET"
	Post    HttpMethod = "POST"
	Put     HttpMethod = "PUT"
	Patch   HttpMethod = "PATCH"
	Delete  HttpMethod = "DELETE"
	Options HttpMethod = "OPTIONS"
	None    HttpMethod = "NONE"
)

// HttpMethods provides string-to-HttpMethod mapping for request parsing.
// This map is used internally by the request parser to convert incoming
// HTTP method strings into strongly-typed HttpMethod values, enabling
// type-safe routing and handler dispatch.
//
// The map includes all standard HTTP methods and an internal "NONE" value
// used as a placeholder for invalid or unrecognized methods.
var (
	HttpMethods = map[string]HttpMethod{
		"GET":     Get,
		"POST":    Post,
		"PUT":     Put,
		"PATCH":   Patch,
		"DELETE":  Delete,
		"OPTIONS": Options,
		"NONE":    None,
	}
)
