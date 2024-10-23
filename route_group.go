package pilot

type RouteGroup[RouteState any] struct {
	Routes []RouteGroupInstance[RouteState]
}
type RouteMapping[RouteState any] map[string]RouteHandler[RouteState]

func NewRouteGroup[RouteState any]() *RouteGroup[RouteState] {
	return &RouteGroup[RouteState]{
		Routes: []RouteGroupInstance[RouteState]{},
	}
}

func (rg *RouteGroup[RouteState]) Get(data RouteMapping[RouteState]) *RouteGroup[RouteState] {
	for key, handler := range data {
		rg.Routes = append(rg.Routes, RouteGroupInstance[RouteState]{
			Route:   key,
			Method:  Get,
			Handler: handler,
		})
	}
	return rg
}
func (rg *RouteGroup[RouteState]) Post(data RouteMapping[RouteState]) *RouteGroup[RouteState] {
	for key, handler := range data {
		rg.Routes = append(rg.Routes, RouteGroupInstance[RouteState]{
			Route:   key,
			Method:  Post,
			Handler: handler,
		})
	}
	return rg
}
func (rg *RouteGroup[RouteState]) Put(data RouteMapping[RouteState]) *RouteGroup[RouteState] {
	for key, handler := range data {
		rg.Routes = append(rg.Routes, RouteGroupInstance[RouteState]{
			Route:   key,
			Method:  Put,
			Handler: handler,
		})
	}
	return rg
}
func (rg *RouteGroup[RouteState]) Patch(data RouteMapping[RouteState]) *RouteGroup[RouteState] {
	for key, handler := range data {
		rg.Routes = append(rg.Routes, RouteGroupInstance[RouteState]{
			Route:   key,
			Method:  Patch,
			Handler: handler,
		})
	}
	return rg
}
func (rg *RouteGroup[RouteState]) Delete(data RouteMapping[RouteState]) *RouteGroup[RouteState] {
	for key, handler := range data {
		rg.Routes = append(rg.Routes, RouteGroupInstance[RouteState]{
			Route:   key,
			Method:  Delete,
			Handler: handler,
		})
	}
	return rg
}

type RouteGroupInstance[RouteState any] struct {
	Route   string
	Method  HttpMethod
	Handler RouteHandler[RouteState]
}
