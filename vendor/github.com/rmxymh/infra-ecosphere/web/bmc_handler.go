package web

import (
	"net/http"
	"encoding/json"
)

import (
	"github.com/rmxymh/infra-ecosphere/bmc"
	"github.com/gorilla/mux"
	"strings"
	"fmt"
	"net"
	"github.com/rmxymh/infra-ecosphere/vm"
)

type WebRespBMC struct {
	IP		string
	PowerStatus	string
}

type WebRespBMCList struct {
	BMCs	[]WebRespBMC
}

func GetAllBMCs(writer http.ResponseWriter, request *http.Request) {
	RespBMCs := make([]WebRespBMC, 0)
	for _, b := range bmc.BMCs {
		status := "OFF"
		if b.IsPowerOn() {
			status = "ON"
		}
		RespBMCs = append(RespBMCs, WebRespBMC{
					IP: b.Addr.String(),
					PowerStatus: status,
		})
	}

	resp := WebRespBMCList{
		BMCs: RespBMCs,
	}
	json.NewEncoder(writer).Encode(resp)
}

func GetBMC(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	resp := WebRespBMC{}
	resp.IP = vars["bmcip"]

	bmcobj, ok := bmc.GetBMC(net.ParseIP(resp.IP))
	if ! ok {
		resp.PowerStatus = "ERROR: Not found"
	} else {
		status := "OFF"
		if bmcobj.IsPowerOn() {
			status = "ON"
		}

		resp.PowerStatus = status
	}

	json.NewEncoder(writer).Encode(resp)
}

type WebReqPowerOp struct {
	Operation	string
}

type WebRespPowerOp struct {
	IP		string
	Operation	string
	Status		string
}

func SetPowerStatus(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	resp := WebRespPowerOp{}
	resp.IP = vars["bmcip"]

	bmcobj, ok := bmc.GetBMC(net.ParseIP(resp.IP))
	if ! ok {
		resp.Status = fmt.Sprintf("BMC %s does not exist.", resp.IP)
	} else {
		powerOpReq := WebReqPowerOp{}
		err := json.NewDecoder(request.Body).Decode(&powerOpReq)

		if err != nil {
			resp.Operation = "Unknown"
			resp.Status = err.Error()
		} else {
			resp.Operation = strings.ToUpper(powerOpReq.Operation)
			switch resp.Operation {
			case "ON":
				bmcobj.PowerOn()
				resp.Status = "OK"
			case "OFF":
				bmcobj.PowerOff()
				resp.Status = "OK"
			case "SOFT":
				bmcobj.PowerSoft()
				resp.Status = "OK"
			case "RESET":
				bmcobj.PowerReset()
				resp.Status = "OK"
			case "CYCLE":
				bmcobj.PowerReset()
				resp.Status = "OK"
			default:
				resp.Status = fmt.Sprintf("Power Operation %s is not supported.", resp.Operation)
			}
		}
	}

	json.NewEncoder(writer).Encode(resp)
}

type WebReqBootDev struct {
	Device		string
}

type WebRespBootDev struct {
	IP		string
	Device		string
	Status		string
}

func SetBootDevice(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	resp := WebRespBootDev{}
	resp.IP = vars["bmcip"]

	bmcobj, ok := bmc.GetBMC(net.ParseIP(resp.IP))
	if ! ok {
		resp.Status = fmt.Sprintf("BMC %s does not exist.", resp.IP)
	} else {
		bootDevReq := WebReqBootDev{}
		err := json.NewDecoder(request.Body).Decode(&bootDevReq)

		if err != nil {
			resp.Status = "Unknown"
			resp.Status = err.Error()
		} else {
			resp.Device = strings.ToUpper(bootDevReq.Device)
			switch resp.Device {
			case "PXE":
				bmcobj.SetBootDev(vm.BOOT_DEVICE_PXE)
				resp.Status = "OK"
			case "DISK":
				bmcobj.SetBootDev(vm.BOOT_DEVICE_DISK)
				resp.Status = "OK"
			default:
				resp.Status = fmt.Sprintf("Device %s is not supported.", resp.Device)
			}

		}
	}

	json.NewEncoder(writer).Encode(resp)
}