package engine

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestParseDNSResponseExtractsARecords(t *testing.T) {
	query, err := buildDNSQuery("example.com", dnsTypeA)
	if err != nil {
		t.Fatal(err)
	}
	response := append([]byte(nil), query...)
	response[2] = 0x81
	response[3] = 0x80
	binary.BigEndian.PutUint16(response[6:8], 1)
	response = append(response,
		0xc0, 0x0c,
		0x00, 0x01,
		0x00, 0x01,
		0x00, 0x00, 0x00, 0x3c,
		0x00, 0x04,
		0xcb, 0x00, 0x71, 0x0a,
	)

	ips, err := parseDNSResponse(response, dnsTypeA)
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 1 || !ips[0].Equal(net.IPv4(203, 0, 113, 10)) {
		t.Fatalf("unexpected IPs: %#v", ips)
	}
}

func TestNormalizeTestURLFallsBack(t *testing.T) {
	got := normalizedTestURL("not a url", "https://example.com/generate_204")
	if got != "https://example.com/generate_204" {
		t.Fatalf("fallback mismatch: %s", got)
	}
}

func TestExtractPuritySummary(t *testing.T) {
	html := `<div class="line line-risk">
<div class="riskitem riskcurrent" title="15-25 纯净"><span class="value">22%</span><span class="lab"> 纯净</span></div>
</div>
<div class="line line-nativeip"><div class="content"><span class="label">原生 IP</span></div></div>
<div class="line line-iptype"><div class="content"><span class="label">IDC机房 IP</span><span class="label">谷歌公共 DNS IP</span></div></div>`

	got := extractPuritySummary(html)
	want := "Ping0 22% 纯净 | 原生 IP | IDC机房 IP / 谷歌公共 DNS IP"
	if got != want {
		t.Fatalf("summary mismatch:\n got: %s\nwant: %s", got, want)
	}
}
