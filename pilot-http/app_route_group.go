package pilot_http

type RouteGroup[RouteState any] struct {
	Routes []GroupedRoute[RouteState]
}

func NewRouteGroup[RouteState any]() *RouteGroup[RouteState] {
	return &RouteGroup[RouteState]{
		Routes: []GroupedRoute[RouteState]{},
	}
}

func (rg *RouteGroup[RouteState]) Get(path string, middleware []MiddlewareFn[RouteState], handler RouteHandlerFn[RouteState]) *RouteGroup[RouteState] {
	rg.Routes = append(rg.Routes, GroupedRoute[RouteState]{
		Route:      path,
		Method:     Get,
		Handler:    handler,
		Middleware: middleware,
	})
	return rg
}
func (rg *RouteGroup[RouteState]) Post(path string, middleware []MiddlewareFn[RouteState], handler RouteHandlerFn[RouteState]) *RouteGroup[RouteState] {
	rg.Routes = append(rg.Routes, GroupedRoute[RouteState]{
		Route:      path,
		Method:     Post,
		Handler:    handler,
		Middleware: middleware,
	})
	return rg
}
func (rg *RouteGroup[RouteState]) Put(path string, middleware []MiddlewareFn[RouteState], handler RouteHandlerFn[RouteState]) *RouteGroup[RouteState] {
	rg.Routes = append(rg.Routes, GroupedRoute[RouteState]{
		Route:      path,
		Method:     Put,
		Handler:    handler,
		Middleware: middleware,
	})
	return rg
}
func (rg *RouteGroup[RouteState]) Patch(path string, middleware []MiddlewareFn[RouteState], handler RouteHandlerFn[RouteState]) *RouteGroup[RouteState] {
	rg.Routes = append(rg.Routes, GroupedRoute[RouteState]{
		Route:      path,
		Method:     Patch,
		Handler:    handler,
		Middleware: middleware,
	})
	return rg
}
func (rg *RouteGroup[RouteState]) Delete(path string, middleware []MiddlewareFn[RouteState], handler RouteHandlerFn[RouteState]) *RouteGroup[RouteState] {
	rg.Routes = append(rg.Routes, GroupedRoute[RouteState]{
		Route:      path,
		Method:     Delete,
		Handler:    handler,
		Middleware: middleware,
	})
	return rg
}

type GroupedRoute[RouteState any] struct {
	Route      string
	Method     HttpMethod
	Handler    RouteHandlerFn[RouteState]
	Middleware []MiddlewareFn[RouteState]
}
