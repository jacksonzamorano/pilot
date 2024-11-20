package pilot_http

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"

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

func NewInlineApplication[RouteState any](port string, cfg DatabaseConfiguration, middlewareFn func(*HttpRequest) *RouteState, ctx context.Context) *Application[RouteState] {
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

func NewApplication[RouteState any](port string, cfg DatabaseConfiguration, middlewareFn func(*HttpRequest) *RouteState) *Application[RouteState] {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
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
	var wg sync.WaitGroup
	queue := make(chan net.Conn, a.WorkerCount*10)
	recvQueue := make(chan net.Conn, a.WorkerCount*10)
	for i := int32(0); i < a.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			handleRequest(queue, a, (*a).Context)
			wg.Done()
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

func handleRequest[RouteState any](conn <-chan net.Conn, app *Application[RouteState], context context.Context) {
	db, err := (*app).Database.Acquire(context)
	if err != nil {
		panic(err)
	}
	ReqLoop: for {
		select {
		case <-context.Done():
			db.Release()
			return
		case conn := <-conn:
			request := ParseRequest(&conn)
			if request == nil {
				continue ReqLoop
			}

			response := StringResponse("")
			response.Body = []byte("404 not found")
			response.SetStatus(StatusNotFound)
			response.ApplyCors(&app.CorsOrigin, &app.CorsHeaders, &app.CorsMethods)
			if request.Method == Options {
				response.Write(conn)
				continue ReqLoop
			}
			route := (*app).Routes.FindPath(request.Path, false)
			if route == nil {
				response.Write(conn)
				continue ReqLoop
			}
			handler, found := route.Handlers[request.Method]
			if !found {
				response.Write(conn)
				continue ReqLoop
			}

			routeState := (app.GlobalMiddleware)(request)

			for i := range handler.Middleware {
				response = handler.Middleware[i](request, db, routeState)
				if response != nil {
					response.ApplyCors(&app.CorsOrigin, &app.CorsHeaders, &app.CorsMethods)
					response.Write(conn)
					continue ReqLoop
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
