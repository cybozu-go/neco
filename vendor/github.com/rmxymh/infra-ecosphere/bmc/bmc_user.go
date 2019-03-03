package bmc

import (
	"log"
)

type BMCUser struct {
	Username string
	Password string
}

var bmcUsers map[string]BMCUser

func init() {
	log.Println("Initialize BMCUser Map...")
	bmcUsers = make(map[string]BMCUser)
}

func AddBMCUser(name string, password string) {
	newUser := BMCUser{
		Username: name,
		Password: password,
	}
	bmcUsers[name] = newUser
	log.Printf("BMCUSer: Add user %s\n", name)
}

func RemoveBMCUser(name string) {
	_, ok := bmcUsers[name]

	if ok {
		delete(bmcUsers, name)
	}
}

func GetBMCUser(name string) (BMCUser, bool) {
	obj, ok := bmcUsers[name]

	return obj, ok
}