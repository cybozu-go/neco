package ipmi

import (
	"net"
	"log"
	"bytes"
	"encoding/binary"
	"crypto/md5"
	"github.com/htruong/go-md2"
	"unsafe"
)

const (
	GROUP_EXT_CMD_ATCA_GET_PICMG_PROP =	0x00
)

type IPMI_GroupExt_Handler func(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage)

type IPMIGroupExtHandlerSet struct {
	GetPICMGPropHandler	IPMI_GroupExt_Handler
	UnsupportedHandler	IPMI_GroupExt_Handler
}

var IPMIGroupExtHandler IPMIGroupExtHandlerSet = IPMIGroupExtHandlerSet{}

func IPMI_GROUPEXT_SetHandler(command int, handler IPMI_GroupExt_Handler) {
	switch command {
	case GROUP_EXT_CMD_ATCA_GET_PICMG_PROP:
		IPMIGroupExtHandler.GetPICMGPropHandler = handler
	}
}

func init() {
	IPMIGroupExtHandler.UnsupportedHandler = HandleIPMIUnsupportedGroupExtCommand

	IPMI_GROUPEXT_SetHandler(GROUP_EXT_CMD_ATCA_GET_PICMG_PROP, HandleIPMIGroupExtATCAGetPICMGPropHandler)
}


// Default Handler Implementation
func HandleIPMIUnsupportedGroupExtCommand(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	log.Println("      IPMI GroupExt: This command is not supported currently, ignore.")
}

type IPMIGroupExtGetPICMGPropertiesRequest struct {
	Signature	uint8
}

type PICMGData struct {
	data		uint64
}

func GetAuthenticationCodePICMG(authenticationType uint8, password string, sessionID uint32, message PICMGData, sessionSeq uint32) [16]byte {
	var passwordBytes [16]byte
	copy(passwordBytes[:], password)

	context := bytes.Buffer{}
	binary.Write(&context, binary.LittleEndian, passwordBytes)
	binary.Write(&context, binary.LittleEndian, sessionID)
	binary.Write(&context, binary.LittleEndian, message)
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

func HandleIPMIGroupExtATCAGetPICMGPropHandler(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	buf := bytes.NewBuffer(message.Data)
	request := IPMIChassisControlRequest{}
	binary.Read(buf, binary.LittleEndian, &request)

	session, ok := GetSession(wrapper.SessionId)
	if ! ok {
		log.Printf("Unable to find session 0x%08x\n", wrapper.SessionId)
	} else {
		bmcUser := session.User
		responseMessage := PICMGData{
			data: 0x81b4cb201800c107,
		}

		session.LocalSessionSequenceNumber += 1
		session.RemoteSessionSequenceNumber += 1

		responseWrapper := IPMISessionWrapper{}
		responseWrapper.AuthenticationType = wrapper.AuthenticationType
		responseWrapper.SequenceNumber = 0xff
		responseWrapper.SessionId = wrapper.SessionId
		responseWrapper.SequenceNumber = session.RemoteSessionSequenceNumber
		responseWrapper.AuthenticationCode = GetAuthenticationCodePICMG(wrapper.AuthenticationType, bmcUser.Password, responseWrapper.SessionId, responseMessage, responseWrapper.SequenceNumber)
		responseWrapper.MessageLen = uint8(unsafe.Sizeof(responseMessage))
		rmcp := BuildUpRMCPForIPMI()

		obuf := bytes.Buffer{}
		SerializeRMCP(&obuf, rmcp)
		SerializeIPMISessionWrapper(&obuf, responseWrapper)
		binary.Write(&obuf, binary.LittleEndian, responseMessage)
		server.WriteToUDP(obuf.Bytes(), addr)
	}
}

func IPMI_GROUPEXT_DeserializeAndExecute(addr *net.UDPAddr, server *net.UDPConn, wrapper IPMISessionWrapper, message IPMIMessage) {
	switch message.Command {
	case GROUP_EXT_CMD_ATCA_GET_PICMG_PROP:
		log.Println("      IPMI CHASSIS: Command = GROUP_EXT_CMD_ATCA_GET_PICMG_PROP")
		IPMIGroupExtHandler.GetPICMGPropHandler(addr, server, wrapper, message)
	default:
		IPMIGroupExtHandler.UnsupportedHandler(addr, server, wrapper, message)
	}
}