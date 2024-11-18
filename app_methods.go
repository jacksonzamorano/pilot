package pilot

type HttpMethod string
type RequestHandler[RouteState any] func(RouteState, *HttpRequest) *HttpResponse

const (
	Get     HttpMethod = "GET"
	Post               = "POST"
	Put                = "PUT"
	Patch              = "PATCH"
	Delete             = "DELETE"
	Options            = "OPTIONS"
	None               = "NONE"
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
