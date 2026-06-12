//go:build windows

package tunbridge

import (
	"crypto/md5"
	"encoding/binary"

	"golang.org/x/sys/windows"
	wgtun "golang.zx2c4.com/wireguard/tun"
)

func prepareTunDevice(name string) {
	sum := md5.Sum([]byte(name))
	wgtun.WintunStaticRequestedGUID = &windows.GUID{
		Data1: binary.LittleEndian.Uint32(sum[0:4]),
		Data2: binary.LittleEndian.Uint16(sum[4:6]),
		Data3: binary.LittleEndian.Uint16(sum[6:8]),
		Data4: [8]byte{sum[8], sum[9], sum[10], sum[11], sum[12], sum[13], sum[14], sum[15]},
	}
}
