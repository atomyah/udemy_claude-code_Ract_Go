package utils

import "regexp"

// mentionPattern はフロントエンド（LinkifiedText）のリンク化と同じ条件で@メンションを抽出する
var mentionPattern = regexp.MustCompile(`@([A-Za-z0-9_]+)`)

// ExtractMentions はテキストから@ハンドルを重複なく抽出する（@は含めない）
func ExtractMentions(content string) []string {
	matches := mentionPattern.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var handles []string
	for _, m := range matches {
		handle := m[1]
		if !seen[handle] {
			seen[handle] = true
			handles = append(handles, handle)
		}
	}
	return handles
}
