package utils

import "regexp"

var hashtagPattern = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)

// ExtractHashtags はテキストから#タグを重複なく抽出する（#は含めない）
func ExtractHashtags(content string) []string {
	matches := hashtagPattern.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		tag := m[1]
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}
