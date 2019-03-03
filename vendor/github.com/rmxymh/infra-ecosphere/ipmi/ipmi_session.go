package ipmi

import (
	"math/rand"
	"log"
)
import (
	"github.com/rmxymh/infra-ecosphere/bmc"
)

type IPMISession struct {
	SessionID uint32
	RemoteSessionSequenceNumber uint32
	LocalSessionSequenceNumber uint32
	User bmc.BMCUser
}

var ipmiSessions map[uint32]IPMISession

func init() {
	log.Println("Initialize IPMI Session Map...")
	ipmiSessions = make(map[uint32]IPMISession)
}

func GetNewSession(user bmc.BMCUser) IPMISession {
	sessionId := rand.Uint32()
	for {
		if _, ok := ipmiSessions[sessionId]; ok {
			sessionId = rand.Uint32()
		} else {
			break
		}
	}

	session := IPMISession{}
	session.SessionID = sessionId
	session.User = user

	ipmiSessions[sessionId] = session
	return session
}

func GetSession(id uint32) (IPMISession, bool) {
	obj, ok := ipmiSessions[id]

	return obj, ok
}

func RemoveSession(id uint32) {
	_, ok := ipmiSessions[id]
	if ok {
		delete(ipmiSessions, id)
	}
}

func (session *IPMISession)Inc() {
	session.LocalSessionSequenceNumber += 1
	session.RemoteSessionSequenceNumber += 1
	session.Save()
}

func (session *IPMISession)Save() {
	ipmiSessions[session.SessionID] = *session
}