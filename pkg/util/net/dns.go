package net

import (
	"context"
	"net"
)

func SetDefaultDNSAddress(dnsAddress string) {
	if _, _, err := net.SplitHostPort(dnsAddress); err != nil {
		dnsAddress = net.JoinHostPort(dnsAddress, "53")
	}
	// Change default dns server
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return net.Dial(network, dnsAddress)
		},
	}
}
