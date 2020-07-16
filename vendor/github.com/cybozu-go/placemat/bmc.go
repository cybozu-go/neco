package placemat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/rmxymh/infra-ecosphere/bmc"
	"github.com/rmxymh/infra-ecosphere/ipmi"
	"github.com/rmxymh/infra-ecosphere/utils"
)

const (
	maxBufferSize = 256
)

type bmcServer struct {
	nodeCh   <-chan bmcInfo
	networks []*Network

	muVMs   sync.Mutex
	nodeVMs map[string]*NodeVM // key: serial

	muSerials   sync.Mutex
	nodeSerials map[string]string // key: address

	cert string
	key  string
}

func newBMCServer(vms map[string]*NodeVM, networks []*Network, cert, key string, ch <-chan bmcInfo) *bmcServer {
	s := &bmcServer{
		nodeCh:      ch,
		nodeVMs:     vms,
		nodeSerials: make(map[string]string),
		cert:        cert,
		key:         key,
	}
	for _, n := range networks {
		if n.typ == NetworkBMC {
			s.networks = append(s.networks, n)
		}
	}

	bmc.AddBMCUser("cybozu", "cybozu")

	ipmi.IPMI_CHASSIS_SetHandler(ipmi.IPMI_CMD_GET_CHASSIS_STATUS, s.handleIPMIGetChassisStatus)
	ipmi.IPMI_CHASSIS_SetHandler(ipmi.IPMI_CMD_CHASSIS_CONTROL, s.handleIPMIChassisControl)

	return s
}

func (s *bmcServer) getVMByAddress(addr string) (*NodeVM, error) {
	s.muSerials.Lock()
	serial, ok := s.nodeSerials[addr]
	s.muSerials.Unlock()
	if !ok {
		return nil, errors.New("address not registered: " + addr)
	}

	s.muVMs.Lock()
	vm, ok := s.nodeVMs[serial]
	s.muVMs.Unlock()
	if !ok {
		return nil, errors.New("serial not registered: " + serial)
	}

	return vm, nil
}

// This function is largely copied from github.com/rmxymh/infra-ecosphere,
// which licensed under the MIT License by Yu-Ming Huang.
func (s *bmcServer) handleIPMIGetChassisStatus(addr *net.UDPAddr, server *net.UDPConn, wrapper ipmi.IPMISessionWrapper, message ipmi.IPMIMessage) {
	session, ok := ipmi.GetSession(wrapper.SessionId)
	if !ok {
		fmt.Printf("Unable to find session 0x%08x\n", wrapper.SessionId)
		return
	}

	localIP := utils.GetLocalIP(server)
	vm, err := s.getVMByAddress(localIP)
	if err != nil {
		fmt.Println(err)
		return
	}

	session.Inc()

	response := ipmi.IPMIGetChassisStatusResponse{}
	if vm.IsRunning() {
		response.CurrentPowerState |= ipmi.CHASSIS_POWER_STATE_BITMASK_POWER_ON
	}
	response.LastPowerEvent = 0
	response.MiscChassisState = 0
	response.FrontPanelButtonCapabilities = 0

	dataBuf := bytes.Buffer{}
	binary.Write(&dataBuf, binary.LittleEndian, response)

	responseWrapper, responseMessage := ipmi.BuildResponseMessageTemplate(
		wrapper, message, (ipmi.IPMI_NETFN_CHASSIS | ipmi.IPMI_NETFN_RESPONSE), ipmi.IPMI_CMD_GET_CHASSIS_STATUS)
	responseMessage.Data = dataBuf.Bytes()

	responseWrapper.SessionId = wrapper.SessionId
	responseWrapper.SequenceNumber = session.RemoteSessionSequenceNumber
	rmcp := ipmi.BuildUpRMCPForIPMI()

	obuf := bytes.Buffer{}
	ipmi.SerializeRMCP(&obuf, rmcp)
	ipmi.SerializeIPMI(&obuf, responseWrapper, responseMessage, session.User.Password)
	server.WriteToUDP(obuf.Bytes(), addr)
}

// This function is largely copied from github.com/rmxymh/infra-ecosphere,
// which licensed under the MIT License by Yu-Ming Huang.
func (s *bmcServer) handleIPMIChassisControl(addr *net.UDPAddr, server *net.UDPConn, wrapper ipmi.IPMISessionWrapper, message ipmi.IPMIMessage) {
	buf := bytes.NewBuffer(message.Data)
	request := ipmi.IPMIChassisControlRequest{}
	binary.Read(buf, binary.LittleEndian, &request)

	session, ok := ipmi.GetSession(wrapper.SessionId)
	if !ok {
		fmt.Printf("Unable to find session 0x%08x\n", wrapper.SessionId)
		return
	}

	bmcUser := session.User
	code := ipmi.GetAuthenticationCode(wrapper.AuthenticationType, bmcUser.Password, wrapper.SessionId, message, wrapper.SequenceNumber)
	if bytes.Compare(wrapper.AuthenticationCode[:], code[:]) == 0 {
		fmt.Println("      IPMI Authentication Pass.")
	} else {
		fmt.Println("      IPMI Authentication Failed.")
	}

	localIP := utils.GetLocalIP(server)
	vm, err := s.getVMByAddress(localIP)
	if err != nil {
		fmt.Println(err)
		return
	}

	switch request.ChassisControl {
	case ipmi.CHASSIS_CONTROL_POWER_DOWN:
		vm.PowerOff()
	case ipmi.CHASSIS_CONTROL_POWER_UP:
		vm.PowerOn()
	case ipmi.CHASSIS_CONTROL_POWER_CYCLE:
		vm.PowerOff()
		vm.PowerOn()
	case ipmi.CHASSIS_CONTROL_HARD_RESET:
		vm.PowerOff()
		vm.PowerOn()
	case ipmi.CHASSIS_CONTROL_PULSE:
		// do nothing
	case ipmi.CHASSIS_CONTROL_POWER_SOFT:
		//vm.powerSoft()
	}

	session.Inc()

	responseWrapper, responseMessage := ipmi.BuildResponseMessageTemplate(
		wrapper, message, (ipmi.IPMI_NETFN_CHASSIS | ipmi.IPMI_NETFN_RESPONSE), ipmi.IPMI_CMD_CHASSIS_CONTROL)

	responseWrapper.SessionId = wrapper.SessionId
	responseWrapper.SequenceNumber = session.RemoteSessionSequenceNumber
	rmcp := ipmi.BuildUpRMCPForIPMI()

	obuf := bytes.Buffer{}
	ipmi.SerializeRMCP(&obuf, rmcp)
	ipmi.SerializeIPMI(&obuf, responseWrapper, responseMessage, bmcUser.Password)
	server.WriteToUDP(obuf.Bytes(), addr)
}

func (s *bmcServer) listenIPMI(ctx context.Context, addr string) error {
	serverAddr, err := net.ResolveUDPAddr("udp", addr+":623")
	if err != nil {
		return err
	}

	server, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		server.Close()
	}()

	buf := make([]byte, 1024)
	for {
		_, addr, err := server.ReadFromUDP(buf)
		if err != nil {
			return err
		}

		bytebuf := bytes.NewBuffer(buf)
		ipmi.DeserializeAndExecute(bytebuf, addr, server)
	}
}

func (s *bmcServer) listenHTTPS(ctx context.Context, addr string) error {
	serv := &well.HTTPServer{
		Server: &http.Server{
			Addr:    addr + ":443",
			Handler: myHandler{},
		},
	}

	err := serv.ListenAndServeTLS(s.cert, s.key)
	if err != nil {
		return err
	}
	<-ctx.Done()
	return serv.Close()
}

func (s *bmcServer) handleNode(ctx context.Context) error {
	env := well.NewEnvironment(ctx)

OUTER:
	for {
		select {
		case info := <-s.nodeCh:
			err := s.addPort(ctx, info)
			if err != nil {
				log.Warn("failed to add BMC port", map[string]interface{}{
					log.FnError:   err,
					"serial":      info.serial,
					"bmc_address": info.bmcAddress,
				})
			}
			env.Go(func(ctx context.Context) error {
				return s.listenIPMI(ctx, info.bmcAddress)
			})

			if s.cert != "" && s.key != "" {
				log.Info("start HTTPS server for BMC", map[string]interface{}{
					"cert": s.cert,
					"key":  s.key,
				})
				env.Go(func(ctx context.Context) error {
					return s.listenHTTPS(ctx, info.bmcAddress)
				})
			}
		case <-ctx.Done():
			break OUTER
		}
	}

	env.Cancel(nil)
	return env.Wait()
}

func (s *bmcServer) addPort(ctx context.Context, info bmcInfo) error {
	s.muSerials.Lock()
	s.nodeSerials[info.bmcAddress] = info.serial
	s.muSerials.Unlock()

	br, network, err := s.findBridge(info.bmcAddress)
	if err != nil {
		return err
	}

	prefixLen, _ := network.Mask.Size()
	address := info.bmcAddress + "/" + strconv.Itoa(prefixLen)

	log.Info("creating BMC port", map[string]interface{}{
		"serial":      info.serial,
		"bmc_address": address,
		"bridge":      br,
	})

	c := well.CommandContext(ctx, "ip", "addr", "add", address, "dev", br)
	c.Severity = log.LvDebug
	return c.Run()
}

func (s *bmcServer) findBridge(address string) (string, *net.IPNet, error) {
	ip := net.ParseIP(address)

	for _, n := range s.networks {
		if n.ipNet.Contains(ip) {
			return n.Name, n.ipNet, nil
		}
	}

	return "", nil, errors.New("BMC address not in range of BMC networks: " + address)
}

func (s *bmcServer) registerVM(serial string, vm *NodeVM) {
	s.muVMs.Lock()
	s.nodeVMs[serial] = vm
	s.muVMs.Unlock()
}

// bmcInfo represents BMC information notified by a guest VM.
type bmcInfo struct {
	serial     string
	bmcAddress string
}

type guestConnection struct {
	serial string
	sent   bool
	guest  net.Conn
	ch     chan<- bmcInfo
}

func (g *guestConnection) Handle() {
	bufr := bufio.NewReader(g.guest)
	for {
		line, err := bufr.ReadBytes('\n')
		if err != nil {
			return
		}

		if g.sent {
			continue
		}

		bmcAddress := string(bytes.TrimSpace(line))
		g.ch <- bmcInfo{
			serial:     g.serial,
			bmcAddress: bmcAddress,
		}
		g.sent = true
	}
}

type myHandler struct{}

func (b myHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "Hello I am BMC.")
}
