package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCursor(t *testing.T) {
	t.Run("空文字はnilを返す", func(t *testing.T) {
		got, err := ParseCursor("")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("RFC3339Nanoをパースできる", func(t *testing.T) {
		got, err := ParseCursor("2024-01-01T12:00:00Z")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, 2024, got.Year())
	})

	t.Run("不正な形式はエラーを返す", func(t *testing.T) {
		_, err := ParseCursor("2024/01/01")
		assert.Error(t, err)
	})
}

func TestFormatCursor(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	formatted := FormatCursor(base)
	parsed, err := ParseCursor(formatted)

	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.True(t, base.Equal(*parsed))
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "未指定はデフォルト", raw: "", want: 20},
		{name: "有効な値はそのまま", raw: "10", want: 10},
		{name: "上限を超えたら上限値", raw: "100", want: 50},
		{name: "0以下はデフォルト", raw: "0", want: 20},
		{name: "数値以外はデフォルト", raw: "abc", want: 20},
		{name: "負の値はデフォルト", raw: "-5", want: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseLimit(tt.raw, 20, 50))
		})
	}
}

func TestExtractHashtags(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "ハッシュタグなし", content: "こんにちは", want: nil},
		{name: "英字のハッシュタグ", content: "hello #golang world", want: []string{"golang"}},
		{name: "日本語のハッシュタグ", content: "今日は #天気 がいい", want: []string{"天気"}},
		{name: "複数のハッシュタグ", content: "#go #react", want: []string{"go", "react"}},
		{name: "重複は1つにまとめる", content: "#go #go #react", want: []string{"go", "react"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractHashtags(tt.content))
		})
	}
}
