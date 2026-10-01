package ogpparser

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// client OGP取得に関する外部へのリクエストは全てこのクライアントを使う
//
// SSRF対策: DNS解決後の接続先IPアドレスを検証してプライベートIPへのアクセスをブロックする。
// 接続ごとに検証するため、リダイレクト先や DNS rebinding にも有効。
var client = http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		// 環境変数のプロキシを使うと接続先IPの検証がプロキシに対して行われてしまうため、使わない
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
			Control: func(_, address string, _ syscall.RawConn) error {
				host, _, err := net.SplitHostPort(address)
				if err != nil {
					return err
				}
				ip := net.ParseIP(host)
				if isPrivateIP(ip) {
					return errors.New("private IP address is not allowed")
				}
				return nil
			},
		}).DialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	},
}

// nonPublicPrefixes net.IP のメソッドでは判定できない、外部からアクセスされるべきでないアドレス範囲
var nonPublicPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),      // "This network" (Linuxでは 0.x.x.x が自ホストに繋がる)
	netip.MustParsePrefix("100.64.0.0/10"),  // Shared Address Space (CGNAT, 一部クラウドのメタデータサーバー)
	netip.MustParsePrefix("192.0.0.0/24"),   // IETF Protocol Assignments
	netip.MustParsePrefix("198.18.0.0/15"),  // Benchmarking
	netip.MustParsePrefix("240.0.0.0/4"),    // Reserved, Broadcast
	netip.MustParsePrefix("64:ff9b::/96"),   // NAT64 (内部のIPv4アドレスに変換され得る)
	netip.MustParsePrefix("64:ff9b:1::/48"), // Local-Use NAT64
	netip.MustParsePrefix("100::/64"),       // Discard-Only
	netip.MustParsePrefix("2001:db8::/32"),  // Documentation
	netip.MustParsePrefix("fec0::/10"),      // Site-local (deprecated)
}

// isPrivateIP はIPアドレスがプライベート、ループバック、リンクローカル、またはその他の内部アドレスかどうかを判定します
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true // 不明なIPはブロック
	}
	// ループバック (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return true
	}
	// プライベートアドレス (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7)
	if ip.IsPrivate() {
		return true
	}
	// リンクローカル (169.254.0.0/16, fe80::/10)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// 未指定アドレス (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return true
	}
	// マルチキャスト
	if ip.IsMulticast() {
		return true
	}
	// その他の非公開アドレス
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	addr = addr.Unmap() // IPv4-mapped IPv6 (::ffff:a.b.c.d) をIPv4として扱う
	for _, prefix := range nonPublicPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
