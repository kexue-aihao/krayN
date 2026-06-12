package tunbridge

import (
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"strings"
	"sync"
)

const (
	tunIPv4     = "172.18.0.1"
	tunIPv4CIDR = "172.18.0.1/30"
	tunIPv4Peer = "172.18.0.2"
	tunIPv4Mask = "255.255.255.252"
	splitRouteA = "1.0.0.0/1"
	splitRouteB = "128.0.0.0/1"
)

type routeOptions struct {
	InterfaceName string
	MTU           int
	EndpointIPs   []net.IP
	ExcludeCIDRs  []string
	Logger        *slog.Logger
}

type routeManager struct {
	cleanup []func()
	once    sync.Once
}

func (r *routeManager) Close() {
	if r == nil {
		return
	}
	r.once.Do(func() {
		for i := len(r.cleanup) - 1; i >= 0; i-- {
			r.cleanup[i]()
		}
	})
}

func (r *routeManager) addCleanup(fn func()) {
	if fn != nil {
		r.cleanup = append(r.cleanup, fn)
	}
}

func runLogged(logger *slog.Logger, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if logger != nil {
			logger.Warn("route command failed", "command", commandString(name, args), "output", message, "error", err)
		}
		if message != "" {
			return fmt.Errorf("%s: %w: %s", commandString(name, args), err, message)
		}
		return fmt.Errorf("%s: %w", commandString(name, args), err)
	}
	if logger != nil {
		if message := strings.TrimSpace(string(out)); message != "" {
			logger.Debug("route command output", "command", commandString(name, args), "output", message)
		}
	}
	return nil
}

func runOutput(logger *slog.Logger, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if logger != nil {
			logger.Warn("route command failed", "command", commandString(name, args), "output", text, "error", err)
		}
		if text != "" {
			return "", fmt.Errorf("%s: %w: %s", commandString(name, args), err, text)
		}
		return "", fmt.Errorf("%s: %w", commandString(name, args), err)
	}
	return text, nil
}

func commandString(name string, args []string) string {
	return strings.TrimSpace(name + " " + strings.Join(args, " "))
}

func ipv4Strings(ips []net.IP) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		ip4 := ip.To4()
		if ip4 == nil {
			continue
		}
		text := ip4.String()
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		out = append(out, text)
	}
	return out
}
