package bmc

import (
	"net"
	"log"
	"github.com/rmxymh/infra-ecosphere/vm"
)

type BMC struct {
	Addr net.IP
	VM vm.Instance
}

var BMCs map[string]BMC

func init() {
	log.Println("Initialize BMC Map...")
	BMCs = make(map[string]BMC)
}

func AddBMC(ip net.IP, instance vm.Instance) BMC {
	newBMC := BMC{
		Addr: ip,
		VM: instance,
	}

	BMCs[ip.String()] = newBMC
	log.Println("Add new BMC with IP ", ip.String())

	return newBMC
}

func RemoveBMC(ip net.IP) {
	_, ok := BMCs[ip.String()]

	if ok {
		delete(BMCs, ip.String())
	}
}

func GetBMC(ip net.IP) (BMC, bool) {
	obj, ok := BMCs[ip.String()]

	return obj, ok
}

func (bmc *BMC)Save() {
	if bmc != nil {
		BMCs[bmc.Addr.String()] = *bmc
	}
}

func (bmc *BMC)SetBootDev(dev string) {
	switch dev {
	case vm.BOOT_DEVICE_PXE:
		fallthrough
	case vm.BOOT_DEVICE_DISK:
		fallthrough
	case vm.BOOT_DEVICE_CD_DVD:
		bmc.VM.SetBootDevice(dev)
		bmc.Save()
		log.Println("BMC ", bmc.Addr.String(), " changes its boot device as ", dev)
	case vm.BOOT_DEVICE_FLOPPY:
		log.Println("Device Floppy is not supported.")
	default:
		log.Println("Set Boot Device: ", dev, " is not supported.")
	}

	log.Println(bmc.VM)
}

func (bmc *BMC)PowerOn() {
	log.Println(bmc.VM)
	if ! bmc.VM.IsRunning() {
		bmc.VM.PowerOn()
	}
}

func (bmc *BMC)PowerOff() {
	log.Println(bmc.VM)
	if bmc.VM.IsRunning() {
		bmc.VM.PowerOff()
	}
}

func (bmc *BMC)PowerSoft() {
	if bmc.VM.IsRunning() {
		bmc.VM.ACPIOff()
	}
}

func (bmc *BMC)PowerReset() {
	/* VBox Limitation:
	 *   Because it is not allowed to modify VM properties when VM is running,
  	 *   SetBootDevice does not have any effect in reset. In other words, we
 	 *   need to use this way to simulate power reset to make Set Boot Device
 	 *   working normally.
  	 */

	if bmc.VM.IsRunning() {
		bmc.VM.PowerOff()
		bmc.VM.PowerOn()
	}
}

func (bmc *BMC)IsPowerOn() bool {
	return bmc.VM.IsRunning()
}


