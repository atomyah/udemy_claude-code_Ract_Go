package utils

import "time"

// ParseCursor はクエリパラメータのcursor文字列をtime.Timeに変換する
func ParseCursor(cursor string) (*time.Time, error) {
	if cursor == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, cursor)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// FormatCursor はtime.Timeをレスポンスのnext_cursor文字列に変換する
func FormatCursor(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}

// ParseLimit はlimitクエリパラメータを検証しデフォルト値・上限を適用する
func ParseLimit(raw string, defaultLimit, maxLimit int) int {
	if raw == "" {
		return defaultLimit
	}
	n := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return defaultLimit
		}
		n = n*10 + int(ch-'0')
	}
	if n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}
