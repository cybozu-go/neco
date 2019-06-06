package gql

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/cybozu-go/sabakan/v2"
	"github.com/vektah/gqlparser/gqlerror"
)

const (
	// ErrInvalidStateName is an error code when state name is invalid.
	ErrInvalidStateName = "INVALID_STATE_NAME"

	// ErrInvalidStateTransition is an error code when state transition is invalid.
	ErrInvalidStateTransition = "INVALID_STATE_TRANSITION"

	// ErrEncryptionKeyExists is an error code when a retiring machine to retired that still has disk encryption keys.
	ErrEncryptionKeyExists = "ENCRYPTION_KEY_EXISTS"

	// ErrMachineNotFound is an error code when no specified machine found.
	ErrMachineNotFound = "MACHINE_NOT_FOUND"

	// ErrInternalServerError is an error code when internal server error has occurred.
	ErrInternalServerError = "INTERNAL_SERVER_ERROR"
)

// IPAddress represents "IPAddress" GraphQL custom scalar.
type IPAddress net.IP

// UnmarshalGQL implements graphql.Marshaler interface.
func (a *IPAddress) UnmarshalGQL(v interface{}) error {
	str, err := graphql.UnmarshalString(v)
	if err != nil {
		return fmt.Errorf("invalid IPAddress: %v, %v", v, err)
	}

	ip := net.ParseIP(str)
	if ip == nil {
		return fmt.Errorf("invalid IPAddress: %s", str)
	}

	*a = IPAddress(ip)
	return nil
}

// MarshalGQL implements graphql.Marshaler interface.
func (a IPAddress) MarshalGQL(w io.Writer) {
	graphql.MarshalString(net.IP(a).String()).MarshalGQL(w)
}

// DateTime represents "DateTime" GraphQL custom scalar.
type DateTime time.Time

// UnmarshalGQL implements graphql.Marshaler interface.
func (dt *DateTime) UnmarshalGQL(v interface{}) error {
	t, err := graphql.UnmarshalTime(v)
	if err != nil {
		return fmt.Errorf("invalid DateTime: %v, %v", v, err)
	}

	*dt = DateTime(t)
	return nil
}

// MarshalGQL implements graphql.Marshaler interface.
func (dt DateTime) MarshalGQL(w io.Writer) {
	graphql.MarshalTime(time.Time(dt)).MarshalGQL(w)
}

// MarshalMachineState helps mapping sabakan.MachineState with GraphQL enum.
func MarshalMachineState(state sabakan.MachineState) graphql.Marshaler {
	return graphql.MarshalString(strings.ToUpper(state.String()))
}

// UnmarshalMachineState helps mapping sabakan.MachineState with GraphQL enum.
func UnmarshalMachineState(v interface{}) (sabakan.MachineState, error) {
	str, err := graphql.UnmarshalString(v)
	if err != nil {
		return "", err
	}
	st := sabakan.MachineState(strings.ToLower(str))
	if !st.IsValid() {
		return "", &gqlerror.Error{
			Message: "invalid state: " + str,
			Extensions: map[string]interface{}{
				"type": ErrInvalidStateName,
			},
		}
	}

	return st, nil
}
