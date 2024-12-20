package main

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IP Setter", func() {
	Context("selectDHCPServerAddr", func() {
		It("should select IP address", func() {
			testCases := []struct {
				etcdMembers []string
				rackAndIP   map[int]string // key: rack, value: expected IP
			}{
				{
					etcdMembers: []string{"boot-0", "boot-1", "boot-2"},
					rackAndIP: map[int]string{
						0: "10.71.255.1",
						1: "10.71.255.2",
						2: "10.71.255.3",
						3: "",
					},
				},
				{
					etcdMembers: []string{"boot-1", "boot-5", "boot-9", "boot-10", "boot-99", "boot-100", "boot-999"},
					rackAndIP: map[int]string{
						0:    "",
						1:    "10.71.255.1",
						2:    "",
						4:    "",
						5:    "10.71.255.2",
						9:    "10.71.255.3",
						10:   "10.71.255.4",
						99:   "10.71.255.5",
						100:  "10.71.255.1",
						999:  "10.71.255.2",
						1000: "",
					},
				},
			}
			for i, tc := range testCases {
				for rack, expected := range tc.rackAndIP {
					actual, err := selectDHCPServerAddr(tc.etcdMembers, rack)
					Expect(err).NotTo(HaveOccurred(), "Test %d: member=%s, rack=%d", i, tc.etcdMembers, rack)
					Expect(actual).To(Equal(expected), "Test %d: member=%s, rack=%d", i, tc.etcdMembers, rack)
				}
			}
		})

		It("should return error", func() {
			testCases := []struct {
				etcdMembers []string
				rack        int
			}{
				{
					etcdMembers: []string{"boot-0", "invalid-name", "boot-2"},
					rack:        0,
				},
				{
					etcdMembers: []string{"boot-0", "boot-1", "boot-invalidracknumber"},
					rack:        0,
				},
				{
					etcdMembers: []string{"boot-0", "boot-1", "boot-2"},
					rack:        -1,
				},
			}
			for i, tc := range testCases {
				_, err := selectDHCPServerAddr(tc.etcdMembers, tc.rack)
				Expect(err).To(HaveOccurred(), "Test %d: member=%s, rack=%d", i, tc.etcdMembers, tc.rack)
			}
		})
	})

	Context("decideOps", func() {
		It("should decide operations", func() {
			testCases := []struct {
				currentAddrs []string
				nextAddrs    []string
				expected     []*op
			}{
				{
					currentAddrs: []string{},
					nextAddrs:    []string{},
					expected:     []*op{{op: opDeleteAll}, {op: opDown}},
				},
				{
					currentAddrs: []string{"10.0.0.1", "10.0.0.2"},
					nextAddrs:    []string{},
					expected:     []*op{{op: opDeleteAll}, {op: opDown}},
				},
				{
					currentAddrs: []string{"10.0.0.1", "10.0.0.2"},
					nextAddrs:    []string{"10.0.0.1", "10.0.0.2"},
					expected:     []*op{{op: opUp}},
				},
				{
					currentAddrs: []string{},
					nextAddrs:    []string{"10.0.0.1", "10.0.0.2"},
					expected:     []*op{{op: opUp}, {op: opAdd, address: "10.0.0.1"}, {op: opAdd, address: "10.0.0.2"}},
				},
				{
					currentAddrs: []string{"10.0.0.1", "10.0.0.2"},
					nextAddrs:    []string{"10.0.0.3", "10.0.0.2"},
					expected:     []*op{{op: opUp}, {op: opAdd, address: "10.0.0.3"}, {op: opDelete, address: "10.0.0.1"}},
				},
				{
					currentAddrs: []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
					nextAddrs:    []string{"10.0.0.2", "10.0.0.4"},
					expected:     []*op{{op: opUp}, {op: opAdd, address: "10.0.0.4"}, {op: opDelete, address: "10.0.0.1"}, {op: opDelete, address: "10.0.0.3"}},
				},
			}
			for i, tc := range testCases {
				actual := decideOps(tc.currentAddrs, tc.nextAddrs)
				Expect(actual).To(Equal(tc.expected), "Test %d", i)
			}
		})
	})

	Context("runOps", func() {
		It("should operate network interface", func() {
			testCases := []struct {
				ops    []*op
				result []string
			}{
				{
					ops:    []*op{}, // nop
					result: []string{},
				},
				{
					ops:    []*op{{op: opUp}},
					result: []string{"up"},
				},
				{
					ops:    []*op{{op: opDown}},
					result: []string{"down"},
				},
				{
					ops:    []*op{{op: opAdd, address: "10.0.0.1"}},
					result: []string{"add:10.0.0.1"},
				},
				{
					ops:    []*op{{op: opDelete, address: "10.0.0.2"}},
					result: []string{"delete:10.0.0.2"},
				},
				{
					ops:    []*op{{op: opDeleteAll}},
					result: []string{"deleteAll"},
				},
				{
					ops:    []*op{{op: opDeleteAll}, {op: opDown}},
					result: []string{"deleteAll", "down"},
				},
				{
					ops:    []*op{{op: opUp}, {op: opAdd, address: "10.0.0.2"}, {op: opDelete, address: "10.0.0.1"}},
					result: []string{"up", "add:10.0.0.2", "delete:10.0.0.1"},
				},
			}
			for i, tc := range testCases {
				netif := &mockNetIF{called: []string{}}
				err := runOps(netif, tc.ops)
				Expect(err).NotTo(HaveOccurred(), "Test %d", i)
				Expect(netif.called).To(Equal(tc.result), "Test %d", i)
			}
		})

		It("should return error", func() {
			testCases := []struct {
				ops []*op
			}{
				{
					ops: []*op{{op: opUp}},
				},
				{
					ops: []*op{{op: opDown}},
				},
				{
					ops: []*op{{op: opAdd}},
				},
				{
					ops: []*op{{op: opDelete}},
				},
				{
					ops: []*op{{op: opDeleteAll}},
				},
			}
			for i, tc := range testCases {
				netif := &mockNetIF{err: errors.New("test error")}
				err := runOps(netif, tc.ops)
				Expect(err).To(HaveOccurred(), "Test %d", i)
			}
		})
	})
})
