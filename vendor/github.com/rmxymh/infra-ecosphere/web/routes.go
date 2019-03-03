package web

import (
	"net/http"
	"github.com/gorilla/mux"
)

// The idea of routes.go is from "Making a RESTful JSON API in Go"
// Link: http://thenewstack.io/make-a-restful-json-api-go/

type Route struct {
	Name		string
	Method		string
	Pattern		string
	HandlerFunc	http.HandlerFunc
}

var routes = []Route {
	Route {
		"GetAllBMCs",
		"GET",
		"/api/BMCs",
		GetAllBMCs,
	},
	Route {
		"GetBMC",
		"GET",
		"/api/BMCs/{bmcip}",
		GetBMC,
	},
	Route {
		"SetPowerStatus",
		"PUT",
		"/api/BMCs/{bmcip}/power",
		SetPowerStatus,
	},
	Route {
		"SetBootDevice",
		"PUT",
		"/api/BMCs/{bmcip}/bootdev",
		SetBootDevice,
	},
}


// Utility function
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	for _, route := range routes {
		handler := WebLogger(route.HandlerFunc, route.Name)
		router.Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}