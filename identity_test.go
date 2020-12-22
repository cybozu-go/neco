package neco

import (
	"io/ioutil"
	"os"
	"testing"
)

func testMyLRN(t *testing.T) {
	t.Parallel()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	_, err = f.WriteString("1\n")
	if err != nil {
		t.Fatal(err)
	}
	RackFile = f.Name()

	lrn, err := MyLRN()
	if err != nil {
		t.Fatal(err)
	}
	if lrn != 1 {
		t.Error("MyLRN should return 1, actual: ", lrn)
	}
}

func testMyCluster(t *testing.T) {
	t.Parallel()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	_, err = f.WriteString("my-cluster\n")
	if err != nil {
		t.Fatal(err)
	}
	ClusterFile = f.Name()

	cluster, err := MyCluster()
	if err != nil {
		t.Fatal(err)
	}
	if cluster != "my-cluster" {
		t.Error("MyCluster should return 'my-cluster', actual: ", cluster)
	}
}

func testOSCodename(t *testing.T) {
	t.Parallel()

	codename, err := OSCodename()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("os codename:", codename)
}

func TestIdentity(t *testing.T) {
	t.Run("MyLRN", testMyLRN)
	t.Run("MyCluster", testMyCluster)
	t.Run("OSCodename", testOSCodename)
}
