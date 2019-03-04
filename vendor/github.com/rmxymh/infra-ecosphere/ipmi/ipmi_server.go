package ipmi

import (
	"fmt"
	"os"
	"net"
	"bytes"
	"log"
	"syscall"
	"io"
	"os/signal"
	"time"
)


import (
	"github.com/rmxymh/infra-ecosphere/bmc"
	"github.com/rmxymh/infra-ecosphere/utils"
)

var running bool = false

func DeserializeAndExecute(buf io.Reader, addr *net.UDPAddr, server *net.UDPConn) {
	RMCPDeserializeAndExecute(buf, addr, server)
}

func IPMIServerHandler(BMCIP string) {
	addr := fmt.Sprintf("%s:623", BMCIP)
	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	utils.CheckError(err)

	server, err := net.ListenUDP("udp", serverAddr)
	utils.CheckError(err)
	defer server.Close()

	buf := make([]byte, 1024)
	for running {
		_, addr, _ := server.ReadFromUDP(buf)
		log.Println("Receive a UDP packet from ", addr.IP.String(), ":", addr.Port)

		bytebuf := bytes.NewBuffer(buf)
		DeserializeAndExecute(bytebuf, addr, server)
	}
}

func IPMIServerServiceRun() {
	signalChan := make(chan os.Signal, 1)
	exitChan := make(chan bool, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT)
	go func() {
		<- signalChan
		log.Println("Capture Interrupt from System, terminate this server.")
		running = false
		exitChan <- true
	}()

	running = true
	for ip, _ := range bmc.BMCs {
		go func(ip string) {
			log.Println("Start BMC Listener for BMC ", ip)
			IPMIServerHandler(ip)
			log.Println("BMC Listener ", ip, " is terminated.")
		}(ip)
	}

	<- exitChan
	log.Println("Wait for Listener terminating...")
	time.Sleep(3 * time.Second)
}
