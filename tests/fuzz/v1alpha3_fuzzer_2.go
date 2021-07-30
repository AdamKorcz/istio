package fuzz

import (
	"errors"
	"fmt"
	fuzz "github.com/AdaLogics/go-fuzz-headers"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/simulation"
	"istio.io/istio/pilot/pkg/xds"
	"testing"
)

func init() {
	testing.Init()
}

func getProtocol(f *fuzz.ConsumeFuzzer) (simulation.Protocol, error) {
	randInt, err := f.GetInt()
	if err != nil {
		return simulation.HTTP, err
	}
	switch randInt % 3 {
	case 1:
		return simulation.HTTP, nil
	case 2:
		return simulation.HTTP2, nil
	case 3:
		return simulation.TCP, nil
	}
	return simulation.HTTP, errors.New("Could not get a protocol")
}

func getPort(f *fuzz.ConsumeFuzzer) (int, error) {
	randInt, err := f.GetInt()
	if err != nil {
		return 0, err
	}
	return randInt % 64738, nil
}

func ValidateFakeOptions(fo xds.FakeOptions) error {
	for _, ko := range fo.KubernetesObjects {
		if ko == nil {
			return errors.New("a Kubernetes Object was nil")
		}
	}
	return nil
}

func FuzzWithCall(data []byte) int {
	t := &testing.T{}
	f := fuzz.NewConsumer(data)
	maxCalls := 500
	noOfCalls, err := f.GetInt()
	if err != nil {
		return 0
	}
	noOfCalls = noOfCalls % maxCalls
	calls := make([]simulation.Call, noOfCalls, noOfCalls)
	for i := 0; i < noOfCalls; i++ {
		call := simulation.Call{}
		err := f.GenerateStruct(&call)
		if err != nil {
			return 0
		}
		port, err := getPort(f)
		if err != nil {
			return 0
		}
		protocol, err := getProtocol(f)
		if err != nil {
			return 0
		}
		call.Port = port
		call.Protocol = protocol
		calls = append(calls, call)
	}
	if len(calls) == 0 {
		return 0
	}

	fo := xds.FakeOptions{}
	err = f.GenerateStruct(&fo)
	if err != nil {
		return 0
	}
	err = ValidateFakeOptions(fo)
	if err != nil {
		return 0
	}
	s := xds.NewFakeDiscoveryServer(t, fo)
	proxy := &model.Proxy{}
	err = f.GenerateStruct(proxy)
	if err != nil {
		return 0
	}
	sim := simulation.NewSimulation(t, s, s.SetupProxy(proxy))
	for _, call := range calls {
		sim.Run(call)
	}
	return 1
}
