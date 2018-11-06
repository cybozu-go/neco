package neco

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestUpdateStatus(t *testing.T) {
	st := &UpdateStatus{
		Version: "1.2.3",
		Step:    3,
		Cond:    CondAbort,
		Message: "msg",
	}

	data, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}

	st2 := new(UpdateStatus)
	err = json.Unmarshal(data, st2)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(st, st2) {
		t.Errorf("st != st2, %+v", st2)
	}
}
