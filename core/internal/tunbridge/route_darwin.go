//go:build darwin

package tunbridge

import (
	"log/slog"
	"strings"
)

func setupAutoRoute(options routeOptions) (*routeManager, error) {
	manager := &routeManager{}
	logger := options.Logger
	name := options.InterfaceName

	_ = runLogged(logger, "ifconfig", name, "inet", tunIPv4, tunIPv4Peer, "netmask", tunIPv4Mask, "up")
	manager.addCleanup(func() {
		_ = runLogged(logger, "ifconfig", name, "down")
	})

	defaultGateway := detectDarwinDefaultGateway(logger)
	for _, endpoint := range ipv4Strings(options.EndpointIPs) {
		if defaultGateway == "" {
			continue
		}
		_ = runLogged(logger, "route", "-n", "add", "-host", endpoint, defaultGateway)
		manager.addCleanup(func() {
			_ = runLogged(logger, "route", "-n", "delete", "-host", endpoint)
		})
	}

	if err := runLogged(logger, "route", "-n", "add", "-net", "1.0.0.0/1", "-interface", name); err != nil {
		manager.Close()
		return nil, err
	}
	manager.addCleanup(func() {
		_ = runLogged(logger, "route", "-n", "delete", "-net", "1.0.0.0/1")
	})
	if err := runLogged(logger, "route", "-n", "add", "-net", "128.0.0.0/1", "-interface", name); err != nil {
		manager.Close()
		return nil, err
	}
	manager.addCleanup(func() {
		_ = runLogged(logger, "route", "-n", "delete", "-net", "128.0.0.0/1")
	})

	for _, cidr := range options.ExcludeCIDRs {
		if defaultGateway == "" {
			continue
		}
		_ = runLogged(logger, "route", "-n", "add", "-net", cidr, defaultGateway)
		manager.addCleanup(func() {
			_ = runLogged(logger, "route", "-n", "delete", "-net", cidr)
		})
	}
	return manager, nil
}

func detectDarwinDefaultGateway(logger *slog.Logger) string {
	output, err := runOutput(logger, "route", "-n", "get", "default")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "gateway:" {
			return fields[1]
		}
	}
	return ""
}
