package menu

import (
	"net"
	"reflect"
	"testing"
)

func TestAddToIPNet(t *testing.T) {
	expectedIPNet := net.IPNet{IP: net.ParseIP("10.0.0.1"), Mask: net.CIDRMask(24, 32)}
	_, addr, _ := net.ParseCIDR("10.0.0.0/24")
	actual := addToIPNet(addr, 1)
	if expectedIPNet.String() != actual.String() {
		t.Errorf("expected %v, actual %v", expectedIPNet, *actual)
	}
}

func TestAddToIP(t *testing.T) {
	expectedIPNet := net.IPNet{IP: net.ParseIP("10.0.0.1"), Mask: net.CIDRMask(24, 32)}
	ip := net.ParseIP("10.0.0.0")
	actual := addToIP(ip, 1, 24)
	if expectedIPNet.String() != actual.String() {
		t.Errorf("expected %v, actual %v", expectedIPNet, *actual)
	}
}

func TestMakeNodeNetwork(t *testing.T) {
	_, expected, _ := net.ParseCIDR("10.69.1.64/26")
	base := net.ParseIP("10.69.0.0")
	actual := makeNodeNetwork(base, 6, 26, 5)
	if !reflect.DeepEqual(*expected, *actual) {
		t.Errorf("expected %v, actual %v", *expected, *actual)
	}
}
