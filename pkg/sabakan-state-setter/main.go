package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/cybozu-go/log"
	gqlsabakan "github.com/cybozu-go/sabakan/v2/gql"
	"github.com/cybozu-go/well"
	serf "github.com/hashicorp/serf/client"
	"github.com/prometheus/prom2json"
	"github.com/vektah/gqlparser/gqlerror"
)

var (
	flagSabakanAddress = flag.String("sabakan-address", "http://localhost:10080", "sabakan address")
)

type machineStateSource struct {
	serial string
	ipv4   string

	serfStatus *serf.Member
	metrics    []*prom2json.Family
}

func main() {
	flag.Parse()
	well.Go(run)
	well.Stop()
	err := well.Wait()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	sm := new(searchMachineResponse)
	gql, err := newGQLClient(*flagSabakanAddress)
	if err != nil {
		return err
	}

	sm, err = gql.getSabakanMachines(ctx)
	if err != nil {
		return err
	}
	if len(sm.SearchMachines) == 0 {
		return errors.New("no machines found")
	}

	mss := make([]machineStateSource, 0, len(sm.SearchMachines))

	// Get serf members
	serfc, err := serf.NewRPCClient("127.0.0.1:7373")
	if err != nil {
		return err
	}
	members, err := getSerfMembers(serfc)
	if err != nil {
		return err
	}

	for _, m := range sm.SearchMachines {
		mss = append(mss, newMachineStateSource(m, members))
	}

	// Get machine metrics
	env := well.NewEnvironment(ctx)
	for _, m := range mss {
		source := m
		env.Go(source.getMetrics)
	}
	env.Stop()
	err = env.Wait()
	if err != nil {
		// do not exit
		log.Warn("error occurred when get metrics", map[string]interface{}{
			log.FnError: err.Error(),
		})
	}
	for _, ms := range mss {
		state := decideSabakanState(ms)
		if state == stateMetricNotFound {
			continue
		}
		err = gql.setSabakanState(ctx, ms, state)
		if err != nil {
			switch e := err.(type) {
			case *gqlerror.Error:
				// In the case of an invalid state transition, the log may continue to be output.
				// So the log is not output.
				if eType, ok := e.Extensions["type"]; ok && eType == gqlsabakan.ErrInvalidStateTransition {
					break
				}
				log.Warn("error occurred when set state", map[string]interface{}{
					log.FnError: err.Error(),
					"serial":    ms.serial,
				})
			default:
				log.Warn("error occurred when set state", map[string]interface{}{
					log.FnError: err.Error(),
					"serial":    ms.serial,
				})
			}
		}
	}
	return nil
}

func newMachineStateSource(m machine, members []serf.Member) machineStateSource {
	return machineStateSource{
		serial:     m.Spec.Serial,
		ipv4:       m.Spec.IPv4[0],
		serfStatus: findMember(members, m.Spec.IPv4[0]),
	}
}

func findMember(members []serf.Member, addr string) *serf.Member {
	for _, member := range members {
		if member.Addr.Equal(net.ParseIP(addr)) {
			return &member
		}
	}
	return nil
}
