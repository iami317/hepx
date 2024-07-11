package nathole

import (
	"fmt"
	"net"
	"slices"
	"strconv"
)

const (
	EasyNAT = "EasyNAT"
	HardNAT = "HardNAT"

	BehaviorNoChange    = "BehaviorNoChange"
	BehaviorIPChanged   = "BehaviorIPChanged"
	BehaviorPortChanged = "BehaviorPortChanged"
	BehaviorBothChanged = "BehaviorBothChanged"
)

type NatFeature struct {
	NatType            string
	Behavior           string
	PortsDifference    int
	RegularPortsChange bool
	PublicNetwork      bool
}

func ClassifyNATFeature(addresses []string, localIPs []string) (*NatFeature, error) {
	if len(addresses) <= 1 {
		return nil, fmt.Errorf("not enough addresses")
	}
	natFeature := &NatFeature{}
	ipChanged := false
	portChanged := false

	var baseIP, basePort string
	var portMax, portMin int
	for _, addr := range addresses {
		ip, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		portNum, err := strconv.Atoi(port)
		if err != nil {
			return nil, err
		}
		if slices.Contains(localIPs, ip) {
			natFeature.PublicNetwork = true
		}

		if baseIP == "" {
			baseIP = ip
			basePort = port
			portMax = portNum
			portMin = portNum
			continue
		}

		if portNum > portMax {
			portMax = portNum
		}
		if portNum < portMin {
			portMin = portNum
		}
		if baseIP != ip {
			ipChanged = true
		}
		if basePort != port {
			portChanged = true
		}
	}

	switch {
	case ipChanged && portChanged:
		natFeature.NatType = HardNAT
		natFeature.Behavior = BehaviorBothChanged
	case ipChanged:
		natFeature.NatType = HardNAT
		natFeature.Behavior = BehaviorIPChanged
	case portChanged:
		natFeature.NatType = HardNAT
		natFeature.Behavior = BehaviorPortChanged
	default:
		natFeature.NatType = EasyNAT
		natFeature.Behavior = BehaviorNoChange
	}
	if natFeature.Behavior == BehaviorPortChanged {
		natFeature.PortsDifference = portMax - portMin
		if natFeature.PortsDifference <= 5 && natFeature.PortsDifference >= 1 {
			natFeature.RegularPortsChange = true
		}
	}
	return natFeature, nil
}

func ClassifyFeatureCount(features []*NatFeature) (int, int, int) {
	easyCount := 0
	hardCount := 0
	// for HardNAT
	portsChangedRegularCount := 0
	for _, feature := range features {
		if feature.NatType == EasyNAT {
			easyCount++
			continue
		}

		hardCount++
		if feature.RegularPortsChange {
			portsChangedRegularCount++
		}
	}
	return easyCount, hardCount, portsChangedRegularCount
}
