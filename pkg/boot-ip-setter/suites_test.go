package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBootIPSetter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "boot-ip-setter")
}
