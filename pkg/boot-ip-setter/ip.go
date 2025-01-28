package main

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cybozu-go/neco"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func runIPSetter(ctx context.Context, logger *slog.Logger, etcdClient *clientv3.Client, netif NetworkInterface, errorCounter *atomic.Int32, interval time.Duration, rack int) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		logger.Debug("runOnce")
		err := runOnce(ctx, logger, etcdClient, netif, errorCounter, rack)
		if err != nil {
			return err
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func runOnce(ctx context.Context, logger *slog.Logger, etcdClient *clientv3.Client, netif NetworkInterface, errorCounter *atomic.Int32, rack int) error {
	memberListResp, err := etcdClient.MemberList(ctx)
	if err != nil {
		return fmt.Errorf("failed to get member list: %w", err)
	}
	members := make([]string, 0, len(memberListResp.Members))
	for _, m := range memberListResp.Members {
		members = append(members, m.Name)
	}

	var nextAddrs []string
	dhcpAddr, err := selectDHCPServerAddr(members, rack)
	if err != nil {
		return fmt.Errorf("failed to select dhcp server address: %w", err)
	}
	if dhcpAddr != "" {
		nextAddrs = []string{dhcpAddr, neco.VirtualIPAddrActiveBootServer}
	}
	logger.Debug("expected addresses", "addrs", nextAddrs)

	currentAddrs, err := netif.ListAddrs()
	if err != nil {
		errorCounter.Add(1)
		logger.Error("failed to list current addresses", "error", err)
		return nil // ignore error
	}
	logger.Debug("current addresses", "addrs", currentAddrs)

	ops := decideOps(currentAddrs, nextAddrs)
	logger.Debug("ops", "ops", ops)

	err = runOps(netif, ops)
	if err != nil {
		errorCounter.Add(1)
		logger.Error("operation error", "error", err)
		return nil // ignore error
	}

	logger.Debug("operations completed")
	return nil
}

func selectDHCPServerAddr(etcdMembers []string, rack int) (string, error) {
	if rack < 0 {
		return "", fmt.Errorf("invalid rack number: %d", rack)
	}

	var rackList []int
	for _, name := range etcdMembers {
		s, found := strings.CutPrefix(name, "boot-")
		if !found {
			return "", fmt.Errorf("failed to cut rack number from etcd member name: %s", name)
		}
		r, err := strconv.Atoi(s)
		if err != nil {
			return "", fmt.Errorf("failed to convert rack number: name=%s, %v", name, err)
		}
		rackList = append(rackList, r)
	}
	slices.Sort(rackList)

	if n, found := slices.BinarySearch(rackList, rack); found {
		return neco.DHCPServerAddressList[n%len(neco.DHCPServerAddressList)], nil
	}
	return "", nil
}

const (
	opUp = iota
	opDown
	opAdd
	opDelete
	opDeleteAll
)

type op struct {
	op      int
	address string
}

func (o *op) String() string {
	if o == nil {
		return "nil"
	}
	switch o.op {
	case opUp:
		return "up"
	case opDown:
		return "down"
	case opAdd:
		return "add:" + o.address
	case opDelete:
		return "delete:" + o.address
	case opDeleteAll:
		return "deleteAll"
	}
	return "invalid"
}

func decideOps(currentAddrs []string, nextAddrs []string) []*op {
	if len(nextAddrs) == 0 {
		return []*op{{op: opDeleteAll}, {op: opDown}}
	}

	currentAddrMap := map[string]bool{}
	nextAddrMap := map[string]bool{}
	for _, a := range currentAddrs {
		currentAddrMap[a] = true
	}
	for _, a := range nextAddrs {
		nextAddrMap[a] = true
	}

	ret := []*op{{op: opUp}}
	for _, a := range nextAddrs {
		if !currentAddrMap[a] {
			ret = append(ret, &op{op: opAdd, address: a})
		}
	}
	for _, a := range currentAddrs {
		if !nextAddrMap[a] {
			ret = append(ret, &op{op: opDelete, address: a})
		}
	}
	return ret
}

func runOps(netif NetworkInterface, ops []*op) error {
	for _, op := range ops {
		var err error
		switch op.op {
		case opUp:
			err = netif.Up()
		case opDown:
			err = netif.Down()
		case opAdd:
			err = netif.AddAddr(op.address)
		case opDelete:
			err = netif.DeleteAddr(op.address)
		case opDeleteAll:
			err = netif.DeleteAllAddr()
		}
		if err != nil {
			return fmt.Errorf("failed to run operation %s: %w", op.String(), err)
		}
	}
	return nil
}
