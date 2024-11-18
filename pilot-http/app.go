package pilot_http

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RouteStateCompatible interface{}

func handleRequest[RouteState any](conn net.Conn, app *Application[RouteState]) {
	db, err := (*app).Database.Acquire(context.Background())
	if err != nil {
		panic(err)
	}
	defer db.Release()
	defer conn.Close()
	request := ParseRequest(&conn)
	if request == nil {
		return
	}

	response := StringResponse("")
	response.Body = []byte("404 not found")
	response.SetStatus(StatusNotFound)
	response.ApplyCors(&app.CorsOrigin, &app.CorsHeaders, &app.CorsMethods)
	if request.Method == Options {
		response.Write(conn)
		return
	}
	route := (*app).Routes.FindPath(request.Path, false)
	if route == nil {
		response.Write(conn)
		return
	}
	handler, found := route.Handlers[request.Method]
	if !found {
		response.Write(conn)
		return
	}

	routeState := (*app.GlobalMiddleware)(request)

	for i := range handler.Middleware {
		response = handler.Middleware[i](routeState, request)
		if response != nil {
			response.ApplyCors(&app.CorsOrigin, &app.CorsHeaders, &app.CorsMethods)
			response.Write(conn)
			return
		}
	}

	response = handler.Handler(request, db, routeState)
	if response == nil {
		response = StringResponse("500 Internal Server Error")
	}
	response.ApplyCors(&app.CorsOrigin, &app.CorsHeaders, &app.CorsMethods)
	response.Write(conn)
}

type Application[RouteState RouteStateCompatible] struct {
	GlobalMiddleware *func(*HttpRequest) *RouteState
	Port             string
	Routes           *RouteCollection[RouteState]
	CorsOrigin       string
	CorsHeaders      string
	CorsMethods      string
	SilentMode       bool
	Database         *pgxpool.Pool
}

func NewApplication[RouteState any](port string, cfg DatabaseConfiguration) *Application[RouteState] {
	pool, err := pgxpool.New(context.Background(), cfg.GetConnectionString())
	if err != nil {
		panic(err)
	}

	return &Application[RouteState]{
		Port:             port,
		CorsOrigin:       "*",
		CorsHeaders:      "*",
		CorsMethods:      "GET, PUT, POST, DELETE, HEAD",
		Routes:           NewRouteCollection[RouteState](),
		SilentMode:       false,
		GlobalMiddleware: nil,
		Database:         pool,
	}
}
func (a *Application[RouteState]) AddRouteGroup(prefix string, rg *RouteGroup[RouteState]) {
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

		a.Routes.AddRouteWithMiddleware((*rg).Routes[i].Method, prefix+route, (*rg).Routes[i].Handler, (*rg).Routes[i].Middleware)
	}
}

func (a *Application[RouteState]) Start() {
	if !a.SilentMode {
		fmt.Printf("Starting server on port %v.\n\nRegistered routes:\n", a.Port)
		a.Routes.PrintTree()
	}
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
