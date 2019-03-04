package web

import (
	"fmt"
	"log"
	"net/http"
)

var ListenPort int

func init() {
	ListenPort = 9090
}

func WebAPIServiceRun() {
	tcpAddrString := fmt.Sprintf(":%d", ListenPort)
	log.Println("Web API Server listens on", tcpAddrString)

	log.Fatal(http.ListenAndServe(tcpAddrString, NewRouter()))
}