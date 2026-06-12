//go:build windows

package tunbridge

import (
	"errors"
	"log/slog"
	"net"
	"strconv"
	"strings"
)

func setupAutoRoute(options routeOptions) (*routeManager, error) {
	manager := &routeManager{}
	name := options.InterfaceName
	logger := options.Logger
	metric := "1"
	defaultGateway := detectWindowsDefaultGateway(logger)

	_ = runLogged(logger, "netsh", "interface", "ip", "set", "address",
		"name="+name, "static", tunIPv4, tunIPv4Mask)
	manager.addCleanup(func() {
		_ = runLogged(logger, "netsh", "interface", "ip", "delete", "address", "name="+name, "addr="+tunIPv4)
	})

	if options.MTU > 0 {
		_ = runLogged(logger, "netsh", "interface", "ipv4", "set", "subinterface",
			name, "mtu="+strconv.Itoa(options.MTU), "store=active")
	}

	for _, endpoint := range ipv4Strings(options.EndpointIPs) {
		if defaultGateway != "" {
			_ = runLogged(logger, "route", "ADD", endpoint, "MASK", "255.255.255.255", defaultGateway, "METRIC", metric)
			manager.addCleanup(func() {
				_ = runLogged(logger, "route", "DELETE", endpoint, "MASK", "255.255.255.255")
			})
		}
	}

	addRoute := func(destination, mask string) error {
		if err := runLogged(logger, "route", "ADD", destination, "MASK", mask, tunIPv4Peer, "METRIC", metric); err != nil {
			return err
		}
		manager.addCleanup(func() {
			_ = runLogged(logger, "route", "DELETE", destination, "MASK", mask)
		})
		return nil
	}
	if err := addRoute("1.0.0.0", "128.0.0.0"); err != nil {
		manager.Close()
		return nil, err
	}
	if err := addRoute("128.0.0.0", "128.0.0.0"); err != nil {
		manager.Close()
		return nil, err
	}

	for _, cidr := range options.ExcludeCIDRs {
		network, err := parseIPv4CIDR(cidr)
		if err != nil || defaultGateway == "" {
			continue
		}
		_ = runLogged(logger, "route", "ADD", network.ip, "MASK", network.mask, defaultGateway, "METRIC", metric)
		manager.addCleanup(func() {
			_ = runLogged(logger, "route", "DELETE", network.ip, "MASK", network.mask)
		})
	}

	return manager, nil
}

func detectWindowsDefaultGateway(logger *slog.Logger) string {
	output, err := runOutput(logger, "route", "PRINT", "0.0.0.0")
	if err != nil {
		return ""
	}
	var bestGateway string
	bestMetric := int(^uint(0) >> 1)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "0.0.0.0" || fields[1] != "0.0.0.0" {
			continue
		}
		metric, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil {
			continue
		}
		gateway := fields[2]
		if net.ParseIP(gateway).To4() == nil {
			continue
		}
		if metric < bestMetric {
			bestMetric = metric
			bestGateway = gateway
		}
	}
	return bestGateway
}

type ipv4Network struct {
	ip   string
	mask string
}

func parseIPv4CIDR(cidr string) (ipv4Network, error) {
	ip, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return ipv4Network{}, err
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return ipv4Network{}, errors.New("not an IPv4 CIDR")
	}
	mask := net.IP(network.Mask).String()
	return ipv4Network{ip: ip4.String(), mask: mask}, nil
}
