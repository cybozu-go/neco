package ipmi

import (
	"encoding/binary"
	"unsafe"
	"bytes"
	"io"
	"net"
	"log"
)

type AlertStandardFormat struct {
	IANA uint32
	MessageType uint8
	MessageTag uint8
	Reserved uint8
	DataLen uint8
	Data []uint8
}

const (
	ASF_RMCP_IANA	= 0x000011be
)

const (
	ASF_TYPE_PING	= 0x80
	ASF_TYPE_PONG	= 0x40
)

func DeserializeASF(buf io.Reader)  (length uint32, header AlertStandardFormat) {
	length = 0

	binary.Read(buf, binary.LittleEndian, &header.IANA)
	length += uint32(unsafe.Sizeof(header.IANA))
	binary.Read(buf, binary.LittleEndian, &header.MessageType)
	length += uint32(unsafe.Sizeof(header.MessageType))
	binary.Read(buf, binary.LittleEndian, &header.MessageTag)
	length += uint32(unsafe.Sizeof(header.MessageTag))
	binary.Read(buf, binary.LittleEndian, &header.Reserved)
	length += uint32(unsafe.Sizeof(header.Reserved))
	binary.Read(buf, binary.LittleEndian, &header.DataLen)
	length += uint32(unsafe.Sizeof(header.DataLen))

	header.Data = make([]uint8, header.DataLen, header.DataLen)
	binary.Read(buf, binary.LittleEndian, &header.Data)
	length += uint32(header.DataLen)

	return length, header
}

func SerializeASF(buf *bytes.Buffer, header AlertStandardFormat) {
	binary.Write(buf, binary.LittleEndian, header.IANA)
	binary.Write(buf, binary.LittleEndian, header.MessageType)
	binary.Write(buf, binary.LittleEndian, header.MessageTag)
	binary.Write(buf, binary.LittleEndian, header.Reserved)
	binary.Write(buf, binary.LittleEndian, header.DataLen)
	buf.Write(header.Data)
}


// Handlers
func HandleASFPong(pingHeader AlertStandardFormat, addr *net.UDPAddr, server *net.UDPConn) {
	rmcp := RemoteManagementControlProtocol{
		Version: 1,
		Reserved: 0,
		Sequence: 0xff,
		Class: RMCP_CLASS_ASF,
	}

	asf := AlertStandardFormat{
		IANA: ASF_RMCP_IANA,
		MessageType: ASF_TYPE_PONG,
		MessageTag: pingHeader.MessageTag,
		DataLen: 0x10,
		Data: make([]uint8, 0x10, 0x10),
	}

	asf.Data[4] = 0x00
	asf.Data[8] = 0x81

	buf := bytes.Buffer{}
	SerializeRMCP(&buf, rmcp)
	SerializeASF(&buf, asf)
	log.Println("    ASF: Ready to response PONG message")
	server.WriteToUDP(buf.Bytes(), addr)
}

// Comamand Analyzer and Executor
func ASFDeserializeAndExecute(buf io.Reader, addr *net.UDPAddr, server *net.UDPConn) {
	_, asf := DeserializeASF(buf)

	switch asf.MessageType {
	case ASF_TYPE_PING:
		log.Println("    ASF: Message Type = PING")
		HandleASFPong(asf, addr, server)
	}

	// Other commands are ignored here
}