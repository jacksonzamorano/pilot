package pilot

import (
	"fmt"
	"strings"
)

type RouteHandler[RouteState any] func(*RouteState, *HttpRequest) *HttpResponse

func (m HttpMethod) String() string {
	return string(m)
}

type RouteCollection[RouteState any] struct {
	Routes []*Route[RouteState]
}

func NewRouteCollection[RouteState any]() *RouteCollection[RouteState] {
	return &RouteCollection[RouteState]{
		Routes: []*Route[RouteState]{},
	}
}

func (self *RouteCollection[RouteState]) PrintTree() {
	for i := range self.Routes {
		self.Routes[i].PrintTree(0)
	}
}

func (self *RouteCollection[RouteState]) FindPath(path string, create bool) *Route[RouteState] {
	comps := PathListFromString(path)
	var node *Route[RouteState] = nil
	for i := range self.Routes {
		if self.Routes[i].PathComponent == comps[0] {
			node = self.Routes[i]
			break
		}
	}
	if node == nil {
		if create {
			newNode := NewEmptyRoute[RouteState](comps[0])
			self.Routes = append(self.Routes, &newNode)
			node = &newNode
		} else {
			return nil
		}
	}
	i := 1
	for i < len(comps) {
		foundAtDepth := false
		for childIdx := range node.Children {
			if node.Children[childIdx].PathComponent == comps[i] {
				node = node.Children[childIdx]
				foundAtDepth = true
				break
			}
		}
		if !foundAtDepth && create {
			newRoute := NewEmptyRoute[RouteState](comps[i])
			node.Children = append(node.Children, &newRoute)
			node = &newRoute
		} else if !foundAtDepth {
			return nil
		}
		i++
	}
	return node
}
func (self *RouteCollection[RouteState]) AddRoute(method HttpMethod, path string, handler RouteHandler[RouteState]) {
	self.FindPath(path, true).Handlers[method] = handler
}

type Route[RouteState any] struct {
	PathComponent string
	Handlers      map[HttpMethod]RouteHandler[RouteState] `json:"-"`
	Children      []*Route[RouteState]
}

func (self *Route[RouteState]) PrintTree(level int) {
	methods := make([]string, 0, len(self.Handlers))
	for k := range self.Handlers {
		methods = append(methods, string(k))
	}
	for i := 0; i < level; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("/%v [%v]\n", self.PathComponent, strings.Join(methods, ", "))
	for i := range self.Children {
		self.Children[i].PrintTree(level + 1)
	}
}

func NewEmptyRoute[RouteState any](path string) Route[RouteState] {
	return Route[RouteState]{
		PathComponent: path,
		Handlers:      map[HttpMethod]RouteHandler[RouteState]{},
		Children:      []*Route[RouteState]{},
	}
}
