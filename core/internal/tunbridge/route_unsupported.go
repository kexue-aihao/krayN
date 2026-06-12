//go:build !windows && !linux && !darwin

package tunbridge

import (
	"fmt"
	"runtime"
)

func setupAutoRoute(options routeOptions) (*routeManager, error) {
	return nil, fmt.Errorf("tun auto_route is not supported on %s", runtime.GOOS)
}
