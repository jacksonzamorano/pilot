package pilot

type HttpMethod string
type RequestHandler[RouteState any] func(RouteState, *HttpRequest) *HttpResponse

const (
	Get     HttpMethod = "GET"
	Post    HttpMethod = "POST"
	Put     HttpMethod = "PUT"
	Patch   HttpMethod = "PATCH"
	Delete  HttpMethod = "DELETE"
	Options HttpMethod = "OPTIONS"
	None    HttpMethod = "NONE"
)

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
