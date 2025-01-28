package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkInterface", func() {
	Context("operating interface", Ordered, func() {
		// This test operates actual network interfaces on the host.
		// If that is okay with you, please run this test with the environment variable "RUN_NETIF_TEST=yes".
		// And this test cannot run on a Cicle CI instance. Please run it manually.
		skip := true
		if os.Getenv("RUN_NETIF_TEST") == "yes" {
			skip = false
		}

		const testInterface1 = "test-netif-1"

		BeforeEach(func() {
			if skip {
				Skip("RUN_NETIF_TEST is not set")
			}

			DeferCleanup(func() {
				ipLinkDelete(testInterface1) // ignore error
			})

			By("setting up target network interface")
			err := ipLinkAdd(testInterface1)
			Expect(err).NotTo(HaveOccurred())

			ret, err := ipAddressShow(testInterface1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret).To(HaveLen(1))
			Expect(ret[0].State).To(Equal("DOWN"))
			Expect(ret[0].Addrs).To(HaveLen(0))
		})

		It("should return correct information", func() {
			if skip {
				Skip("RUN_NETIF_TEST is not set")
			}

			By("getting interface name")
			netif := NewInterface(testInterface1)
			Expect(netif.Name()).To(Equal(testInterface1))

			By("listing addresses")
			addrs, err := netif.ListAddrs()
			Expect(err).NotTo(HaveOccurred())
			Expect(addrs).To(BeEmpty())

			By("setting addresses by ip command")
			err = ipAddressAdd(testInterface1, "192.168.0.100")
			Expect(err).NotTo(HaveOccurred())
			err = ipAddressAdd(testInterface1, "192.168.0.101")
			Expect(err).NotTo(HaveOccurred())

			By("listing addresses")
			addrs, err = netif.ListAddrs()
			Expect(err).NotTo(HaveOccurred())
			Expect(addrs).To(ConsistOf("192.168.0.100", "192.168.0.101"))
		})

		It("should change operational state", func() {
			if skip {
				Skip("RUN_NETIF_TEST is not set")
			}

			netif := NewInterface(testInterface1)

			By("enabling interface")
			err := netif.Up()
			Expect(err).NotTo(HaveOccurred())

			ret, err := ipAddressShow(testInterface1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret).To(HaveLen(1))
			// The operational state becomes "UNKNOWN", when a dummy interface is up.
			// https://serverfault.com/questions/629676/dummy-network-interface-in-linux
			Expect(ret[0].State).To(Equal("UNKNOWN"))

			By("enabling interface again")
			err = netif.Up()
			Expect(err).NotTo(HaveOccurred())

			ret, err = ipAddressShow(testInterface1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret).To(HaveLen(1))
			Expect(ret[0].State).To(Equal("UNKNOWN"))

			By("disabling interface")
			err = netif.Down()
			Expect(err).NotTo(HaveOccurred())

			ret, err = ipAddressShow(testInterface1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret).To(HaveLen(1))
			Expect(ret[0].State).To(Equal("DOWN"))

			By("disabling interface again")
			err = netif.Down()
			Expect(err).NotTo(HaveOccurred())

			ret, err = ipAddressShow(testInterface1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret).To(HaveLen(1))
			Expect(ret[0].State).To(Equal("DOWN"))
		})

		It("should add address", func() {
			if skip {
				Skip("RUN_NETIF_TEST is not set")
			}

			netif := NewInterface(testInterface1)

			By("adding address")
			err := netif.AddAddr("192.168.0.100")
			Expect(err).NotTo(HaveOccurred())

			ret, err := ipAddressShow(testInterface1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret[0].ipv4Addrs()).To(ConsistOf("192.168.0.100"))

			By("adding another address")
			err = netif.AddAddr("192.168.0.101")
			Expect(err).NotTo(HaveOccurred())

			ret, err = ipAddressShow(testInterface1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret[0].ipv4Addrs()).To(ConsistOf("192.168.0.100", "192.168.0.101"))

			By("adding address that has already set") // error case
			err = netif.AddAddr("192.168.0.101")
			Expect(err).To(HaveOccurred())
		})

		It("should delete address", func() {
			if skip {
				Skip("RUN_NETIF_TEST is not set")
			}

			By("setting addresses by ip command")
			err := ipAddressAdd(testInterface1, "192.168.0.100")
			Expect(err).NotTo(HaveOccurred())
			err = ipAddressAdd(testInterface1, "192.168.0.101")
			Expect(err).NotTo(HaveOccurred())

			netif := NewInterface(testInterface1)

			By("deleting address")
			err = netif.DeleteAddr("192.168.0.100")
			Expect(err).NotTo(HaveOccurred())

			ret, err := ipAddressShow(testInterface1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret[0].ipv4Addrs()).To(ConsistOf("192.168.0.101"))

			By("deleting address again") // error case
			err = netif.DeleteAddr("192.168.0.100")
			Expect(err).To(HaveOccurred())
		})

		It("should delete all addresses", func() {
			if skip {
				Skip("RUN_NETIF_TEST is not set")
			}

			By("setting addresses by ip command")
			err := ipAddressAdd(testInterface1, "192.168.0.100")
			Expect(err).NotTo(HaveOccurred())
			err = ipAddressAdd(testInterface1, "192.168.0.101")
			Expect(err).NotTo(HaveOccurred())

			netif := NewInterface(testInterface1)

			By("deleting all addresses")
			err = netif.DeleteAllAddr()
			Expect(err).NotTo(HaveOccurred())

			ret, err := ipAddressShow(testInterface1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret[0].ipv4Addrs()).To(BeEmpty())

			By("deleting all addresses again")
			err = netif.DeleteAllAddr()
			Expect(err).NotTo(HaveOccurred())

			ret, err = ipAddressShow(testInterface1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret[0].ipv4Addrs()).To(BeEmpty())
		})
	})

	Context("interface does not exist", Ordered, func() {
		It("should return error", func() {
			netif := NewInterface("test-netif-notfound")

			err := netif.Up()
			Expect(err).To(HaveOccurred())

			err = netif.Down()
			Expect(err).To(HaveOccurred())

			addrs, err := netif.ListAddrs()
			Expect(err).To(HaveOccurred())
			Expect(addrs).To(BeEmpty())

			err = netif.AddAddr("192.168.0.100")
			Expect(err).To(HaveOccurred())

			err = netif.DeleteAddr("192.168.0.100")
			Expect(err).To(HaveOccurred())

			err = netif.DeleteAllAddr()
			Expect(err).To(HaveOccurred())
		})
	})

})

func ipLinkAdd(name string) error {
	_, err := exec.Command("ip", "link", "add", name, "type", "dummy").Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("%v, %s", err, string(exitErr.Stderr))
	}
	return err
}

func ipLinkDelete(name string) error {
	_, err := exec.Command("ip", "link", "delete", name).Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("%v, %s", err, string(exitErr.Stderr))
	}
	return err
}

func ipAddressAdd(name, addr string) error {
	_, err := exec.Command("ip", "address", "add", addr, "dev", name).Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("%v, %s", err, string(exitErr.Stderr))
	}
	return err
}

func ipAddressShow(name string) ([]*ipAddressShowResult, error) {
	out, err := exec.Command("ip", "-j", "address", "show", name).Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("%w, %s", err, string(exitErr.Stderr))
	}
	var ret []*ipAddressShowResult
	err = json.Unmarshal(out, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type ipAddressShowResult struct {
	State string      `json:"operstate"`
	Addrs []*addrInfo `json:"addr_info"`
}

type addrInfo struct {
	Family string `json:"family"`
	Local  string `json:"local"`
}

func (r *ipAddressShowResult) ipv4Addrs() []string {
	if r == nil {
		return nil
	}
	ret := []string{}
	for _, a := range r.Addrs {
		if a.Family == "inet" {
			ret = append(ret, a.Local)
		}
	}
	return ret
}

// NOTE: The outputs of the ip command are as follows.

// (1) After creating a dummy interface.
//
// $ sudo ip link add test type dummy
// $ ip -j address show test | jq .
// [
//   {
//     "ifindex": 14,
//     "ifname": "test",
//     "flags": [
//       "BROADCAST",
//       "NOARP"
//     ],
//     "mtu": 1500,
//     "qdisc": "noop",
//     "operstate": "DOWN",
//     "group": "default",
//     "txqlen": 1000,
//     "link_type": "ether",
//     "address": "c2:55:1b:fc:8c:9c",
//     "broadcast": "ff:ff:ff:ff:ff:ff",
//     "addr_info": []
//   }
// ]

// (2) After adding an address to the interface.
//
// $ sudo ip address add 192.168.0.99 dev test
// $ ip -j address show test | jq .
// [
//   {
//     "ifindex": 14,
//     "ifname": "test",
//     "flags": [
//       "BROADCAST",
//       "NOARP"
//     ],
//     "mtu": 1500,
//     "qdisc": "noop",
//     "operstate": "DOWN",
//     "group": "default",
//     "txqlen": 1000,
//     "link_type": "ether",
//     "address": "c2:55:1b:fc:8c:9c",
//     "broadcast": "ff:ff:ff:ff:ff:ff",
//     "addr_info": [
//       {
//         "family": "inet",
//         "local": "192.168.0.99",
//         "prefixlen": 32,
//         "scope": "global",
//         "label": "test",
//         "valid_life_time": 4294967295,
//         "preferred_life_time": 4294967295
//       }
//     ]
//   }
// ]

// (3) After enabling the interface.
//
// $ sudo ip link set test up
// $ ip -j address show test | jq .
// [
//   {
//     "ifindex": 14,
//     "ifname": "test",
//     "flags": [
//       "BROADCAST",
//       "NOARP",
//       "UP",
//       "LOWER_UP"
//     ],
//     "mtu": 1500,
//     "qdisc": "noqueue",
//     "operstate": "UNKNOWN",
//     "group": "default",
//     "txqlen": 1000,
//     "link_type": "ether",
//     "address": "c2:55:1b:fc:8c:9c",
//     "broadcast": "ff:ff:ff:ff:ff:ff",
//     "addr_info": [
//       {
//         "family": "inet",
//         "local": "192.168.0.99",
//         "prefixlen": 32,
//         "scope": "global",
//         "label": "test",
//         "valid_life_time": 4294967295,
//         "preferred_life_time": 4294967295
//       },
//       {
//         "family": "inet6",
//         "local": "fe80::c055:1bff:fefc:8c9c",
//         "prefixlen": 64,
//         "scope": "link",
//         "valid_life_time": 4294967295,
//         "preferred_life_time": 4294967295
//       }
//     ]
//   }
// ]
