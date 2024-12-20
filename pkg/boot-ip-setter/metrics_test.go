package main

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

var _ = Describe("Metrics", Ordered, func() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	var collector prometheus.Collector
	var netif *mockNetIF
	var errorCounter *atomic.Int32

	BeforeAll(func() {
		netif = &mockNetIF{addrs: []string{"10.0.0.1", "10.0.0.2"}}
		errorCounter = &atomic.Int32{}
		collector = newCollector(logger, "testhost", netif, errorCounter)
	})

	It("should return hostname", func() {
		expected := `
			# HELP boot_ip_setter_hostname The hostname this program runs on.
			# TYPE boot_ip_setter_hostname gauge
			boot_ip_setter_hostname{hostname="testhost"} 1
		`
		Expect(testutil.CollectAndCompare(collector, strings.NewReader(expected), "boot_ip_setter_hostname")).NotTo(HaveOccurred())
	})

	It("should update address metrics", func() {
		expected := `
			# HELP boot_ip_setter_interface_address The IP address set to the target interface.
			# TYPE boot_ip_setter_interface_address gauge
			boot_ip_setter_interface_address{interface="mock",ipv4="10.0.0.1"} 1
			boot_ip_setter_interface_address{interface="mock",ipv4="10.0.0.2"} 1
		`
		err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "boot_ip_setter_interface_address")
		Expect(err).NotTo(HaveOccurred())

		netif.addrs = []string{}
		expected = ``
		err = testutil.CollectAndCompare(collector, strings.NewReader(expected), "boot_ip_setter_interface_address")
		Expect(err).NotTo(HaveOccurred())

		netif.addrs = []string{"10.0.0.10", "10.0.0.11", "10.0.0.12"}
		expected = `
			# HELP boot_ip_setter_interface_address The IP address set to the target interface.
			# TYPE boot_ip_setter_interface_address gauge
			boot_ip_setter_interface_address{interface="mock",ipv4="10.0.0.10"} 1
			boot_ip_setter_interface_address{interface="mock",ipv4="10.0.0.11"} 1
			boot_ip_setter_interface_address{interface="mock",ipv4="10.0.0.12"} 1
		`
		err = testutil.CollectAndCompare(collector, strings.NewReader(expected), "boot_ip_setter_interface_address")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should count up error metrics", func() {
		expected := `
			# HELP boot_ip_setter_interface_operation_errors_total The number of times the interface operation failed.
			# TYPE boot_ip_setter_interface_operation_errors_total counter
			boot_ip_setter_interface_operation_errors_total 0
		`
		err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "boot_ip_setter_interface_operation_errors_total")
		Expect(err).NotTo(HaveOccurred())

		errorCounter.Add(1)
		expected = `
			# HELP boot_ip_setter_interface_operation_errors_total The number of times the interface operation failed.
			# TYPE boot_ip_setter_interface_operation_errors_total counter
			boot_ip_setter_interface_operation_errors_total 1
		`
		err = testutil.CollectAndCompare(collector, strings.NewReader(expected), "boot_ip_setter_interface_operation_errors_total")
		Expect(err).NotTo(HaveOccurred())

		errorCounter.Add(3)
		expected = `
			# HELP boot_ip_setter_interface_operation_errors_total The number of times the interface operation failed.
			# TYPE boot_ip_setter_interface_operation_errors_total counter
			boot_ip_setter_interface_operation_errors_total 4
		`
		err = testutil.CollectAndCompare(collector, strings.NewReader(expected), "boot_ip_setter_interface_operation_errors_total")
		Expect(err).NotTo(HaveOccurred())

		// It will be count up, when NetworkInterface returns error in the metrics collector.
		netif.err = errors.New("metrics test")
		expected = `
			# HELP boot_ip_setter_interface_operation_errors_total The number of times the interface operation failed.
			# TYPE boot_ip_setter_interface_operation_errors_total counter
			boot_ip_setter_interface_operation_errors_total 5
		`
		err = testutil.CollectAndCompare(collector, strings.NewReader(expected), "boot_ip_setter_interface_operation_errors_total")
		Expect(err).NotTo(HaveOccurred())
	})
})
