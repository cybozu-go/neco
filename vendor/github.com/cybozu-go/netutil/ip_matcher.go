package netutil

import (
	"errors"
	"fmt"
	"net"

	"github.com/onsi/gomega/types"
)

type equalIP struct {
	expected net.IP
}

// EqualIP is a custom mather of Gomega to assert IP equivalence
func EqualIP(ip net.IP) types.GomegaMatcher {
	return equalIP{expected: ip}
}

func (m equalIP) Match(actual interface{}) (success bool, err error) {
	ip, ok := actual.(net.IP)
	if !ok {
		return false, errors.New("EqualIP matcher expects an net.IP")
	}

	return ip.Equal(m.expected), nil
}

func (m equalIP) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf(`Expected
	%s
to be the same as
	%s`, actual, m.expected)
}

func (m equalIP) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf(`Expected
	%s
not equal to
	%s`, actual, m.expected)
}
