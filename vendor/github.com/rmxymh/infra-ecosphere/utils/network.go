package utils

import (
	"net"
	"strings"
	"log"
	"os"
)

func GetLocalIP(server *net.UDPConn) (ip string) {
	endpoint := server.LocalAddr().String()

	ip = endpoint
	delimiter_pos := strings.Index(endpoint, ":")
	if delimiter_pos > 0 {
		ip = endpoint[:delimiter_pos]
	}
	return ip
}

func CheckError(err error) {
	if err != nil {
		log.Println("Error: ", err)
		os.Exit(-1)
	}
}
