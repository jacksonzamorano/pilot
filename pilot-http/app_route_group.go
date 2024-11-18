package pilot_http

type RouteGroup[RouteState any] struct {
	Routes []GroupedRoute[RouteState]
}

func NewRouteGroup[RouteState any](routes ...GroupedRoute[RouteState]) *RouteGroup[RouteState] {
	return &RouteGroup[RouteState]{
		Routes: routes,
	}
}

// Functions for Get, post, put, patch, delete, etc.
func (rg *RouteGroup[RouteState]) Get(path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) *GroupedRoute[RouteState] {
	return &GroupedRoute[RouteState]{
		Route:      path,
		Method:     Get,
		Handler:    handler,
		Middleware: middleware,
	}
}
func (rg *RouteGroup[RouteState]) Post(path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) *RouteGroup[RouteState] {
	rg.Routes = append(rg.Routes, GroupedRoute[RouteState]{
		Route:      path,
		Method:     Post,
		Handler:    handler,
		Middleware: middleware,
	})
	return rg
}
func (rg *RouteGroup[RouteState]) Put(path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) *RouteGroup[RouteState] {
	rg.Routes = append(rg.Routes, GroupedRoute[RouteState]{
		Route:      path,
		Method:     Put,
		Handler:    handler,
		Middleware: middleware,
	})
	return rg
}
func (rg *RouteGroup[RouteState]) Patch(path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) *RouteGroup[RouteState] {
	rg.Routes = append(rg.Routes, GroupedRoute[RouteState]{
		Route:      path,
		Method:     Patch,
		Handler:    handler,
		Middleware: middleware,
	})
	return rg
}
func (rg *RouteGroup[RouteState]) Delete(path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) *RouteGroup[RouteState] {
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
