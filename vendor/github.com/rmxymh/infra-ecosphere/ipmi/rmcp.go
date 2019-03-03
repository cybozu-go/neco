package ipmi

import (
	"bytes"
	"encoding/binary"
	"unsafe"
	"io"
	"net"
	"log"
)

type RemoteManagementControlProtocol struct {
	Version uint8
	Reserved uint8
	Sequence uint8
	Class uint8
}

const (
	RMCP_VERSION_1	= 0x06
)

const (
	RMCP_CLASS_ASF	= 0x06
	RMCP_CLASS_IPMI	= 0x07
	RMCP_CLASS_OEM	= 0x08
)

func DeserializeRMCP(buf io.Reader)  (length uint32, header RemoteManagementControlProtocol) {
	binary.Read(buf, binary.LittleEndian, &header)
	length += uint32(unsafe.Sizeof(header))

	return length, header
}

func SerializeRMCP(buf *bytes.Buffer, header RemoteManagementControlProtocol) {
	binary.Write(buf, binary.LittleEndian, header)
}

func RMCPDeserializeAndExecute(buf io.Reader, addr *net.UDPAddr, server *net.UDPConn) {
	_, rmcp := DeserializeRMCP(buf)

	switch rmcp.Class {
	case RMCP_CLASS_ASF:
		log.Println("  RMCP: Class = ASF")
		ASFDeserializeAndExecute(buf, addr, server)
	case RMCP_CLASS_IPMI:
		log.Println("  RMCP: Class = IPMI")
		IPMIDeserializeAndExecute(buf, addr, server)
	case RMCP_CLASS_OEM:
		log.Println("  RMCP: Class = OEM")
	}
}