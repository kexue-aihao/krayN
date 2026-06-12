//go:build linux

package tunbridge

import (
	"log/slog"
	"strings"
)

func setupAutoRoute(options routeOptions) (*routeManager, error) {
	manager := &routeManager{}
	logger := options.Logger
	name := options.InterfaceName

	_ = runLogged(logger, "ip", "addr", "replace", tunIPv4CIDR, "dev", name)
	_ = runLogged(logger, "ip", "link", "set", "dev", name, "up")
	manager.addCleanup(func() {
		_ = runLogged(logger, "ip", "addr", "del", tunIPv4CIDR, "dev", name)
	})

	defaultRoute := detectLinuxDefaultRoute(logger)
	for _, endpoint := range ipv4Strings(options.EndpointIPs) {
		args := []string{"route", "replace", endpoint + "/32"}
		if defaultRoute.gateway != "" {
			args = append(args, "via", defaultRoute.gateway)
		}
		if defaultRoute.device != "" {
			args = append(args, "dev", defaultRoute.device)
		}
		if len(args) > 3 {
			_ = runLogged(logger, "ip", args...)
			manager.addCleanup(func() {
				_ = runLogged(logger, "ip", "route", "del", endpoint+"/32")
			})
		}
	}

	if err := runLogged(logger, "ip", "route", "replace", splitRouteA, "dev", name); err != nil {
		manager.Close()
		return nil, err
	}
	manager.addCleanup(func() {
		_ = runLogged(logger, "ip", "route", "del", splitRouteA, "dev", name)
	})
	if err := runLogged(logger, "ip", "route", "replace", splitRouteB, "dev", name); err != nil {
		manager.Close()
		return nil, err
	}
	manager.addCleanup(func() {
		_ = runLogged(logger, "ip", "route", "del", splitRouteB, "dev", name)
	})

	for _, cidr := range options.ExcludeCIDRs {
		args := []string{"route", "replace", cidr}
		if defaultRoute.gateway != "" {
			args = append(args, "via", defaultRoute.gateway)
		}
		if defaultRoute.device != "" {
			args = append(args, "dev", defaultRoute.device)
		}
		if len(args) > 3 {
			_ = runLogged(logger, "ip", args...)
			manager.addCleanup(func() {
				_ = runLogged(logger, "ip", "route", "del", cidr)
			})
		}
	}
	return manager, nil
}

type linuxDefaultRoute struct {
	gateway string
	device  string
}

func detectLinuxDefaultRoute(logger *slog.Logger) linuxDefaultRoute {
	output, err := runOutput(logger, "ip", "route", "show", "default")
	if err != nil {
		return linuxDefaultRoute{}
	}
	fields := strings.Fields(output)
	route := linuxDefaultRoute{}
	for i := 0; i+1 < len(fields); i++ {
		switch fields[i] {
		case "via":
			route.gateway = fields[i+1]
		case "dev":
			route.device = fields[i+1]
		}
	}
	return route
}
