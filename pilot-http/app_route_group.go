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
func GetRoute[RouteState RouteStateCompatible](path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) *GroupedRoute[RouteState] {
	return &GroupedRoute[RouteState]{
		Route:      path,
		Method:     Get,
		Handler:    handler,
		Middleware: middleware,
	}
}
func PostRoute[RouteState RouteStateCompatible](path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) *GroupedRoute[RouteState] {
	return &GroupedRoute[RouteState]{
		Route:      path,
		Method:     Post,
		Handler:    handler,
		Middleware: middleware,
	}
}
func PutRoute[RouteState RouteStateCompatible](path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) *GroupedRoute[RouteState] {
	return &GroupedRoute[RouteState]{
		Route:      path,
		Method:     Put,
		Handler:    handler,
		Middleware: middleware,
	}
}
func PatchRoute[RouteState RouteStateCompatible](path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) *GroupedRoute[RouteState] {
	return &GroupedRoute[RouteState]{
		Route:      path,
		Method:     Patch,
		Handler:    handler,
		Middleware: middleware,
	}
}
func DeleteRoute[RouteState RouteStateCompatible](path string, handler RouteHandlerFn[RouteState], middleware ...MiddlewareFn[RouteState]) *GroupedRoute[RouteState] {
	return &GroupedRoute[RouteState]{
		Route:      path,
		Method:     Delete,
		Handler:    handler,
		Middleware: middleware,
	}
}


type GroupedRoute[RouteState any] struct {
	Route      string
	Method     HttpMethod
	Handler    RouteHandlerFn[RouteState]
	Middleware []MiddlewareFn[RouteState]
}
