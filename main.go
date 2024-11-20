package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jacksonzamorano/pilot/pilot-http"
)

type RouteState struct {
}
func MakeRouteState(req *pilot_http.HttpRequest) *RouteState {
	return &RouteState{}
}

func Index(req *pilot_http.HttpRequest, db *pgxpool.Conn, routeState *RouteState) *pilot_http.HttpResponse {
	db.QueryRow(context.Background(), "SELECT 1")
	return pilot_http.StringResponse("Hello, world!")
}

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	app := pilot_http.NewApplication("8080", pilot_http.DatabaseConfiguration{
		Host:     "10.0.0.3",
		Port:     "5432",
		Username: "jacksonzamorano",
		Password: "LastBastion080202",
		Database: "podcasts",
	}, MakeRouteState, ctx)
	app.Routes.AddRoute(pilot_http.Get, "/", Index)
	app.Start()
	<-ctx.Done()
}
