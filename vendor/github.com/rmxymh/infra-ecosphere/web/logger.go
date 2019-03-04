package web

import (
	"log"
	"net/http"
	"time"
)

func WebLogger(handler http.HandlerFunc, name string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		start := time.Now()

		handler.ServeHTTP(writer, request)

		log.Printf("%s\t%s\t%s\t%s", request.Method, request.RequestURI, name, time.Since(start))
	})
}
