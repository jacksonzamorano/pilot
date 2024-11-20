package pilot_http

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RouteStateCompatible interface{}

type Application[RouteState RouteStateCompatible] struct {
	GlobalMiddleware func(*HttpRequest) *RouteState
	Port             string
	Routes           *RouteCollection[RouteState]
	CorsOrigin       string
	CorsHeaders      string
	CorsMethods      string
	SilentMode       bool
	Database         *pgxpool.Pool
	Context          context.Context
	WorkerCount      int32
}

func NewApplication[RouteState any](port string, cfg DatabaseConfiguration, middlewareFn func(*HttpRequest) *RouteState, ctx context.Context) *Application[RouteState] {
	pool, err := pgxpool.New(ctx, cfg.GetConnectionString())
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
		Database:         pool,
		GlobalMiddleware: middlewareFn,
		WorkerCount:      10,
		Context:          ctx,
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
	(*a).Database.Config().MaxConns = a.WorkerCount
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", a.Port))
	if err != nil {
		panic(err)
	}
	queue := make(chan net.Conn, a.WorkerCount*10)
	recvQueue := make(chan net.Conn, a.WorkerCount*10)
	for i := int32(0); i < a.WorkerCount; i++ {
		go handleRequest(queue, a, (*a).Context)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			recvQueue <- conn
		}
	}()
	func() {
		for {
			select {
			case <-(*a).Context.Done():
				log.Println("Stopping server...")
				listener.Close()
				return
			case conn := <-recvQueue:
				queue <- conn
			}
		}
	}()
}

func handleRequest[RouteState any](conn <-chan net.Conn, app *Application[RouteState], context context.Context) {
	db, err := (*app).Database.Acquire(context)
	if err != nil {
		panic(err)
	}
	for {
		select {
		case <-context.Done():
			log.Println("Stopping worker...")
			db.Release()
			return
		case conn := <-conn:
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

			routeState := (app.GlobalMiddleware)(request)

			for i := range handler.Middleware {
				response = handler.Middleware[i](request, db, routeState)
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
	}
}

func DefaultContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
}
