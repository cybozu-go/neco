package ipmi

import (
	"net"
	"bytes"
	"encoding/binary"
	"log"
	"github.com/rmxymh/infra-ecosphere/bmc"
	"github.com/rmxymh/infra-ecosphere/utils"
	"github.com/rmxymh/infra-ecosphere/vm"
)

const (
	BOOT_SET_IN_PROGRESS = 			0
	BOOT_SERVICE_PARTITION_SELECTOR = 	1
	BOOT_SERVICE_PARTITION_SCAN = 		2
	BOOT_BMC_BOOT_FLAG_VALID_BIT_CLEARING =	3
	BOOT_INFO_ACK =				4
	BOOT_FLAG =				5
	BOOT_INITIATOR_INFO =			6
	BOOT_INITIATOR_MAILBOX =		7
)

type IPMI_Chassis_BootOpt_Handler func(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage, selector IPMIChassisBootOptionParameterSelector)

type IPMIChassisSetBootOptHandlerSet struct {
	SetInProgressHandler			IPMI_Chassis_BootOpt_Handler
	ServicePartitionSelectorHandler		IPMI_Chassis_BootOpt_Handler
	ServicePartitionScanHandler		IPMI_Chassis_BootOpt_Handler
	BMCBootFlagValidBitClearingHandler	IPMI_Chassis_BootOpt_Handler
	BootInfoAcknowledgementHandler		IPMI_Chassis_BootOpt_Handler
	BootFlagHandler				IPMI_Chassis_BootOpt_Handler
	BootInitiatorInfoHandler		IPMI_Chassis_BootOpt_Handler
	BootInitiatorMailbox			IPMI_Chassis_BootOpt_Handler
	Unsupported				IPMI_Chassis_BootOpt_Handler
}

type IPMIChassisGetBootOptHandlerSet struct {
	SetInProgressHandler			IPMI_Chassis_BootOpt_Handler
	ServicePartitionSelectorHandler		IPMI_Chassis_BootOpt_Handler
	ServicePartitionScanHandler		IPMI_Chassis_BootOpt_Handler
	BMCBootFlagValidBitClearingHandler	IPMI_Chassis_BootOpt_Handler
	BootInfoAcknowledgementHandler		IPMI_Chassis_BootOpt_Handler
	BootFlagHandler				IPMI_Chassis_BootOpt_Handler
	BootInitiatorInfoHandler		IPMI_Chassis_BootOpt_Handler
	BootInitiatorMailbox			IPMI_Chassis_BootOpt_Handler
	Unsupported				IPMI_Chassis_BootOpt_Handler
}

var IPMIChassisSetBootOptHandler IPMIChassisSetBootOptHandlerSet = IPMIChassisSetBootOptHandlerSet{}
var IPMIChassisGetBootOptHandler IPMIChassisGetBootOptHandlerSet = IPMIChassisGetBootOptHandlerSet{}

func IPMI_CHASSIS_SET_BOOT_OPTION_SetHandler(command int, handler IPMI_Chassis_BootOpt_Handler) {
	switch command {
	case BOOT_SET_IN_PROGRESS:
		IPMIChassisSetBootOptHandler.SetInProgressHandler = handler
	case BOOT_SERVICE_PARTITION_SELECTOR:
		IPMIChassisSetBootOptHandler.ServicePartitionSelectorHandler = handler
	case BOOT_SERVICE_PARTITION_SCAN:
		IPMIChassisSetBootOptHandler.ServicePartitionScanHandler = handler
	case BOOT_BMC_BOOT_FLAG_VALID_BIT_CLEARING:
		IPMIChassisSetBootOptHandler.BMCBootFlagValidBitClearingHandler = handler
	case BOOT_INFO_ACK:
		IPMIChassisSetBootOptHandler.BootInfoAcknowledgementHandler = handler
	case BOOT_FLAG:
		IPMIChassisSetBootOptHandler.BootFlagHandler = handler
	case BOOT_INITIATOR_INFO:
		IPMIChassisSetBootOptHandler.BootInitiatorInfoHandler = handler
	case BOOT_INITIATOR_MAILBOX:
		IPMIChassisSetBootOptHandler.BootInitiatorMailbox = handler
	}
}

func IPMI_CHASSIS_GET_BOOT_OPTION_SetHandler(command int, handler IPMI_Chassis_BootOpt_Handler) {
	switch command {
	case BOOT_SET_IN_PROGRESS:
		IPMIChassisGetBootOptHandler.SetInProgressHandler = handler
	case BOOT_SERVICE_PARTITION_SELECTOR:
		IPMIChassisGetBootOptHandler.ServicePartitionSelectorHandler = handler
	case BOOT_SERVICE_PARTITION_SCAN:
		IPMIChassisGetBootOptHandler.ServicePartitionScanHandler = handler
	case BOOT_BMC_BOOT_FLAG_VALID_BIT_CLEARING:
		IPMIChassisGetBootOptHandler.BMCBootFlagValidBitClearingHandler = handler
	case BOOT_INFO_ACK:
		IPMIChassisGetBootOptHandler.BootInfoAcknowledgementHandler = handler
	case BOOT_FLAG:
		IPMIChassisGetBootOptHandler.BootFlagHandler = handler
	case BOOT_INITIATOR_INFO:
		IPMIChassisGetBootOptHandler.BootInitiatorInfoHandler = handler
	case BOOT_INITIATOR_MAILBOX:
		IPMIChassisGetBootOptHandler.BootInitiatorMailbox = handler
	}
}

func init() {
	IPMIChassisSetBootOptHandler.Unsupported = HandleIPMIChassisBootOptionNotSupport
	IPMIChassisGetBootOptHandler.Unsupported = HandleIPMIChassisBootOptionNotSupport

	IPMI_CHASSIS_SET_BOOT_OPTION_SetHandler(BOOT_SET_IN_PROGRESS, HandleIPMIChassisSetBootOptionSetInProgress)
	IPMI_CHASSIS_SET_BOOT_OPTION_SetHandler(BOOT_INFO_ACK, HandleIPMIChassisSetBootOptionBootInfoAck)
	IPMI_CHASSIS_SET_BOOT_OPTION_SetHandler(BOOT_FLAG, HandleIPMIChassisSetBootOptionBootFlags)
	IPMI_CHASSIS_SET_BOOT_OPTION_SetHandler(BOOT_BMC_BOOT_FLAG_VALID_BIT_CLEARING, HandleIPMIChassisSetBootOptionValidBitClearing)

	IPMI_CHASSIS_SET_BOOT_OPTION_SetHandler(BOOT_SERVICE_PARTITION_SELECTOR, HandleIPMIChassisBootOptionNotSupport)
	IPMI_CHASSIS_SET_BOOT_OPTION_SetHandler(BOOT_SERVICE_PARTITION_SCAN, HandleIPMIChassisBootOptionNotSupport)
	IPMI_CHASSIS_SET_BOOT_OPTION_SetHandler(BOOT_INITIATOR_INFO, HandleIPMIChassisBootOptionNotSupport)
	IPMI_CHASSIS_SET_BOOT_OPTION_SetHandler(BOOT_INITIATOR_MAILBOX, HandleIPMIChassisBootOptionNotSupport)

	IPMI_CHASSIS_GET_BOOT_OPTION_SetHandler(BOOT_FLAG, HandleIPMIChassisGetBootOptionBootFlags)

	IPMI_CHASSIS_GET_BOOT_OPTION_SetHandler(BOOT_SET_IN_PROGRESS, HandleIPMIChassisBootOptionNotSupport)
	IPMI_CHASSIS_GET_BOOT_OPTION_SetHandler(BOOT_INFO_ACK, HandleIPMIChassisBootOptionNotSupport)
	IPMI_CHASSIS_GET_BOOT_OPTION_SetHandler(BOOT_SERVICE_PARTITION_SELECTOR, HandleIPMIChassisBootOptionNotSupport)
	IPMI_CHASSIS_GET_BOOT_OPTION_SetHandler(BOOT_SERVICE_PARTITION_SCAN, HandleIPMIChassisBootOptionNotSupport)
	IPMI_CHASSIS_GET_BOOT_OPTION_SetHandler(BOOT_BMC_BOOT_FLAG_VALID_BIT_CLEARING, HandleIPMIChassisBootOptionNotSupport)
	IPMI_CHASSIS_GET_BOOT_OPTION_SetHandler(BOOT_INITIATOR_INFO, HandleIPMIChassisBootOptionNotSupport)
	IPMI_CHASSIS_GET_BOOT_OPTION_SetHandler(BOOT_INITIATOR_MAILBOX, HandleIPMIChassisBootOptionNotSupport)
}



// Utility
func SendIPMIChassisSetBootOptionResponseBack(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	session, ok := GetSession(wrapper.SessionId)
	if ! ok {
		log.Printf("        IPMI CHASSIS SET BOOT OPTION: Unable to find session 0x%08x\n", wrapper.SessionId)
	} else {
		bmcUser := session.User
		code := GetAuthenticationCode(wrapper.AuthenticationType, bmcUser.Password, wrapper.SessionId, message, wrapper.SequenceNumber)
		if bytes.Compare(wrapper.AuthenticationCode[:], code[:]) == 0 {
			log.Println("        IPMI Authentication Pass.")
		} else {
			log.Println("        IPMI Authentication Failed.")
		}

		session.Inc()

		responseWrapper, responseMessage := BuildResponseMessageTemplate(wrapper, message, (IPMI_NETFN_CHASSIS | IPMI_NETFN_RESPONSE), IPMI_CMD_SET_SYSTEM_BOOT_OPTIONS)

		responseWrapper.SessionId = wrapper.SessionId
		responseWrapper.SequenceNumber = session.RemoteSessionSequenceNumber
		rmcp := BuildUpRMCPForIPMI()

		obuf := bytes.Buffer{}
		SerializeRMCP(&obuf, rmcp)
		SerializeIPMI(&obuf, responseWrapper, responseMessage, bmcUser.Password)
		server.WriteToUDP(obuf.Bytes(), addr)
	}
}

// Default Handler Implementation
func HandleIPMIChassisBootOptionNotSupport(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage, selector IPMIChassisBootOptionParameterSelector) {
	log.Printf("        IPMI BootOption %s is not supported currently.", GetBootOptionParameterSelectorString(int(selector.BootOptionParameterSelector)))
}

const (
	BOOT_SET_IN_PROGRESS_SET_COMPLETE =	0x00
	BOOT_SET_IN_PROGRESS_SET_IN_PROTRESS =	0x01
	BOOT_SET_IN_PROGRESS_COMMIT_WRITE =	0x02
)

type IPMIChassisBootOptionSetInProgressRequest struct {
	SetInProgressParameter	uint8
}

func HandleIPMIChassisSetBootOptionSetInProgress(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage, selector IPMIChassisBootOptionParameterSelector) {
	buf := bytes.NewBuffer(selector.Parameters)
	param := uint8(0)
	binary.Read(buf, binary.LittleEndian, &param)
	request := IPMIChassisBootOptionSetInProgressRequest{}
	request.SetInProgressParameter = param & 0x03

	// Simulate: We just dump log but do nothing here.
	switch request.SetInProgressParameter {
	case BOOT_SET_IN_PROGRESS_SET_COMPLETE:
		log.Println("        IPMI CHASSIS BOOT SET_IN_PROGRESS: BOOT_SET_IN_PROGRESS_SET_COMPLETE")
	case BOOT_SET_IN_PROGRESS_SET_IN_PROTRESS:
		log.Println("        IPMI CHASSIS BOOT SET_IN_PROGRESS: BOOT_SET_IN_PROGRESS_SET_IN_PROTRESS")
	case BOOT_SET_IN_PROGRESS_COMMIT_WRITE:
		log.Println("        IPMI CHASSIS BOOT SET_IN_PROGRESS: BOOT_SET_IN_PROGRESS_COMMIT_WRITE")
	}

	SendIPMIChassisSetBootOptionResponseBack(addr, server, wrapper, message);
}

const (
	BOOT_INFO_ACK_BITMASK_WRITE_MASK_0 = 		0x01
	BOOT_INFO_ACK_BITMASK_WRITE_MASK_1 = 		0x02
	BOOT_INFO_ACK_BITMASK_WRITE_MASK_2 = 		0x04
	BOOT_INFO_ACK_BITMASK_WRITE_MASK_3 = 		0x08
	BOOT_INFO_ACK_BITMASK_WRITE_MASK_4 = 		0x10
	BOOT_INFO_ACK_BITMASK_WRITE_MASK_5 = 		0x20
	BOOT_INFO_ACK_BITMASK_WRITE_MASK_6 = 		0x40
	BOOT_INFO_ACK_BITMASK_WRITE_MASK_7 = 		0x80
)

const (
	BOOT_INFO_ACK_BITMASK_BIOS_POST_HANDLED =	0x01
	BOOT_INFO_ACK_BITMASK_OS_LOADER_HANDLED =	0x02
	BOOT_INFO_ACK_BITMASK_OS_SERVICE_HANDLED =	0x04
	BOOT_INFO_ACK_BITMASK_SMS_HANDLED =		0x08
	BOOT_INFO_ACK_BITMASK_OEM_HANDLED =		0x10
)

type IPMIChassisBootOptionBootInfoReuqest struct {
	WriteMask		uint8
	BootInitiatorAckData	uint8
}

func HandleIPMIChassisSetBootOptionBootInfoAck(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage, selector IPMIChassisBootOptionParameterSelector) {
	buf := bytes.NewBuffer(selector.Parameters)
	request := IPMIChassisBootOptionBootInfoReuqest{}
	binary.Read(buf, binary.LittleEndian, &request)

	// Simulate: We just dump log but do nothing here.
	if request.WriteMask & BOOT_INFO_ACK_BITMASK_WRITE_MASK_0 != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: Enable Write to Bit 0")
	}
	if request.WriteMask & BOOT_INFO_ACK_BITMASK_WRITE_MASK_1 != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: Enable Write to Bit 1")
	}
	if request.WriteMask & BOOT_INFO_ACK_BITMASK_WRITE_MASK_2 != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: Enable Write to Bit 2")
	}
	if request.WriteMask & BOOT_INFO_ACK_BITMASK_WRITE_MASK_3 != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: Enable Write to Bit 3")
	}
	if request.WriteMask & BOOT_INFO_ACK_BITMASK_WRITE_MASK_4 != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: Enable Write to Bit 4")
	}
	if request.WriteMask & BOOT_INFO_ACK_BITMASK_WRITE_MASK_5 != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: Enable Write to Bit 5")
	}
	if request.WriteMask & BOOT_INFO_ACK_BITMASK_WRITE_MASK_6 != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: Enable Write to Bit 6")
	}
	if request.WriteMask & BOOT_INFO_ACK_BITMASK_WRITE_MASK_7 != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: Enable Write to Bit 7")
	}

	// Simulate: We just dump log but do nothing here.
	if request.BootInitiatorAckData & BOOT_INFO_ACK_BITMASK_BIOS_POST_HANDLED != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: BIOS/POST has handled boot info")
	}
	if request.BootInitiatorAckData & BOOT_INFO_ACK_BITMASK_OS_LOADER_HANDLED != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: OS Loader has handled boot info")
	}
	if request.BootInitiatorAckData & BOOT_INFO_ACK_BITMASK_OS_SERVICE_HANDLED != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: OS / service partition has handled boot info")
	}
	if request.BootInitiatorAckData & BOOT_INFO_ACK_BITMASK_SMS_HANDLED != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: SMS has handled boot info")
	}
	if request.BootInitiatorAckData & BOOT_INFO_ACK_BITMASK_OEM_HANDLED != 0 {
		log.Printf("        IPMI CHASSIS BOOT INFO ACK: OEM has handled boot info")
	}

	SendIPMIChassisSetBootOptionResponseBack(addr, server, wrapper, message);
}

// BootParam
const (
	BOOT_PARAM_BITMASK_VALID =		0x80
	BOOT_PARAM_BITMASK_PERSISTENT = 	0x40
	BOOT_PARAM_BITMASK_BOOT_TYPE_EFI =	0x20
)


// BootDevice
const (
	BOOT_DEVICE_BITMASK_CMOS_CLEAR =	0x80
	BOOT_DEVICE_BITMASK_LOCK_KEYBOARD =	0x40
	BOOT_DEVICE_BITMASK_DEVICE =		0x3C
	BOOT_DEVICE_BITMASK_SCREEN_BLANK =	0x02
	BOOT_DEVICE_BITMASK_LOCK_RESET =	0x01
)

const (
	BOOT_DEVICE_FORCE_PXE =			0x01
	BOOT_DEVICE_FORCE_HDD =			0x02
	BOOT_DEVICE_FORCE_HDD_SAFE =		0x03
	BOOT_DEVICE_FORCE_DIAG_PARTITION =	0x04
	BOOT_DEVICE_FORCE_CD =			0x05
	BOOT_DEVICE_FORCE_BIOS =		0x06
	BOOT_DEVICE_FORCE_REMOTE_FLOPPY =	0x07
	BOOT_DEVICE_FORCE_REMOTE_MEDIA =	0x08
	BOOT_DEVICE_FORCE_REMOTE_CD =		0x09
	// 0x0A is reserved
	BOOT_DEVICE_FORCE_REMOTE_HDD =		0x0B
)

// Boot BIOS Verbosity
const (
	BOOT_BIOS_BITMASK_LOCK_VIA_POWER =	0x80
	BOOT_BIOS_BITMASK_FIRMWARE =		0x60
	BOOT_BIOS_BITMASK_EVENT_TRAP = 		0x10
	BOOT_BIOS_BITMASK_PASSWORD_BYPASS = 	0x08
	BOOT_BIOS_BITMASK_LOCK_SLEEP =		0x04
	BOOT_BIOS_BITMASK_CONSOLE_REDIRECT =	0x03
)

const (
	BOOT_BIOS_FIRMWARE_SYSTEM_DEFAULT =	0x00
	BOOT_BIOS_FIRMWARE_REQUEST_QUIET =	0x01
	BOOT_BIOS_FIRMWARE_REQUEST_VERBOSE =	0x02
)

const (
	BOOT_BIOS_CONSOLE_REDIRECT_OCCURS_PER_BIOS_SETTING =	0x00
	BOOT_BIOS_CONSOLE_REDIRECT_SUPRESS_CONSOLE_IF_ENABLED =	0x01
	BOOT_BIOS_CONSOLE_REDIRECT_REQUEST_ENABLED =		0x02
)

// BIOS Shared Mode
const (
	BOOT_BIOS_SHARED_BITMASK_OVERRIDE =			0x04
	BOOT_BIOS_SHARED_BITMASK_MUX_CONTROL_OVERRIDE =		0x03
)

const (
	BOOT_BIOS_SHARED_MUX_RECOMMENDED =	0x00
	BOOT_BIOS_SHARED_MUX_TO_BMC =		0x01
	BOOT_BIOS_SHARED_MUX_TO_SYSTEM =	0x02
)

type IPMIChassisSetBootOptionBootFlags struct {
	BootParam	uint8
	BootDevice	uint8
	BIOSVerbosity	uint8
	BIOSSharedMode	uint8
	Reserved	uint8
}

func HandleIPMIChassisSetBootOptionBootFlags(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage, selector IPMIChassisBootOptionParameterSelector) {
	localIP := utils.GetLocalIP(server)
	bmc, ok := bmc.GetBMC(net.ParseIP(localIP))
	if ! ok {
		log.Println("        IPMI CHASSIS BOOT DEVICE: BMC", localIP, " is not found, skip this request.")
		return
	}

	buf := bytes.NewBuffer(selector.Parameters)
	request := IPMIChassisSetBootOptionBootFlags{}
	binary.Read(buf, binary.LittleEndian, &request)

	// Simulate: We just dump log but do nothing here.
	if request.BootParam & BOOT_PARAM_BITMASK_VALID != 0 {
		log.Println("        IPMI CHASSIS BOOT FLAG: Valid")
	}
	if request.BootParam & BOOT_PARAM_BITMASK_PERSISTENT != 0 {
		log.Println("        IPMI CHASSIS BOOT FLAG: Persistent")
	} else {
		log.Println("        IPMI CHASSIS BOOT FLAG: Only on the next boot")
	}
	if request.BootParam & BOOT_PARAM_BITMASK_BOOT_TYPE_EFI != 0 {
		log.Println("        IPMI CHASSIS BOOT FLAG: Boot Type = EFI")
	} else {
		log.Println("        IPMI CHASSIS BOOT FLAG: Boot Type = PC Compatible (Legacy)")
	}

	// Simulate: We just dump log but do nothing here
	if request.BootDevice & BOOT_DEVICE_BITMASK_CMOS_CLEAR != 0 {
		log.Println("        IPMI CHASSIS BOOT DEVICE: CMOS Clear")
	}
	if request.BootDevice & BOOT_DEVICE_BITMASK_LOCK_KEYBOARD != 0 {
		log.Println("        IPMI CHASSIS BOOT DEVICE: Lock Keyboard")
	}
	if request.BootDevice & BOOT_DEVICE_BITMASK_SCREEN_BLANK != 0 {
		log.Println("        IPMI CHASSIS BOOT DEVICE: Screen Blank")
	}
	if request.BootDevice & BOOT_DEVICE_BITMASK_LOCK_RESET != 0 {
		log.Println("        IPMI CHASSIS BOOT DEVICE: Lock RESET Buttons")
	}

	// This part contains some options that we only support: PXE, CD, HDD
	//   Maybe there is another way to simulate remote device.
	device := (request.BootDevice & BOOT_DEVICE_BITMASK_DEVICE) >> 2
	switch device {
	case BOOT_DEVICE_FORCE_PXE:
		log.Println("        IPMI CHASSIS BOOT DEVICE: BOOT_DEVICE_FORCE_PXE")
		bmc.SetBootDev(vm.BOOT_DEVICE_PXE)
	case BOOT_DEVICE_FORCE_HDD:
		log.Println("        IPMI CHASSIS BOOT DEVICE: BOOT_DEVICE_FORCE_HDD")
		bmc.SetBootDev(vm.BOOT_DEVICE_DISK)
	case BOOT_DEVICE_FORCE_HDD_SAFE:
		log.Println("        IPMI CHASSIS BOOT DEVICE: BOOT_DEVICE_FORCE_HDD_SAFE")
	case BOOT_DEVICE_FORCE_DIAG_PARTITION:
		log.Println("        IPMI CHASSIS BOOT DEVICE: BOOT_DEVICE_FORCE_DIAG_PARTITION")
	case BOOT_DEVICE_FORCE_CD:
		log.Println("        IPMI CHASSIS BOOT DEVICE: BOOT_DEVICE_FORCE_CD")
		bmc.SetBootDev(vm.BOOT_DEVICE_CD_DVD)
	case BOOT_DEVICE_FORCE_BIOS:
		log.Println("        IPMI CHASSIS BOOT DEVICE: BOOT_DEVICE_FORCE_BIOS")
	case BOOT_DEVICE_FORCE_REMOTE_FLOPPY:
		log.Println("        IPMI CHASSIS BOOT DEVICE: BOOT_DEVICE_FORCE_REMOTE_FLOPPY")
	case BOOT_DEVICE_FORCE_REMOTE_MEDIA:
		log.Println("        IPMI CHASSIS BOOT DEVICE: BOOT_DEVICE_FORCE_REMOTE_MEDIA")
	case BOOT_DEVICE_FORCE_REMOTE_CD:
		log.Println("        IPMI CHASSIS BOOT DEVICE: BOOT_DEVICE_FORCE_REMOTE_CD")
	case BOOT_DEVICE_FORCE_REMOTE_HDD:
		log.Println("        IPMI CHASSIS BOOT DEVICE: BOOT_DEVICE_FORCE_REMOTE_HDD")
	}

	// Simulate: We just dump log but do nothing here.
	if request.BIOSVerbosity & BOOT_BIOS_BITMASK_LOCK_VIA_POWER != 0 {
		log.Println("        IPMI CHASSIS BOOT DEVICE: Lock out (power off / sleep request) via Power Button")
	}
	if request.BIOSVerbosity & BOOT_BIOS_BITMASK_EVENT_TRAP != 0 {
		log.Println("        IPMI CHASSIS BOOT DEVICE: Force Progress Event Trap (Only for IPMI 2.0)")
	}
	if request.BIOSVerbosity & BOOT_BIOS_BITMASK_PASSWORD_BYPASS != 0 {
		log.Println("        IPMI CHASSIS BOOT DEVICE: User password bypass")
	}
	if request.BIOSVerbosity & BOOT_BIOS_BITMASK_LOCK_SLEEP != 0 {
		log.Println("        IPMI CHASSIS BOOT DEVICE: Lock out Sleep Button")
	}
	verbosity := (request.BIOSVerbosity & BOOT_BIOS_BITMASK_FIRMWARE) >> 5
	switch verbosity {
	case BOOT_BIOS_FIRMWARE_SYSTEM_DEFAULT:
		log.Println("        IPMI CHASSIS BOOT BIOS: BOOT_BIOS_FIRMWARE_SYSTEM_DEFAULT")
	case BOOT_BIOS_FIRMWARE_REQUEST_QUIET:
		log.Println("        IPMI CHASSIS BOOT BIOS: BOOT_BIOS_FIRMWARE_REQUEST_QUIET")
	case BOOT_BIOS_FIRMWARE_REQUEST_VERBOSE:
		log.Println("        IPMI CHASSIS BOOT BIOS: BOOT_BIOS_FIRMWARE_REQUEST_VERBOSE")
	}
	console_redirect := (request.BIOSVerbosity & BOOT_BIOS_BITMASK_CONSOLE_REDIRECT)
	switch console_redirect {
	case BOOT_BIOS_CONSOLE_REDIRECT_OCCURS_PER_BIOS_SETTING:
		log.Println("        IPMI CHASSIS BOOT BIOS: BOOT_BIOS_CONSOLE_REDIRECT_OCCURS_PER_BIOS_SETTING")
	case BOOT_BIOS_CONSOLE_REDIRECT_SUPRESS_CONSOLE_IF_ENABLED:
		log.Println("        IPMI CHASSIS BOOT BIOS: BOOT_BIOS_CONSOLE_REDIRECT_SUPRESS_CONSOLE_IF_ENABLED")
	case BOOT_BIOS_CONSOLE_REDIRECT_REQUEST_ENABLED:
		log.Println("        IPMI CHASSIS BOOT BIOS: BOOT_BIOS_CONSOLE_REDIRECT_REQUEST_ENABLED")
	}

	// Simulate: We just dump log but do nothing here.
	if request.BIOSSharedMode & BOOT_BIOS_SHARED_BITMASK_OVERRIDE != 0 {
		log.Println("        IPMI CHASSIS BOOT BIOS: BOOT_BIOS_SHARED_BITMASK_OVERRIDE")
	}
	mux_control := request.BIOSSharedMode & BOOT_BIOS_SHARED_BITMASK_MUX_CONTROL_OVERRIDE
	switch mux_control {
	case BOOT_BIOS_SHARED_MUX_RECOMMENDED:
		log.Println("        IPMI CHASSIS BOOT BIOS: BOOT_BIOS_SHARED_MUX_RECOMMENDED")
	case BOOT_BIOS_SHARED_MUX_TO_SYSTEM:
		log.Println("        IPMI CHASSIS BOOT BIOS: BOOT_BIOS_SHARED_MUX_TO_SYSTEM")
	case BOOT_BIOS_SHARED_MUX_TO_BMC:
		log.Println("        IPMI CHASSIS BOOT BIOS: BOOT_BIOS_SHARED_MUX_TO_BMC")
	}

	SendIPMIChassisSetBootOptionResponseBack(addr, server, wrapper, message);
}

const (
	BOOT_FLAG_DONT_CLEAR_BITMASK_RESET_CYCLE_BY_PEF = 				0x10
	BOOT_FLAG_DONT_CLEAR_BITMASK_CHASSIS_CTRL_NOT_RECV_WITHIN_60S_TIMEOUT = 	0x08
	BOOT_FLAG_DONT_CLEAR_BITMASK_RESET_CYCLE_BY_WATCHDOG = 				0x04
	BOOT_FLAG_DONT_CLEAR_BITMASK_PUSHBOTTON_OR_SOFT_RESET =				0x02
	BOOT_FLAG_DONT_CLEAR_BITMASK_POWER_UP_VIA_PUSHBUTTON = 				0x01
)

func HandleIPMIChassisSetBootOptionValidBitClearing(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage, selector IPMIChassisBootOptionParameterSelector) {
	localIP := utils.GetLocalIP(server)
	_, ok := bmc.GetBMC(net.ParseIP(localIP))
	if ! ok {
		log.Println("        IPMI CHASSIS BOOT DEVICE: BMC", localIP, " is not found, skip this request.")
		return
	}

	validBitDontClearOn := selector.Parameters[0]

	// Simulate: We just dump log but do nothing here.
	if validBitDontClearOn & BOOT_FLAG_DONT_CLEAR_BITMASK_RESET_CYCLE_BY_PEF != 0 {
		log.Println("        IPMI CHASSIS BOOT FLAG Don't Clear On: Power Reset / Cycle caused by PEF")
	}
	if validBitDontClearOn & BOOT_FLAG_DONT_CLEAR_BITMASK_CHASSIS_CTRL_NOT_RECV_WITHIN_60S_TIMEOUT != 0 {
		log.Println("        IPMI CHASSIS BOOT FLAG Don't Clear On: Chassis Control command not received within 60s timeout")
	}
	if validBitDontClearOn & BOOT_FLAG_DONT_CLEAR_BITMASK_RESET_CYCLE_BY_WATCHDOG != 0 {
		log.Println("        IPMI CHASSIS BOOT FLAG Don't Clear On: Power Reset / Cycle caused by Watchdog")
	}
	if validBitDontClearOn & BOOT_FLAG_DONT_CLEAR_BITMASK_PUSHBOTTON_OR_SOFT_RESET != 0 {
		log.Println("        IPMI CHASSIS BOOT FLAG Don't Clear On: push button reset or soft reset")
	}
	if validBitDontClearOn & BOOT_FLAG_DONT_CLEAR_BITMASK_POWER_UP_VIA_PUSHBUTTON != 0 {
		log.Println("        IPMI CHASSIS BOOT FLAG Don't Clear On: Power up via pushbutton or wake event")
	}

	SendIPMIChassisSetBootOptionResponseBack(addr, server, wrapper, message);
}

type IPMIChassisBootOptionParameterSelector struct {
	Validity			bool
	BootOptionParameterSelector	uint8
	Parameters			[]uint8
}

func GetBootOptionParameterSelectorString(selector int) (string) {
	switch selector {
	case BOOT_SET_IN_PROGRESS:
		return "BOOT_SET_IN_PROGRESS"
	case BOOT_SERVICE_PARTITION_SELECTOR:
		return "BOOT_SERVICE_PARTITION_SELECTOR"
	case BOOT_SERVICE_PARTITION_SCAN:
		return "BOOT_SERVICE_PARTITION_SCAN"
	case BOOT_BMC_BOOT_FLAG_VALID_BIT_CLEARING:
		return "BOOT_BMC_BOOT_FLAG_VALID_BIT_CLEARING"
	case BOOT_INFO_ACK:
		return "BOOT_INFO_ACK"
	case BOOT_FLAG:
		return "BOOT_FLAG"
	case BOOT_INITIATOR_INFO:
		return "BOOT_INITIATOR_INFO"
	case BOOT_INITIATOR_MAILBOX:
		return "BOOT_INITIATOR_MAILBOX"
	}
	return "UNKNOWN"
}

func IPMI_CHASSIS_SetBootOption_DeserializeAndExecute(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	buf := bytes.NewBuffer(message.Data)
	request := IPMIChassisBootOptionParameterSelector{}
	selector := uint8(0x00)
	binary.Read(buf, binary.LittleEndian, &selector)

	request.Validity = ((selector & 0x80) >> 7 != 0)
	request.BootOptionParameterSelector = selector & 0x7f
	request.Parameters = message.Data[1:]

	switch request.BootOptionParameterSelector {
	case BOOT_SET_IN_PROGRESS:
		IPMIChassisSetBootOptHandler.SetInProgressHandler(addr, server, wrapper, message, request)
	case BOOT_SERVICE_PARTITION_SELECTOR:
		IPMIChassisSetBootOptHandler.ServicePartitionSelectorHandler(addr, server, wrapper, message, request)
	case BOOT_SERVICE_PARTITION_SCAN:
		IPMIChassisSetBootOptHandler.ServicePartitionScanHandler(addr, server, wrapper, message, request)
	case BOOT_BMC_BOOT_FLAG_VALID_BIT_CLEARING:
		IPMIChassisSetBootOptHandler.BMCBootFlagValidBitClearingHandler(addr, server, wrapper, message, request)
	case BOOT_INFO_ACK:
		IPMIChassisSetBootOptHandler.BootInfoAcknowledgementHandler(addr, server, wrapper, message, request)
	case BOOT_FLAG:
		IPMIChassisSetBootOptHandler.BootFlagHandler(addr, server, wrapper, message, request)
	case BOOT_INITIATOR_INFO:
		IPMIChassisSetBootOptHandler.BootInitiatorInfoHandler(addr, server, wrapper, message, request)
	case BOOT_INITIATOR_MAILBOX:
		IPMIChassisSetBootOptHandler.BootInitiatorMailbox(addr, server, wrapper, message, request)
	}
}

type IPMIChassisGetBootOptionBootFlags struct {
	ParamVersion	uint8
	BootOptSelector	uint8
	BootParam	uint8
	BootDevice	uint8
	BIOSVerbosity	uint8
	BIOSSharedMode	uint8
	Reserved	uint8
}

func HandleIPMIChassisGetBootOptionBootFlags(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage, selector IPMIChassisBootOptionParameterSelector) {
	session, ok := GetSession(wrapper.SessionId)
	if ! ok {
		log.Printf("        IPMI CHASSIS SET BOOT OPTION: Unable to find session 0x%08x\n", wrapper.SessionId)
	} else {
		bmcUser := session.User
		code := GetAuthenticationCode(wrapper.AuthenticationType, bmcUser.Password, wrapper.SessionId, message, wrapper.SequenceNumber)
		if bytes.Compare(wrapper.AuthenticationCode[:], code[:]) == 0 {
			log.Println("        IPMI Authentication Pass.")
		} else {
			log.Println("        IPMI Authentication Failed.")
		}

		session.Inc()

		data := IPMIChassisGetBootOptionBootFlags{}
		data.ParamVersion = 0x01
		data.BootOptSelector = BOOT_FLAG
		data.BootParam = 0
		data.BootDevice = 0
		data.BIOSVerbosity = 0
		data.BIOSSharedMode = 0
		data.Reserved = 0

		dbuf := bytes.Buffer{}
		binary.Write(&dbuf, binary.LittleEndian, data)

		responseWrapper, responseMessage := BuildResponseMessageTemplate(wrapper, message, (IPMI_NETFN_CHASSIS | IPMI_NETFN_RESPONSE), IPMI_CMD_SET_SYSTEM_BOOT_OPTIONS)
		responseMessage.Data = dbuf.Bytes()

		responseWrapper.SessionId = wrapper.SessionId
		responseWrapper.SequenceNumber = session.RemoteSessionSequenceNumber
		rmcp := BuildUpRMCPForIPMI()

		obuf := bytes.Buffer{}
		SerializeRMCP(&obuf, rmcp)
		SerializeIPMI(&obuf, responseWrapper, responseMessage, bmcUser.Password)
		server.WriteToUDP(obuf.Bytes(), addr)
	}
}

func IPMI_CHASSIS_GetBootOption_DeserializeAndExecute(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	buf := bytes.NewBuffer(message.Data)
	request := IPMIChassisBootOptionParameterSelector{}
	selector := uint8(0x00)
	binary.Read(buf, binary.LittleEndian, &selector)

	request.Validity = ((selector & 0x80) >> 7 != 0)
	request.BootOptionParameterSelector = selector & 0x7f
	request.Parameters = message.Data[1:]

	switch request.BootOptionParameterSelector {
	case BOOT_SET_IN_PROGRESS:
		IPMIChassisGetBootOptHandler.SetInProgressHandler(addr, server, wrapper, message, request)
	case BOOT_SERVICE_PARTITION_SELECTOR:
		IPMIChassisGetBootOptHandler.ServicePartitionSelectorHandler(addr, server, wrapper, message, request)
	case BOOT_SERVICE_PARTITION_SCAN:
		IPMIChassisGetBootOptHandler.ServicePartitionScanHandler(addr, server, wrapper, message, request)
	case BOOT_BMC_BOOT_FLAG_VALID_BIT_CLEARING:
		IPMIChassisGetBootOptHandler.BMCBootFlagValidBitClearingHandler(addr, server, wrapper, message, request)
	case BOOT_INFO_ACK:
		IPMIChassisGetBootOptHandler.BootInfoAcknowledgementHandler(addr, server, wrapper, message, request)
	case BOOT_FLAG:
		IPMIChassisGetBootOptHandler.BootFlagHandler(addr, server, wrapper, message, request)
	case BOOT_INITIATOR_INFO:
		IPMIChassisGetBootOptHandler.BootInitiatorInfoHandler(addr, server, wrapper, message, request)
	case BOOT_INITIATOR_MAILBOX:
		IPMIChassisGetBootOptHandler.BootInitiatorMailbox(addr, server, wrapper, message, request)
	}
}
