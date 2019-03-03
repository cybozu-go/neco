package ipmi

import (
	"io"
	"encoding/binary"
	"bytes"
	"net"
	"log"
	"unsafe"
)

// port from OpenIPMI
// Network Functions
const (
	IPMI_NETFN_CHASSIS =		0x00
	IPMI_NETFN_BRIDGE =		0x02
	IPMI_NETFN_SENSOR_EVENT =	0x04
	IPMI_NETFN_APP =		0x06
	IPMI_NETFN_FIRMWARE =		0x08
	IPMI_NETFN_STORAGE  =		0x0a
	IPMI_NETFN_TRANSPORT =		0x0c
	IPMI_NETFN_GROUP_EXTENSION =	0x2c
	IPMI_NETFN_OEM_GROUP =		0x2e

	// Response Bit
	IPMI_NETFN_RESPONSE =		0x01
)

type IPMISessionWrapper struct {
	AuthenticationType uint8
	SequenceNumber uint32
	SessionId uint32
	AuthenticationCode [16]byte
	MessageLen uint8
}

type IPMIMessage struct {
	TargetAddress uint8
	TargetLun uint8			// NetFn (6) + Lun (2)
	Checksum uint8
	SourceAddress uint8
	SourceLun uint8			// SequenceNumber (6) + Lun (2)
	Command uint8
	CompletionCode uint8
	Data []uint8
	DataChecksum uint8
}

func DeserializeIPMI(buf io.Reader)  (length uint32, wrapper IPMISessionWrapper, message IPMIMessage) {
	length = 0
	wrapperLength := uint32(0)

	binary.Read(buf, binary.LittleEndian, &wrapper.AuthenticationType)
	wrapperLength += uint32(unsafe.Sizeof(wrapper.AuthenticationType))
	binary.Read(buf, binary.LittleEndian, &wrapper.SequenceNumber)
	wrapperLength += uint32(unsafe.Sizeof(wrapper.SequenceNumber))
	binary.Read(buf, binary.LittleEndian, &wrapper.SessionId)
	wrapperLength += uint32(unsafe.Sizeof(wrapper.SessionId))
	if wrapper.SessionId != 0x00 {
		binary.Read(buf, binary.LittleEndian, &wrapper.AuthenticationCode)
		wrapperLength += uint32(unsafe.Sizeof(wrapper.AuthenticationCode))
	}
	binary.Read(buf, binary.LittleEndian, &wrapper.MessageLen)
	wrapperLength += uint32(unsafe.Sizeof(wrapper.MessageLen))
	length += wrapperLength

	messageHeaderLength := uint32(0)
	binary.Read(buf, binary.LittleEndian, &message.TargetAddress)
	messageHeaderLength += uint32(unsafe.Sizeof(message.TargetAddress))
	binary.Read(buf, binary.LittleEndian, &message.TargetLun)
	messageHeaderLength += uint32(unsafe.Sizeof(message.TargetLun))
	binary.Read(buf, binary.LittleEndian, &message.Checksum)
	messageHeaderLength += uint32(unsafe.Sizeof(message.Checksum))
	binary.Read(buf, binary.LittleEndian, &message.SourceAddress)
	messageHeaderLength += uint32(unsafe.Sizeof(message.SourceAddress))
	binary.Read(buf, binary.LittleEndian, &message.SourceLun)
	messageHeaderLength += uint32(unsafe.Sizeof(message.SourceLun))
	binary.Read(buf, binary.LittleEndian, &message.Command)
	messageHeaderLength += uint32(unsafe.Sizeof(message.Command))

	dataLen := wrapper.MessageLen - uint8(messageHeaderLength) - 1
	if dataLen > 0 {
		message.Data = make([]uint8, dataLen, dataLen)
		binary.Read(buf, binary.LittleEndian, &message.Data)
	}
	binary.Read(buf, binary.LittleEndian, &message.DataChecksum)
	messageHeaderLength += uint32(unsafe.Sizeof(message.DataChecksum))
	length += uint32(wrapper.MessageLen)

	log.Println("    IPMI Session Wrapper Length = ", wrapperLength)
	log.Println("    IPMI Session Wrapper Message Length = ", wrapper.MessageLen)
	log.Println("    IPMI Message Header Length = ", messageHeaderLength)
	log.Println("    IPMI Message Data Length = ", dataLen)

	return length, wrapper, message
}

func isNetFunctionResponse(targetLun uint8) bool {
	return (((targetLun >> 2) & IPMI_NETFN_RESPONSE ) == IPMI_NETFN_RESPONSE)
}

func SerializeIPMISessionWrapper(buf *bytes.Buffer, wrapper IPMISessionWrapper) {
	binary.Write(buf, binary.LittleEndian, wrapper.AuthenticationType)
	binary.Write(buf, binary.LittleEndian, wrapper.SequenceNumber)
	binary.Write(buf, binary.LittleEndian, wrapper.SessionId)
	if wrapper.SessionId != 0 {
		binary.Write(buf, binary.LittleEndian, wrapper.AuthenticationCode)
	}
	binary.Write(buf, binary.LittleEndian, wrapper.MessageLen)
}

func SerializeIPMIMessage(buf *bytes.Buffer, message IPMIMessage) {
	binary.Write(buf, binary.LittleEndian, message.TargetAddress)
	binary.Write(buf, binary.LittleEndian, message.TargetLun)
	binary.Write(buf, binary.LittleEndian, message.Checksum)
	binary.Write(buf, binary.LittleEndian, message.SourceAddress)
	binary.Write(buf, binary.LittleEndian, message.SourceLun)
	binary.Write(buf, binary.LittleEndian, message.Command)
	if isNetFunctionResponse(message.TargetLun) {
		binary.Write(buf, binary.LittleEndian, message.CompletionCode)
	}
	buf.Write(message.Data)
	binary.Write(buf, binary.LittleEndian, message.DataChecksum)
}

func SerializeIPMI(buf *bytes.Buffer, wrapper IPMISessionWrapper, message IPMIMessage, bmcpass string) {
	// Calculate data checksum
	sum := uint32(0)
	sum += uint32(message.SourceAddress)
	sum += uint32(message.SourceLun)
	sum += uint32(message.Command)
	for i := 0; i < len(message.Data) ; i+=1 {
		sum += uint32(message.Data[i])
	}
	message.DataChecksum = uint8(0x100 - (sum & 0xff))

	// Calculate IPMI Message Checksum
	sum = uint32(message.TargetAddress) + uint32(message.TargetLun)
	message.Checksum = uint8(0x100 - (sum & 0xff))

	// Calculate Message Length
	length := uint32(0)
	length += uint32(unsafe.Sizeof(message.TargetAddress))
	length += uint32(unsafe.Sizeof(message.TargetLun))
	length += uint32(unsafe.Sizeof(message.Checksum))
	length += uint32(unsafe.Sizeof(message.SourceAddress))
	length += uint32(unsafe.Sizeof(message.SourceLun))
	length += uint32(unsafe.Sizeof(message.Command))
	if isNetFunctionResponse(message.TargetLun) {
		length += uint32(unsafe.Sizeof(message.CompletionCode))
	}
	length += uint32(len(message.Data))
	length += uint32(unsafe.Sizeof(message.DataChecksum))
	wrapper.MessageLen = uint8(length)

	if len(bmcpass) > 0 {
		wrapper.AuthenticationCode = GetAuthenticationCode(wrapper.AuthenticationType, bmcpass, wrapper.SessionId, message, wrapper.SequenceNumber)
	}
	// output
	SerializeIPMISessionWrapper(buf, wrapper)
	SerializeIPMIMessage(buf, message)
}

func BuildUpRMCPForIPMI() (rmcp RemoteManagementControlProtocol) {
	rmcp.Version = RMCP_VERSION_1
	rmcp.Reserved = 0x00
	rmcp.Sequence = 0xff
	rmcp.Class = RMCP_CLASS_IPMI

	return rmcp
}

func IPMIDeserializeAndExecute(buf io.Reader, addr *net.UDPAddr, server *net.UDPConn) {
	_, wrapper, message := DeserializeIPMI(buf)

	netFunction := (message.TargetLun & 0xFC) >> 2;

	switch netFunction {
	case IPMI_NETFN_CHASSIS:
		log.Println("    IPMI: NetFunction = CHASSIS")
		IPMI_CHASSIS_DeserializeAndExecute(addr, server, wrapper, message)
	case IPMI_NETFN_BRIDGE:
		log.Println("    IPMI: NetFunction = BRIDGE")
	case IPMI_NETFN_SENSOR_EVENT:
		log.Println("    IPMI: NetFunction = SENSOR / EVENT")
	case IPMI_NETFN_APP:
		log.Println("    IPMI: NetFunction = APP")
		IPMI_APP_DeserializeAndExecute(addr, server, wrapper, message)
	case IPMI_NETFN_FIRMWARE:
		log.Println("    IPMI: NetFunction = FIRMWARE")
	case IPMI_NETFN_STORAGE:
		log.Println("    IPMI: NetFunction = STORAGE")
	case IPMI_NETFN_TRANSPORT:
		log.Println("    IPMI: NetFunction = TRANSPORT")
	case IPMI_NETFN_GROUP_EXTENSION:
		log.Println("    IPMI: NetFunction = GROUP EXTENSION")
		IPMI_GROUPEXT_DeserializeAndExecute(addr, server, wrapper, message)
	case IPMI_NETFN_OEM_GROUP:
		log.Println("    IPMI: NetFunction = OEM GROUP")
	default:
		log.Println("    IPMI: NetFunction = Unknown NetFunction", netFunction)
		log.Println(wrapper)
		log.Println(message)
		wbuf := bytes.Buffer{}
		SerializeIPMI(&wbuf, wrapper, message, "")
		dumpByteBuffer(wbuf)
	}
}