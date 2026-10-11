package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// ClientIPExtractor はクライアントの大元のIPを返す echo.IPExtractor を作る（e.IPExtractor に設定する）。
//
// X-Forwarded-For の先頭はクライアントが自由に書けるため信頼しない。
// 信頼できるプロキシ（Cloud Run の Google Front End など）は自分が見た接続元IPを末尾に追記するので、
// 右端から trustedHops 番目の値を採用する。
//   - trustedHops = 0: X-Forwarded-For を無視し TCP 接続元アドレスを使う（開発・テスト）
//   - trustedHops = 1: 末尾の値（Cloud Run 直接公開）
//   - trustedHops = 2: 末尾から 2 番目（前段に外部ロードバランサーがある構成）
//
// ヘッダーの要素数が足りない・IP として解釈できない場合は TCP 接続元アドレスにフォールバックする。
func ClientIPExtractor(trustedHops int) echo.IPExtractor {
	return func(r *http.Request) string {
		remote := remoteAddrIP(r)
		if trustedHops <= 0 {
			return remote
		}

		var ips []string
		for _, v := range r.Header.Values(echo.HeaderXForwardedFor) {
			for _, p := range strings.Split(v, ",") {
				if p = strings.TrimSpace(p); p != "" {
					ips = append(ips, p)
				}
			}
		}
		if len(ips) < trustedHops {
			return remote
		}
		if ip := net.ParseIP(ips[len(ips)-trustedHops]); ip != nil {
			return ip.String()
		}
		return remote
	}
}

func remoteAddrIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
