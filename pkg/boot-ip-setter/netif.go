package main

import (
	"fmt"
	"slices"

	"github.com/vishvananda/netlink"
)

type NetworkInterface interface {
	Name() string
	Up() error
	Down() error
	ListAddrs() ([]string, error)
	AddAddr(addr string) error
	DeleteAddr(addr string) error
	DeleteAllAddr() error
}

type networkInterface struct {
	linkName string
}

func NewInterface(linkName string) NetworkInterface {
	return &networkInterface{
		linkName: linkName,
	}
}

func (n *networkInterface) Name() string {
	return n.linkName
}

func (n *networkInterface) Up() error {
	link, err := netlink.LinkByName(n.linkName)
	if err != nil {
		return fmt.Errorf("failed to find %s: %w", n.linkName, err)
	}

	err = netlink.LinkSetUp(link)
	if err != nil {
		return fmt.Errorf("failed to up %s: %w", n.linkName, err)
	}
	return nil
}

func (n *networkInterface) Down() error {
	link, err := netlink.LinkByName(n.linkName)
	if err != nil {
		return fmt.Errorf("failed to find %s: %w", n.linkName, err)
	}

	err = netlink.LinkSetDown(link)
	if err != nil {
		return fmt.Errorf("failed to down %s: %w", n.linkName, err)
	}
	return nil
}

func (n *networkInterface) ListAddrs() ([]string, error) {
	link, err := netlink.LinkByName(n.linkName)
	if err != nil {
		return nil, fmt.Errorf("failed to find %s: %w", n.linkName, err)
	}

	addrList, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("failed to list addresses on %s: %w", n.linkName, err)
	}

	ret := []string{}
	for _, a := range addrList {
		ret = append(ret, a.IP.String())
	}
	slices.Sort(ret)
	return ret, nil
}

func (n *networkInterface) AddAddr(addr string) error {
	a, err := netlink.ParseAddr(addr + "/32")
	if err != nil {
		return err
	}

	link, err := netlink.LinkByName(n.linkName)
	if err != nil {
		return fmt.Errorf("failed to find %s: %w", n.linkName, err)
	}

	err = netlink.AddrAdd(link, a)
	if err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", a.IP.String(), n.linkName, err)
	}
	return nil
}

func (n *networkInterface) DeleteAddr(addr string) error {
	a, err := netlink.ParseAddr(addr + "/32")
	if err != nil {
		return err
	}

	link, err := netlink.LinkByName(n.linkName)
	if err != nil {
		return fmt.Errorf("failed to find %s: %w", n.linkName, err)
	}

	err = netlink.AddrDel(link, a)
	if err != nil {
		return fmt.Errorf("failed to delete %s from %s: %w", a.IP.String(), n.linkName, err)
	}
	return nil
}

func (n *networkInterface) DeleteAllAddr() error {
	link, err := netlink.LinkByName(n.linkName)
	if err != nil {
		return err
	}

	addrList, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to list addresses on %s: %w", n.linkName, err)
	}

	for _, a := range addrList {
		err = netlink.AddrDel(link, &a)
		if err != nil {
			return fmt.Errorf("failed to delete %s from %s: %w", a.IP.String(), n.linkName, err)
		}
	}
	return nil
}
