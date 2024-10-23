package pilot

import (
	"fmt"
	"net"
	"strings"
)

func handleRequest[ApplicationState any, RouteState any](conn net.Conn, app *Application[ApplicationState, RouteState]) {
	defer conn.Close()
	request := ParseRequest(&conn)
	if request == nil {
		return
	}
	if request.Method == Options {
		result := StringResponse("")
		if (*app).Cors != "" {
			result.SetHeader("Access-Control-Allow-Methods", "POST, PATCH, GET, OPTIONS, DELETE, PUT")
			result.SetHeader("Access-Control-Allow-Headers", "*")
			result.SetHeader("Access-Control-Allow-Origin", (*app).Cors)
		}
		result.Write(conn)
		return
	}
	route := (*app).Router.FindPath(request.Path, false)

	if route != nil && route.Handlers[request.Method] != nil {
		data := (*app).MakeRouteData(app.ApplicationState, request)
		for i := range (*app).Middleware {
			result := (*app).Middleware[i](data, request)
			if result != nil {
				result.SetHeader("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, HEAD")
				result.SetHeader("Access-Control-Allow-Headers", "*")
				result.SetHeader("Access-Control-Allow-Origin", (*app).Cors)
				result.Write(conn)
				return
			}
		}
		result := route.Handlers[request.Method](data, request)
		if (*app).CleanRouteData != nil {
			(*app).CleanRouteData(data)
		}
		result.SetHeader("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, HEAD")
		result.SetHeader("Access-Control-Allow-Headers", "*")
		result.SetHeader("Access-Control-Allow-Origin", (*app).Cors)
		result.Write(conn)
	} else {
		result := StringResponse("404 not found")
		result.SetHeader("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, HEAD")
		result.SetHeader("Access-Control-Allow-Headers", "*")
		result.SetHeader("Access-Control-Allow-Origin", (*app).Cors)
		result.SetStatus(StatusNotFound)
		result.Write(conn)
	}
}

type Application[ApplicationState any, RouteState any] struct {
	Port             string
	Router           *RouteCollection[RouteState]
	ApplicationState *ApplicationState
	MakeRouteData    func(*ApplicationState, *HttpRequest) *RouteState
	CleanRouteData   func(*RouteState)
	Cors             string
	Middleware       []func(*RouteState, *HttpRequest) *HttpResponse
}

func NewApplication[ApplicationState any, RouteState any](port string, state *ApplicationState, MakeRouteData func(*ApplicationState, *HttpRequest) *RouteState) *Application[ApplicationState, RouteState] {
	return &Application[ApplicationState, RouteState]{
		Port:             port,
		Router:           NewRouteCollection[RouteState](),
		MakeRouteData:    MakeRouteData,
		ApplicationState: state,
		Middleware:       []func(*RouteState, *HttpRequest) *HttpResponse{},
	}
}
func (a *Application[ApplicationState, RouteState]) SetCleanRouteData(crd func(*RouteState)) {
	a.CleanRouteData = crd
}
func (a *Application[ApplicationState, RouteState]) AddRouteGroup(prefix string, rg *RouteGroup[RouteState]) {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for i := range (*rg).Routes {
		route := (*rg).Routes[i].Route
		if strings.HasPrefix(route, "/") {
			route = route[1:]
		}

		a.Router.AddRoute((*rg).Routes[i].Method, prefix+route, (*rg).Routes[i].Handler)
	}
}

func (a *Application[ApplicationState, RouteState]) AddMiddleware(middlewareFn func(*RouteState, *HttpRequest) *HttpResponse) {
	a.Middleware = append(a.Middleware, middlewareFn)
}

func (a *Application[ApplicationState, RouteState]) Start() {
	fmt.Printf("Starting server on port %v.\n\nRegistered routes:\n", a.Port)
	a.Router.PrintTree()
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", a.Port))
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go handleRequest(conn, a)
	}
}
