package ipmi

import (
	"net"
	"bytes"
	"log"
	"encoding/binary"
	"fmt"
	"math/rand"
	"crypto/md5"
)

import (
	"github.com/htruong/go-md2"
	"github.com/rmxymh/infra-ecosphere/bmc"
)

// port from OpenIPMI
// App Network Function
const (
	IPMI_CMD_GET_DEVICE_ID = 			0x01
	IPMI_CMD_BROADCAST_GET_DEVICE_ID = 		0x01
	IPMI_CMD_COLD_RESET = 				0x02
	IPMI_CMD_WARM_RESET = 				0x03
	IPMI_CMD_GET_SELF_TEST_RESULTS = 		0x04
	IPMI_CMD_MANUFACTURING_TEST_ON = 		0x05
	IPMI_CMD_SET_ACPI_POWER_STATE = 		0x06
	IPMI_CMD_GET_ACPI_POWER_STATE = 		0x07
	IPMI_CMD_GET_DEVICE_GUID = 			0x08
	IPMI_CMD_RESET_WATCHDOG_TIMER = 		0x22
	IPMI_CMD_SET_WATCHDOG_TIMER = 			0x24
	IPMI_CMD_GET_WATCHDOG_TIMER = 			0x25
	IPMI_CMD_SET_BMC_GLOBAL_ENABLES = 		0x2e
	IPMI_CMD_GET_BMC_GLOBAL_ENABLES = 		0x2f
	IPMI_CMD_CLEAR_MSG_FLAGS = 			0x30
	IPMI_CMD_GET_MSG_FLAGS = 			0x31
	IPMI_CMD_ENABLE_MESSAGE_CHANNEL_RCV = 		0x32
	IPMI_CMD_GET_MSG = 				0x33
	IPMI_CMD_SEND_MSG = 				0x34
	IPMI_CMD_READ_EVENT_MSG_BUFFER = 		0x35
	IPMI_CMD_GET_BT_INTERFACE_CAPABILITIES = 	0x36
	IPMI_CMD_GET_SYSTEM_GUID = 			0x37
	IPMI_CMD_GET_CHANNEL_AUTH_CAPABILITIES = 	0x38
	IPMI_CMD_GET_SESSION_CHALLENGE = 		0x39
	IPMI_CMD_ACTIVATE_SESSION = 			0x3a
	IPMI_CMD_SET_SESSION_PRIVILEGE = 		0x3b
	IPMI_CMD_CLOSE_SESSION = 			0x3c
	IPMI_CMD_GET_SESSION_INFO = 			0x3d

	IPMI_CMD_GET_AUTHCODE = 			0x3f
	IPMI_CMD_SET_CHANNEL_ACCESS = 			0x40
	IPMI_CMD_GET_CHANNEL_ACCESS = 			0x41
	IPMI_CMD_GET_CHANNEL_INFO = 			0x42
	IPMI_CMD_SET_USER_ACCESS = 			0x43
	IPMI_CMD_GET_USER_ACCESS = 			0x44
	IPMI_CMD_SET_USER_NAME = 			0x45
	IPMI_CMD_GET_USER_NAME = 			0x46
	IPMI_CMD_SET_USER_PASSWORD = 			0x47
	IPMI_CMD_ACTIVATE_PAYLOAD = 			0x48
	IPMI_CMD_DEACTIVATE_PAYLOAD = 			0x49
	IPMI_CMD_GET_PAYLOAD_ACTIVATION_STATUS = 	0x4a
	IPMI_CMD_GET_PAYLOAD_INSTANCE_INFO = 		0x4b
	IPMI_CMD_SET_USER_PAYLOAD_ACCESS = 		0x4c
	IPMI_CMD_GET_USER_PAYLOAD_ACCESS = 		0x4d
	IPMI_CMD_GET_CHANNEL_PAYLOAD_SUPPORT = 		0x4e
	IPMI_CMD_GET_CHANNEL_PAYLOAD_VERSION = 		0x4f
	IPMI_CMD_GET_CHANNEL_OEM_PAYLOAD_INFO = 	0x50

	IPMI_CMD_MASTER_READ_WRITE = 			0x52

	IPMI_CMD_GET_CHANNEL_CIPHER_SUITES = 		0x54
	IPMI_CMD_SUSPEND_RESUME_PAYLOAD_ENCRYPTION = 	0x55
	IPMI_CMD_SET_CHANNEL_SECURITY_KEY = 		0x56
	IPMI_CMD_GET_SYSTEM_INTERFACE_CAPABILITIES = 	0x57
)

type IPMI_App_Handler func(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage)

type IPMIAppHandlerSet struct {
	GetDeviceIDHandler			IPMI_App_Handler
	BroadcastGetDeviceIDHandler		IPMI_App_Handler
	ColdResetHandler			IPMI_App_Handler
	WarmResetHandler			IPMI_App_Handler
	GetSelfTestResultHandler		IPMI_App_Handler
	ManufacturingTestOnHandler		IPMI_App_Handler
	SetACPIPowerStateHandler		IPMI_App_Handler
	GetACPIPowerStateHandler		IPMI_App_Handler
	GetDeviceGUIDHandler			IPMI_App_Handler
	ResetWatchdogTimerHandler		IPMI_App_Handler
	SetWatchdogTimerHandler			IPMI_App_Handler
	GetWatchdogTimerHandler			IPMI_App_Handler
	SetBMCGlobalEnablesHandler		IPMI_App_Handler
	GetBMCGlobalEnablesHandler		IPMI_App_Handler
	ClearMsgFlagsHandler			IPMI_App_Handler
	GetMsgFlagsHandler			IPMI_App_Handler
	EnableMessageChannelRcvHandler		IPMI_App_Handler
	GetMsgHandler				IPMI_App_Handler
	SendMsgHandler				IPMI_App_Handler
	ReadEventMsgBufferHandler		IPMI_App_Handler
	GetBTInterfaceCapabilitiesHandler	IPMI_App_Handler
	GetSystemGUIDHandler			IPMI_App_Handler
	GetChannelAuthCapabilitiesHandler	IPMI_App_Handler
	GetSessionChallengeHandler		IPMI_App_Handler
	ActivateSessionHandler			IPMI_App_Handler
	SetSessionPrivilegeHandler		IPMI_App_Handler
	CloseSessionHandler			IPMI_App_Handler
	GetSessionInfoHandler			IPMI_App_Handler

	GetAuthCodeHandler			IPMI_App_Handler
	SetChannelAccessHandler			IPMI_App_Handler
	GetChannelAccessHandler			IPMI_App_Handler
	GetChannelInfoHandler			IPMI_App_Handler
	SetUserAccessHandler			IPMI_App_Handler
	GetUserAccessHandler			IPMI_App_Handler
	SetUserNameHandler			IPMI_App_Handler
	GetUserNameHandler			IPMI_App_Handler
	SetUserPasswordHandler			IPMI_App_Handler
	ActivatePayloadHandler			IPMI_App_Handler
	DeactivatePayloadHandler		IPMI_App_Handler
	GetPayloadActivationStatusHandler	IPMI_App_Handler
	GetPayloadInstanceInfoHandler		IPMI_App_Handler
	SetUserPayloadAccessHandler		IPMI_App_Handler
	GetUserPayloadAccessHandler		IPMI_App_Handler
	GetChannelPayloadSupportHandler		IPMI_App_Handler
	GetChannelPayloadVersionHandler		IPMI_App_Handler
	GetChannelOEMPayloadInfoHandler		IPMI_App_Handler

	MasterReadWriteHandler			IPMI_App_Handler

	GetChannelCipherSuiteHandler		IPMI_App_Handler
	SuspendResumePayloadEncryptionHandler	IPMI_App_Handler
	SetChannelSecurityKeyHandler		IPMI_App_Handler
	GetSystemInterfaceCapabilitiesHandler	IPMI_App_Handler
	
	Unsupported				IPMI_App_Handler
}

var IPMIAppHandler IPMIAppHandlerSet = IPMIAppHandlerSet{}

func IPMI_APP_SetHandler(command int, handler IPMI_App_Handler) {
	switch command {
	case IPMI_CMD_GET_DEVICE_ID:
		IPMIAppHandler.GetDeviceIDHandler = handler
		IPMIAppHandler.BroadcastGetDeviceIDHandler = handler
	case IPMI_CMD_COLD_RESET:
		IPMIAppHandler.ColdResetHandler = handler
	case IPMI_CMD_WARM_RESET:
		IPMIAppHandler.WarmResetHandler = handler
	case IPMI_CMD_GET_SELF_TEST_RESULTS:
		IPMIAppHandler.GetSelfTestResultHandler = handler
	case IPMI_CMD_MANUFACTURING_TEST_ON:
		IPMIAppHandler.ManufacturingTestOnHandler = handler
	case IPMI_CMD_SET_ACPI_POWER_STATE:
		IPMIAppHandler.SetACPIPowerStateHandler = handler
	case IPMI_CMD_GET_ACPI_POWER_STATE:
		IPMIAppHandler.GetACPIPowerStateHandler = handler
	case IPMI_CMD_GET_DEVICE_GUID:
		IPMIAppHandler.GetDeviceGUIDHandler = handler
	case IPMI_CMD_RESET_WATCHDOG_TIMER:
		IPMIAppHandler.ResetWatchdogTimerHandler = handler
	case IPMI_CMD_SET_WATCHDOG_TIMER:
		IPMIAppHandler.SetWatchdogTimerHandler = handler
	case IPMI_CMD_GET_WATCHDOG_TIMER:
		IPMIAppHandler.GetWatchdogTimerHandler = handler
	case IPMI_CMD_SET_BMC_GLOBAL_ENABLES:
		IPMIAppHandler.SetBMCGlobalEnablesHandler = handler
	case IPMI_CMD_GET_BMC_GLOBAL_ENABLES:
		IPMIAppHandler.GetBMCGlobalEnablesHandler = handler
	case IPMI_CMD_CLEAR_MSG_FLAGS:
		IPMIAppHandler.ClearMsgFlagsHandler = handler
	case IPMI_CMD_GET_MSG_FLAGS:
		IPMIAppHandler.GetMsgFlagsHandler = handler
	case IPMI_CMD_ENABLE_MESSAGE_CHANNEL_RCV:
		IPMIAppHandler.EnableMessageChannelRcvHandler = handler
	case IPMI_CMD_GET_MSG:
		IPMIAppHandler.GetMsgHandler = handler
	case IPMI_CMD_SEND_MSG:
		IPMIAppHandler.SendMsgHandler = handler
	case IPMI_CMD_READ_EVENT_MSG_BUFFER:
		IPMIAppHandler.ReadEventMsgBufferHandler = handler
	case IPMI_CMD_GET_BT_INTERFACE_CAPABILITIES:
		IPMIAppHandler.GetBTInterfaceCapabilitiesHandler = handler
	case IPMI_CMD_GET_SYSTEM_GUID:
		IPMIAppHandler.GetSystemGUIDHandler = handler
	case IPMI_CMD_GET_CHANNEL_AUTH_CAPABILITIES:
		IPMIAppHandler.GetChannelAuthCapabilitiesHandler = handler
	case IPMI_CMD_GET_SESSION_CHALLENGE:
		IPMIAppHandler.GetSessionChallengeHandler = handler
	case IPMI_CMD_ACTIVATE_SESSION:
		IPMIAppHandler.ActivateSessionHandler = handler
	case IPMI_CMD_SET_SESSION_PRIVILEGE:
		IPMIAppHandler.SetSessionPrivilegeHandler = handler
	case IPMI_CMD_CLOSE_SESSION:
		IPMIAppHandler.CloseSessionHandler = handler
	case IPMI_CMD_GET_SESSION_INFO:
		IPMIAppHandler.GetSessionInfoHandler = handler
	case IPMI_CMD_GET_AUTHCODE:
		IPMIAppHandler.GetAuthCodeHandler = handler
	case IPMI_CMD_SET_CHANNEL_ACCESS:
		IPMIAppHandler.SetChannelAccessHandler = handler
	case IPMI_CMD_GET_CHANNEL_ACCESS:
		IPMIAppHandler.GetChannelAccessHandler = handler
	case IPMI_CMD_GET_CHANNEL_INFO:
		IPMIAppHandler.GetChannelInfoHandler = handler
	case IPMI_CMD_SET_USER_ACCESS:
		IPMIAppHandler.SetUserAccessHandler = handler
	case IPMI_CMD_GET_USER_ACCESS:
		IPMIAppHandler.GetUserAccessHandler = handler
	case IPMI_CMD_SET_USER_NAME:
		IPMIAppHandler.SetUserNameHandler = handler
	case IPMI_CMD_GET_USER_NAME:
		IPMIAppHandler.GetUserNameHandler = handler
	case IPMI_CMD_SET_USER_PASSWORD:
		IPMIAppHandler.SetUserPasswordHandler = handler
	case IPMI_CMD_ACTIVATE_PAYLOAD:
		IPMIAppHandler.ActivatePayloadHandler = handler
	case IPMI_CMD_DEACTIVATE_PAYLOAD:
		IPMIAppHandler.DeactivatePayloadHandler = handler
	case IPMI_CMD_GET_PAYLOAD_ACTIVATION_STATUS:
		IPMIAppHandler.GetPayloadActivationStatusHandler = handler
	case IPMI_CMD_GET_PAYLOAD_INSTANCE_INFO:
		IPMIAppHandler.GetPayloadInstanceInfoHandler = handler
	case IPMI_CMD_SET_USER_PAYLOAD_ACCESS:
		IPMIAppHandler.SetUserPayloadAccessHandler = handler
	case IPMI_CMD_GET_USER_PAYLOAD_ACCESS:
		IPMIAppHandler.GetUserPayloadAccessHandler = handler
	case IPMI_CMD_GET_CHANNEL_PAYLOAD_SUPPORT:
		IPMIAppHandler.GetChannelPayloadSupportHandler = handler
	case IPMI_CMD_GET_CHANNEL_PAYLOAD_VERSION:
		IPMIAppHandler.GetChannelPayloadVersionHandler = handler
	case IPMI_CMD_GET_CHANNEL_OEM_PAYLOAD_INFO:
		IPMIAppHandler.GetChannelOEMPayloadInfoHandler = handler
	case IPMI_CMD_MASTER_READ_WRITE:
		IPMIAppHandler.MasterReadWriteHandler = handler
	case IPMI_CMD_GET_CHANNEL_CIPHER_SUITES:
		IPMIAppHandler.GetChannelCipherSuiteHandler = handler
	case IPMI_CMD_SUSPEND_RESUME_PAYLOAD_ENCRYPTION:
		IPMIAppHandler.SuspendResumePayloadEncryptionHandler = handler
	case IPMI_CMD_SET_CHANNEL_SECURITY_KEY:
		IPMIAppHandler.SetChannelSecurityKeyHandler = handler
	case IPMI_CMD_GET_SYSTEM_INTERFACE_CAPABILITIES:
		IPMIAppHandler.GetSystemInterfaceCapabilitiesHandler = handler
	}
}

func init() {
	IPMIAppHandler.Unsupported = HandleIPMIUnsupportedAppCommand

	IPMI_APP_SetHandler(IPMI_CMD_GET_DEVICE_ID, HandleIPMIGetDeviceID)
	IPMI_APP_SetHandler(IPMI_CMD_BROADCAST_GET_DEVICE_ID, HandleIPMIGetDeviceID)
	IPMI_APP_SetHandler(IPMI_CMD_GET_CHANNEL_AUTH_CAPABILITIES, HandleIPMIAuthenticationCapabilities)
	IPMI_APP_SetHandler(IPMI_CMD_GET_SESSION_CHALLENGE, HandleIPMIGetSessionChallenge)
	IPMI_APP_SetHandler(IPMI_CMD_ACTIVATE_SESSION, HandleIPMIActivateSession)
	IPMI_APP_SetHandler(IPMI_CMD_SET_SESSION_PRIVILEGE, HandleIPMISetSessionPrivilegeLevel)
	IPMI_APP_SetHandler(IPMI_CMD_CLOSE_SESSION, HandleIPMICloseSession)
	
	IPMI_APP_SetHandler(IPMI_CMD_COLD_RESET, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_WARM_RESET, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_SELF_TEST_RESULTS, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_MANUFACTURING_TEST_ON, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SET_ACPI_POWER_STATE, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_ACPI_POWER_STATE, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_DEVICE_GUID, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_RESET_WATCHDOG_TIMER, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SET_WATCHDOG_TIMER, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_WATCHDOG_TIMER, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SET_BMC_GLOBAL_ENABLES, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_BMC_GLOBAL_ENABLES, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_CLEAR_MSG_FLAGS, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_MSG_FLAGS, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_ENABLE_MESSAGE_CHANNEL_RCV, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_MSG, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SEND_MSG, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_READ_EVENT_MSG_BUFFER, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_BT_INTERFACE_CAPABILITIES, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_SYSTEM_GUID, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_SESSION_INFO, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_AUTHCODE, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SET_CHANNEL_ACCESS, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_CHANNEL_ACCESS, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_CHANNEL_INFO, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SET_USER_ACCESS, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_USER_ACCESS, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SET_USER_NAME, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_USER_NAME, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SET_USER_PASSWORD, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_ACTIVATE_PAYLOAD, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_DEACTIVATE_PAYLOAD, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_PAYLOAD_ACTIVATION_STATUS, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_PAYLOAD_INSTANCE_INFO, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SET_USER_PAYLOAD_ACCESS, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_USER_PAYLOAD_ACCESS, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_CHANNEL_PAYLOAD_SUPPORT, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_CHANNEL_PAYLOAD_VERSION, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_CHANNEL_OEM_PAYLOAD_INFO, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_MASTER_READ_WRITE, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_CHANNEL_CIPHER_SUITES, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SUSPEND_RESUME_PAYLOAD_ENCRYPTION, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_SET_CHANNEL_SECURITY_KEY, HandleIPMIUnsupportedAppCommand)
	IPMI_APP_SetHandler(IPMI_CMD_GET_SYSTEM_INTERFACE_CAPABILITIES, HandleIPMIUnsupportedAppCommand)
}


// Default Handler Implementation
const (
	AUTH_NONE =	0x00
	AUTH_MD2 =	0x01
	AUTH_MD5 =	0x02
)

const (
	AUTH_BITMASK_NONE = 		0x01
	AUTH_BITMASK_MD2 = 		0x02
	AUTH_BITMASK_MD5 = 		0x04
	AUTH_BITMASK_STRAIGHT_KEY =	0x10
	AUTH_BITMASK_OEM = 		0x20
	AUTH_BITMASK_IPMI_V2 = 		0x80
)

const (
	AUTH_STATUS_ANONYMOUS =		0x01
	AUTH_STATUS_NULL_USER =		0x02
	AUTH_STATUS_NON_NULL_USER =	0x04
	AUTH_STATUS_USER_LEVEL = 	0x08
	AUTH_STATUS_PER_MESSAGE = 	0x10
	AUTH_STATUS_KG =		0x20
)

const (
	COMPLETION_CODE_OK = 			0x00
	COMPLETION_CODE_INVALID_USERNAME =	0x81
)

func dumpByteBuffer(buf bytes.Buffer) {
	bytebuf := buf.Bytes()
	fmt.Print("[")
	for i := 0 ; i < buf.Len(); i += 1 {
		fmt.Printf(" %02x", bytebuf[i])
	}
	fmt.Println("]")
}

func BuildResponseMessageTemplate(requestWrapper IPMISessionWrapper, requestMessage IPMIMessage,  netfn uint8, command uint8) (IPMISessionWrapper, IPMIMessage) {
	responseMessage := IPMIMessage{}
	responseMessage.TargetAddress = requestMessage.SourceAddress
	remoteLun := requestMessage.SourceLun & 0x03
	localLun := requestMessage.TargetLun & 0x03
	responseMessage.TargetLun = remoteLun | (netfn << 2)
	responseMessage.SourceAddress = requestMessage.TargetAddress
	responseMessage.SourceLun = (requestMessage.SourceLun & 0xfc) | localLun
	responseMessage.Command = command
	responseMessage.CompletionCode = COMPLETION_CODE_OK

	responseWrapper := IPMISessionWrapper{}
	responseWrapper.AuthenticationType = requestWrapper.AuthenticationType
	responseWrapper.SequenceNumber = 0xff
	responseWrapper.SessionId = requestWrapper.SessionId

	return responseWrapper, responseMessage
}

func HandleIPMIUnsupportedAppCommand(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	log.Println("      IPMI App: This command is not supported currently, ignore.")
}

const (
	FAKE_DEVICE_ID =		0xF0
	FAKE_DEVICE_HAS_SDR =		0
	FAKE_DEVICE_REVISION =		0x01
	FAKE_FW_REVISION = 		0x01
	FAKE_FW_MINOR_REVISION =	0x00
	FAKE_IPMI_VERSION =		0x51	// 1.5
)

const (
	ADDITIONAL_DEV_BITMASK_CHASSIS =	 	0x80
	ADDITIONAL_DEV_BITMASK_BRIDGE =			0x40
	ADDITIONAL_DEV_BITMASK_IPMB_EVT_GENERATOR =	0x20
	ADDITIONAL_DEV_BITMASK_IPMB_EVT_RECEIVER =	0x10
	ADDITIONAL_DEV_BITMASK_FRU_INVENTORY = 		0x08
	ADDITIONAL_DEV_BITMASK_SEL = 			0x04
	ADDITIONAL_DEV_BITMASK_SDR_REPOSITORY =		0x02
	ADDITIONAL_DEV_BITMASK_SENSOR =			0x01
)

type IPMIGetDeviceIDResponse struct {
	DeviceID		uint8
	DeviceRevision		uint8
	FirmwareRevision	uint8
	FirmwareMinorRev	uint8
	IPMIVersion		uint8
	AdditionalDevSupport	uint8
	ManufactureID		[3]uint8
	ProductID		uint16
	AuxiliaryFWRevisionInfo	[3]uint8
}

func HandleIPMIGetDeviceID(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	// prepare for response data
	// We don't simulate OEM related behavior
	response := IPMIGetDeviceIDResponse{}
	response.DeviceID = FAKE_DEVICE_ID
	response.DeviceRevision = uint8((FAKE_DEVICE_HAS_SDR << 7) | FAKE_DEVICE_REVISION)
	response.FirmwareRevision = FAKE_FW_REVISION
	response.FirmwareMinorRev = FAKE_FW_MINOR_REVISION
	response.IPMIVersion = FAKE_IPMI_VERSION
	response.AdditionalDevSupport |= (ADDITIONAL_DEV_BITMASK_CHASSIS)

	dataBuf := bytes.Buffer{}
	binary.Write(&dataBuf, binary.LittleEndian, response)

	responseWrapper, responseMessage := BuildResponseMessageTemplate(wrapper, message, (IPMI_NETFN_APP | IPMI_NETFN_RESPONSE), IPMI_CMD_GET_DEVICE_ID)
	responseMessage.Data = dataBuf.Bytes()
	rmcp := BuildUpRMCPForIPMI()

	// serialize and send back
	obuf := bytes.Buffer{}
	SerializeRMCP(&obuf, rmcp)
	SerializeIPMI(&obuf, responseWrapper, responseMessage, "")

	server.WriteToUDP(obuf.Bytes(), addr)
}


type IPMIAuthenticationCapabilitiesRequest struct {
	AutnticationTypeSupport uint8
	RequestedPrivilegeLevel uint8
}

type IPMIAuthenticationCapabilitiesResponse struct {
	Channel uint8
	AuthenticationTypeSupport uint8
	AuthenticationStatus uint8
	ExtCapabilities uint8			// In IPMI v1.5, 0 is always put here. (Reserved)
	OEMID [3]uint8
	OEMAuxiliaryData uint8
}

func HandleIPMIAuthenticationCapabilities(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	buf := bytes.NewBuffer(message.Data)
	request := IPMIAuthenticationCapabilitiesRequest{}
	binary.Read(buf, binary.LittleEndian, &request)

	// prepare for response data
	// We don't simulate OEM related behavior
	response := IPMIAuthenticationCapabilitiesResponse{}
	response.Channel = 1
	response.AuthenticationTypeSupport = AUTH_BITMASK_MD5 | AUTH_BITMASK_MD2 | AUTH_BITMASK_NONE
	response.AuthenticationStatus = AUTH_STATUS_NON_NULL_USER | AUTH_STATUS_NULL_USER
	response.ExtCapabilities = 0
	response.OEMAuxiliaryData = 0

	dataBuf := bytes.Buffer{}
	binary.Write(&dataBuf, binary.LittleEndian, response)

	responseWrapper, responseMessage := BuildResponseMessageTemplate(wrapper, message, (IPMI_NETFN_APP | IPMI_NETFN_RESPONSE), IPMI_CMD_GET_CHANNEL_AUTH_CAPABILITIES)
	responseMessage.Data = dataBuf.Bytes()
	rmcp := BuildUpRMCPForIPMI()

	// serialize and send back
	obuf := bytes.Buffer{}
	SerializeRMCP(&obuf, rmcp)
	SerializeIPMI(&obuf, responseWrapper, responseMessage, "")

	server.WriteToUDP(obuf.Bytes(), addr)
}

type IPMIGetSessionChallengeRequest struct {
	AuthenticationType uint8
	Username [16]byte
}

type IPMIGetSessionChallengeResponse struct {
	TempSessionID uint32
	Challenge [16]byte
}

func HandleIPMIGetSessionChallenge(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	buf := bytes.NewBuffer(message.Data)
	request := IPMIGetSessionChallengeRequest{}
	binary.Read(buf, binary.LittleEndian, &request)

	obuf := bytes.Buffer{}

	nameLength := len(request.Username)
	for i := range request.Username {
		if request.Username[i] == 0 {
			nameLength = i
			break
		}
	}
	username := string(request.Username[:nameLength])

	user, found := bmc.GetBMCUser(username)
	if ! found {
		responseWrapper, responseMessage := BuildResponseMessageTemplate(wrapper, message, (IPMI_NETFN_APP | IPMI_NETFN_RESPONSE), IPMI_CMD_GET_SESSION_CHALLENGE)
		responseMessage.CompletionCode = COMPLETION_CODE_INVALID_USERNAME
		rmcp := BuildUpRMCPForIPMI()

		SerializeRMCP(&obuf, rmcp)
		SerializeIPMI(&obuf, responseWrapper, responseMessage, "")
	} else {
		session := GetNewSession(user)
		var challengeCode [16]uint8

		for i := range challengeCode {
			challengeCode[i] = uint8(rand.Uint32() % 0xff)
		}

		responseChallenge := IPMIGetSessionChallengeResponse{}
		responseChallenge.TempSessionID = session.SessionID
		responseChallenge.Challenge = challengeCode
		dataBuf := bytes.Buffer{}
		binary.Write(&dataBuf, binary.LittleEndian, responseChallenge)

		responseWrapper, responseMessage := BuildResponseMessageTemplate(wrapper, message, (IPMI_NETFN_APP | IPMI_NETFN_RESPONSE), IPMI_CMD_GET_SESSION_CHALLENGE)
		responseMessage.Data = dataBuf.Bytes()
		rmcp := BuildUpRMCPForIPMI()

		SerializeRMCP(&obuf, rmcp)
		SerializeIPMI(&obuf, responseWrapper, responseMessage, user.Password)
	}

	server.WriteToUDP(obuf.Bytes(), addr)
}

type IPMIActivateSessionRequest struct {
	AuthenticationType uint8
	RequestMaxPrivilegeLevel uint8
	Challenge [16]byte
	InitialOutboundSeq uint32
}

type IPMIActivateSessionResponse struct {
	AuthenticationType uint8
	SessionId uint32
	InitialOutboundSeq uint32
	MaxPrivilegeLevel uint8
}

func GetAuthenticationCode(authenticationType uint8, password string, sessionID uint32, message IPMIMessage, sessionSeq uint32) [16]byte {
	var passwordBytes [16]byte
	copy(passwordBytes[:], password)

	context := bytes.Buffer{}
	binary.Write(&context, binary.LittleEndian, passwordBytes)
	binary.Write(&context, binary.LittleEndian, sessionID)
	SerializeIPMIMessage(&context, message)
	binary.Write(&context, binary.LittleEndian, sessionSeq)
	binary.Write(&context, binary.LittleEndian, passwordBytes)

	var code [16]byte
	switch authenticationType {
	case AUTH_MD5:
		code = md5.Sum(context.Bytes())
	case AUTH_MD2:
		hash := md2.New()
		md2Code := hash.Sum(context.Bytes())
		for i := range md2Code {
			if i >= len(code) {
				break
			}
			code[i] = md2Code[i]
		}
	}

	return code
}

func HandleIPMIActivateSession(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	buf := bytes.NewBuffer(message.Data)
	request := IPMIActivateSessionRequest{}
	binary.Read(buf, binary.LittleEndian, &request.AuthenticationType)
	binary.Read(buf, binary.LittleEndian, &request.RequestMaxPrivilegeLevel)
	binary.Read(buf, binary.LittleEndian, &request.Challenge)
	binary.Read(buf, binary.LittleEndian, &request.InitialOutboundSeq)

	binary.Read(buf, binary.LittleEndian, &request)

	//obuf := bytes.Buffer{}

	session, ok := GetSession(wrapper.SessionId)
	if ! ok {
		log.Printf("Unable to find session 0x%08x\n", wrapper.SessionId)
	} else {
		bmcUser := session.User
		code := GetAuthenticationCode(wrapper.AuthenticationType, bmcUser.Password, wrapper.SessionId, message, wrapper.SequenceNumber)
		if bytes.Compare(wrapper.AuthenticationCode[:], code[:]) == 0 {
			log.Println("      IPMI Authentication Pass.")
		} else {
			log.Println("      IPMI Authentication Failed.")
		}

		session.Inc()

		response := IPMIActivateSessionResponse{}
		response.AuthenticationType = request.AuthenticationType
		response.SessionId = wrapper.SessionId
		session.LocalSessionSequenceNumber += 1
		response.InitialOutboundSeq = session.LocalSessionSequenceNumber
		response.MaxPrivilegeLevel = request.RequestMaxPrivilegeLevel

		dataBuf := bytes.Buffer{}
		binary.Write(&dataBuf, binary.LittleEndian, response.AuthenticationType)
		binary.Write(&dataBuf, binary.LittleEndian, response.SessionId)
		binary.Write(&dataBuf, binary.LittleEndian, response.InitialOutboundSeq)
		binary.Write(&dataBuf, binary.LittleEndian, response.MaxPrivilegeLevel)
		//binary.Write(&dataBuf, binary.LittleEndian, response)

		responseWrapper, responseMessage := BuildResponseMessageTemplate(wrapper, message, (IPMI_NETFN_APP | IPMI_NETFN_RESPONSE), IPMI_CMD_ACTIVATE_SESSION)
		responseMessage.Data = dataBuf.Bytes()

		responseWrapper.SessionId = response.SessionId
		responseWrapper.SequenceNumber = session.RemoteSessionSequenceNumber
		rmcp := BuildUpRMCPForIPMI()

		obuf := bytes.Buffer{}
		SerializeRMCP(&obuf, rmcp)
		SerializeIPMI(&obuf, responseWrapper, responseMessage, bmcUser.Password)
		server.WriteToUDP(obuf.Bytes(), addr)
	}
}

type IPMISetSessionPrivilegeLevelRequest struct {
	RequestPrivilegeLevel uint8
}

type IPMISetSessionPrivilegeLevelResponse struct {
	NewPrivilegeLevel uint8
}

func HandleIPMISetSessionPrivilegeLevel(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	buf := bytes.NewBuffer(message.Data)
	request := IPMISetSessionPrivilegeLevelRequest{}
	binary.Read(buf, binary.LittleEndian, &request)

	//obuf := bytes.Buffer{}

	session, ok := GetSession(wrapper.SessionId)
	if ! ok {
		log.Printf("Unable to find session 0x%08x\n", wrapper.SessionId)
	} else {
		bmcUser := session.User
		code := GetAuthenticationCode(wrapper.AuthenticationType, bmcUser.Password, wrapper.SessionId, message, wrapper.SequenceNumber)
		if bytes.Compare(wrapper.AuthenticationCode[:], code[:]) == 0 {
			log.Println("      IPMI Authentication Pass.")
		} else {
			log.Println("      IPMI Authentication Failed.")
		}

		session.Inc()

		response := IPMISetSessionPrivilegeLevelResponse{}
		response.NewPrivilegeLevel = request.RequestPrivilegeLevel

		dataBuf := bytes.Buffer{}
		binary.Write(&dataBuf, binary.LittleEndian, response)

		responseWrapper, responseMessage := BuildResponseMessageTemplate(wrapper, message, (IPMI_NETFN_APP | IPMI_NETFN_RESPONSE), IPMI_CMD_SET_SESSION_PRIVILEGE)
		responseMessage.Data = dataBuf.Bytes()

		responseWrapper.SessionId = wrapper.SessionId
		responseWrapper.SequenceNumber = session.RemoteSessionSequenceNumber
		rmcp := BuildUpRMCPForIPMI()

		obuf := bytes.Buffer{}
		SerializeRMCP(&obuf, rmcp)
		SerializeIPMI(&obuf, responseWrapper, responseMessage, bmcUser.Password)
		server.WriteToUDP(obuf.Bytes(), addr)
	}
}

type IPMICloseSessionRequest struct {
	SessionID uint32
}

func HandleIPMICloseSession(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	buf := bytes.NewBuffer(message.Data)
	request := IPMICloseSessionRequest{}
	binary.Read(buf, binary.LittleEndian, &request)

	//obuf := bytes.Buffer{}

	session, ok := GetSession(wrapper.SessionId)
	if ! ok {
		log.Printf("Unable to find session 0x%08x\n", wrapper.SessionId)
	} else {
		bmcUser := session.User
		code := GetAuthenticationCode(wrapper.AuthenticationType, bmcUser.Password, wrapper.SessionId, message, wrapper.SequenceNumber)
		if bytes.Compare(wrapper.AuthenticationCode[:], code[:]) == 0 {
			log.Println("      IPMI Authentication Pass.")
		} else {
			log.Println("      IPMI Authentication Failed.")
		}

		session.Inc()

		responseWrapper, responseMessage := BuildResponseMessageTemplate(wrapper, message, (IPMI_NETFN_APP | IPMI_NETFN_RESPONSE), IPMI_CMD_CLOSE_SESSION)

		responseWrapper.SessionId = wrapper.SessionId
		responseWrapper.SequenceNumber = session.RemoteSessionSequenceNumber
		rmcp := BuildUpRMCPForIPMI()

		obuf := bytes.Buffer{}
		SerializeRMCP(&obuf, rmcp)
		SerializeIPMI(&obuf, responseWrapper, responseMessage, bmcUser.Password)
		server.WriteToUDP(obuf.Bytes(), addr)
		RemoveSession(request.SessionID)
	}
}

func IPMI_APP_DeserializeAndExecute(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	switch message.Command {
	case IPMI_CMD_GET_DEVICE_ID:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_DEVICE_ID")
		IPMIAppHandler.GetDeviceIDHandler(addr, server, wrapper, message)

	case IPMI_CMD_COLD_RESET:
		log.Println("      IPMI APP: Command = IPMI_CMD_COLD_RESET")
		IPMIAppHandler.ColdResetHandler(addr, server, wrapper, message)

	case IPMI_CMD_WARM_RESET:
		log.Println("      IPMI APP: Command = IPMI_CMD_WARM_RESET")
		IPMIAppHandler.WarmResetHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_SELF_TEST_RESULTS:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_SELF_TEST_RESULTS")
		IPMIAppHandler.GetSelfTestResultHandler(addr, server, wrapper, message)

	case IPMI_CMD_MANUFACTURING_TEST_ON:
		log.Println("      IPMI APP: Command = IPMI_CMD_MANUFACTURING_TEST_ON")
		IPMIAppHandler.ManufacturingTestOnHandler(addr, server, wrapper, message)

	case IPMI_CMD_SET_ACPI_POWER_STATE:
		log.Println("      IPMI APP: Command = IPMI_CMD_SET_ACPI_POWER_STATE")
		IPMIAppHandler.SetACPIPowerStateHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_ACPI_POWER_STATE:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_ACPI_POWER_STATE")
		IPMIAppHandler.GetACPIPowerStateHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_DEVICE_GUID:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_DEVICE_GUID")
		IPMIAppHandler.GetDeviceGUIDHandler(addr, server, wrapper, message)

	case IPMI_CMD_RESET_WATCHDOG_TIMER:
		log.Println("      IPMI APP: Command = IPMI_CMD_RESET_WATCHDOG_TIMER")
		IPMIAppHandler.ResetWatchdogTimerHandler(addr, server, wrapper, message)

	case IPMI_CMD_SET_WATCHDOG_TIMER:
		log.Println("      IPMI APP: Command = IPMI_CMD_SET_WATCHDOG_TIMER")
		IPMIAppHandler.SetWatchdogTimerHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_WATCHDOG_TIMER:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_WATCHDOG_TIMER")
		IPMIAppHandler.GetWatchdogTimerHandler(addr, server, wrapper, message)

	case IPMI_CMD_SET_BMC_GLOBAL_ENABLES:
		log.Println("      IPMI APP: Command = IPMI_CMD_SET_BMC_GLOBAL_ENABLES")
		IPMIAppHandler.SetBMCGlobalEnablesHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_BMC_GLOBAL_ENABLES:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_BMC_GLOBAL_ENABLES")
		IPMIAppHandler.GetBMCGlobalEnablesHandler(addr, server, wrapper, message)

	case IPMI_CMD_CLEAR_MSG_FLAGS:
		log.Println("      IPMI APP: Command =IPMI_CMD_CLEAR_MSG_FLAGS")
		IPMIAppHandler.ClearMsgFlagsHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_MSG_FLAGS:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_MSG_FLAGS")
		IPMIAppHandler.GetMsgFlagsHandler(addr, server, wrapper, message)

	case IPMI_CMD_ENABLE_MESSAGE_CHANNEL_RCV:
		log.Println("      IPMI APP: Command = IPMI_CMD_ENABLE_MESSAGE_CHANNEL_RCV")
		IPMIAppHandler.EnableMessageChannelRcvHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_MSG:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_MSG")
		IPMIAppHandler.GetMsgHandler(addr, server, wrapper, message)

	case IPMI_CMD_SEND_MSG:
		log.Println("      IPMI APP: Command = IPMI_CMD_SEND_MSG")
		IPMIAppHandler.SendMsgHandler(addr, server, wrapper, message)

	case IPMI_CMD_READ_EVENT_MSG_BUFFER:
		log.Println("      IPMI APP: Command = IPMI_CMD_READ_EVENT_MSG_BUFFER")
		IPMIAppHandler.ReadEventMsgBufferHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_BT_INTERFACE_CAPABILITIES:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_BT_INTERFACE_CAPABILITIES")
		IPMIAppHandler.GetBTInterfaceCapabilitiesHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_SYSTEM_GUID:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_SYSTEM_GUID")
		IPMIAppHandler.GetSystemGUIDHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_CHANNEL_AUTH_CAPABILITIES:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_AUTH_CAPABILITIES")
		IPMIAppHandler.GetChannelAuthCapabilitiesHandler(addr, server, wrapper, message)
	case IPMI_CMD_GET_SESSION_CHALLENGE:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_SESSION_CHALLENGE")
		IPMIAppHandler.GetSessionChallengeHandler(addr, server, wrapper, message)
	case IPMI_CMD_ACTIVATE_SESSION:
		log.Println("      IPMI APP: Command = IPMI_CMD_ACTIVATE_SESSION")
		IPMIAppHandler.ActivateSessionHandler(addr, server, wrapper, message)
	case IPMI_CMD_SET_SESSION_PRIVILEGE:
		log.Println("      IPMI APP: Command = IPMI_CMD_SET_SESSION_PRIVILEGE")
		IPMIAppHandler.SetSessionPrivilegeHandler(addr, server, wrapper, message)
	case IPMI_CMD_CLOSE_SESSION:
		log.Println("      IPMI APP: Command = IPMI_CMD_CLOSE_SESSION")
		IPMIAppHandler.CloseSessionHandler(addr, server, wrapper, message)
	case IPMI_CMD_GET_SESSION_INFO:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_SESSION_INFO")
		IPMIAppHandler.GetSessionInfoHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_AUTHCODE:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_AUTHCODE")
		IPMIAppHandler.GetAuthCodeHandler(addr, server, wrapper, message)

	case IPMI_CMD_SET_CHANNEL_ACCESS:
		log.Println("      IPMI APP: Command = IPMI_CMD_SET_CHANNEL_ACCESS")
		IPMIAppHandler.SetChannelAccessHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_CHANNEL_ACCESS:
		log.Println("      IPMI APP: Command =IPMI_CMD_GET_CHANNEL_ACCESS")
		IPMIAppHandler.GetChannelAccessHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_CHANNEL_INFO:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_INFO")
		IPMIAppHandler.GetChannelInfoHandler(addr, server, wrapper, message)

	case IPMI_CMD_SET_USER_ACCESS:
		log.Println("      IPMI APP: Command = IPMI_CMD_SET_USER_ACCESS")
		IPMIAppHandler.SetUserAccessHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_USER_ACCESS:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_USER_ACCESS")
		IPMIAppHandler.GetUserAccessHandler(addr, server, wrapper, message)

	case IPMI_CMD_SET_USER_NAME:
		log.Println("      IPMI APP: Command = IPMI_CMD_SET_USER_NAME")
		IPMIAppHandler.SetUserNameHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_USER_NAME:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_USER_NAME")
		IPMIAppHandler.GetUserNameHandler(addr, server, wrapper, message)

	case IPMI_CMD_SET_USER_PASSWORD:
		log.Println("      IPMI APP: Command = IPMI_CMD_SET_USER_PASSWORD")
		IPMIAppHandler.SetUserPasswordHandler(addr, server, wrapper, message)

	case IPMI_CMD_ACTIVATE_PAYLOAD:
		log.Println("      IPMI APP: Command = IPMI_CMD_ACTIVATE_PAYLOAD")
		IPMIAppHandler.ActivatePayloadHandler(addr, server, wrapper, message)

	case IPMI_CMD_DEACTIVATE_PAYLOAD:
		log.Println("      IPMI APP: Command = IPMI_CMD_DEACTIVATE_PAYLOAD")
		IPMIAppHandler.DeactivatePayloadHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_PAYLOAD_ACTIVATION_STATUS:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_PAYLOAD_ACTIVATION_STATUS")
		IPMIAppHandler.GetPayloadActivationStatusHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_PAYLOAD_INSTANCE_INFO:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_PAYLOAD_INSTANCE_INFO")
		IPMIAppHandler.GetPayloadInstanceInfoHandler(addr, server, wrapper, message)

	case IPMI_CMD_SET_USER_PAYLOAD_ACCESS:
		log.Println("      IPMI APP: Command = IPMI_CMD_SET_USER_PAYLOAD_ACCESS")
		IPMIAppHandler.SetUserPayloadAccessHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_USER_PAYLOAD_ACCESS:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_USER_PAYLOAD_ACCESS")
		IPMIAppHandler.GetUserPayloadAccessHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_CHANNEL_PAYLOAD_SUPPORT:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_PAYLOAD_SUPPORT")
		IPMIAppHandler.GetChannelPayloadSupportHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_CHANNEL_PAYLOAD_VERSION:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_PAYLOAD_VERSION")
		IPMIAppHandler.GetChannelPayloadVersionHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_CHANNEL_OEM_PAYLOAD_INFO:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_OEM_PAYLOAD_INFO")
		IPMIAppHandler.GetChannelOEMPayloadInfoHandler(addr, server, wrapper, message)

	case IPMI_CMD_MASTER_READ_WRITE:
		log.Println("      IPMI APP: Command = IPMI_CMD_MASTER_READ_WRITE")
		IPMIAppHandler.MasterReadWriteHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_CHANNEL_CIPHER_SUITES:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_CIPHER_SUITES")
		IPMIAppHandler.GetChannelCipherSuiteHandler(addr, server, wrapper, message)

	case IPMI_CMD_SUSPEND_RESUME_PAYLOAD_ENCRYPTION:
		log.Println("      IPMI APP: Command = IPMI_CMD_SUSPEND_RESUME_PAYLOAD_ENCRYPTION")
		IPMIAppHandler.SuspendResumePayloadEncryptionHandler(addr, server, wrapper, message)

	case IPMI_CMD_SET_CHANNEL_SECURITY_KEY:
		log.Println("      IPMI APP: Command = IPMI_CMD_SET_CHANNEL_SECURITY_KEY")
		IPMIAppHandler.SetChannelSecurityKeyHandler(addr, server, wrapper, message)

	case IPMI_CMD_GET_SYSTEM_INTERFACE_CAPABILITIES:
		log.Println("      IPMI APP: Command = IPMI_CMD_GET_SYSTEM_INTERFACE_CAPABILITIES")
		IPMIAppHandler.GetSystemInterfaceCapabilitiesHandler(addr, server, wrapper, message)

	}
}