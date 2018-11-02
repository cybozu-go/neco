package etcd

import "testing"

func TestRpgen(t *testing.T) {
	t.Parallel()

	rp, err := rpgen()
	if err != nil {
		t.Fatal(err)
	}

	if len(rp) != 32 {
		t.Error("unexpected random password", rp)
	}

	t.Log("rpgen:", rp)
}
