package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// captureLogs はグローバルロガーの出力をバッファに差し替え、テスト終了時に標準出力へ戻す
func captureLogs(t *testing.T, level zerolog.Level) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	SetOutput(buf, level)
	t.Cleanup(func() { SetOutput(os.Stdout, zerolog.InfoLevel) })
	return buf
}

// logLines は出力された JSON ログを 1 行ずつデコードする
func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m), "ログは JSON でなければならない: %s", line)
		lines = append(lines, m)
	}
	return lines
}

func TestParseLevel(t *testing.T) {
	cases := []struct {
		env, level string
		want       zerolog.Level
	}{
		{"production", "debug", zerolog.DebugLevel},
		{"production", "INFO", zerolog.InfoLevel},
		{"production", "warn", zerolog.WarnLevel},
		{"production", "error", zerolog.ErrorLevel},
		{"development", "", zerolog.DebugLevel},
		{"test", "", zerolog.InfoLevel},
		{"production", "", zerolog.InfoLevel},
		{"production", "verbose", zerolog.InfoLevel},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, ParseLevel(tc.env, tc.level), "env=%s level=%s", tc.env, tc.level)
	}
}

func TestLogger_OutputsJSONWithLevelAndTime(t *testing.T) {
	buf := captureLogs(t, zerolog.DebugLevel)

	L().Debug().Msg("d")
	L().Info().Msg("i")
	L().Warn().Msg("w")
	L().Error().Msg("e")

	lines := logLines(t, buf)
	require.Len(t, lines, 4)
	for i, level := range []string{"debug", "info", "warn", "error"} {
		assert.Equal(t, level, lines[i]["level"])
		_, err := time.Parse(time.RFC3339Nano, lines[i]["time"].(string))
		assert.NoError(t, err, "time は RFC3339 形式")
	}
}

func TestLogger_LevelFiltersLowerLogs(t *testing.T) {
	buf := captureLogs(t, zerolog.WarnLevel)

	L().Info().Msg("出力されない")
	L().Warn().Msg("出力される")

	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	assert.Equal(t, "出力される", lines[0]["message"])
}

func TestWithRequestID_AddsRequestIDToContextLogs(t *testing.T) {
	buf := captureLogs(t, zerolog.InfoLevel)

	ctx := WithRequestID(context.Background(), "req-123")
	FromContext(ctx).Info().Msg("hello")

	assert.Equal(t, "req-123", RequestID(ctx))
	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	assert.Equal(t, "req-123", lines[0]["request_id"])
}

func TestFromContext_WithoutLoggerFallsBackToGlobal(t *testing.T) {
	buf := captureLogs(t, zerolog.InfoLevel)

	FromContext(context.Background()).Info().Msg("global")

	assert.Equal(t, "", RequestID(context.Background()))
	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	assert.NotContains(t, lines[0], "request_id")
}

func TestStack_StartsAtCaller(t *testing.T) {
	stack := Stack(0)

	require.NotEmpty(t, stack)
	assert.Contains(t, stack[0], "TestStack_StartsAtCaller")
	assert.Contains(t, stack[0], "logger_test.go:")
}

func TestWithStack_KeepsOriginalErrorAndStack(t *testing.T) {
	base := errors.New("boom")
	err := WithStack(base, 0)

	assert.ErrorIs(t, err, base)
	assert.Equal(t, "boom", err.Error())
	require.NotEmpty(t, StackOf(err))
	assert.Contains(t, StackOf(err)[0], "TestWithStack_KeepsOriginalErrorAndStack")
	assert.Nil(t, StackOf(base))
	assert.Nil(t, WithStack(nil, 0))
}

func TestGormLogger_LogsSQLAtDebugOnlyWhenEnabled(t *testing.T) {
	buf := captureLogs(t, zerolog.DebugLevel)
	ctx := WithRequestID(context.Background(), "req-sql")
	fc := func() (string, int64) { return "SELECT 1", 1 }

	NewGormLogger(false).Trace(ctx, time.Now(), fc, nil)
	assert.Empty(t, buf.String(), "SQL ログ無効時は通常の SQL を出さない")

	NewGormLogger(true).Trace(ctx, time.Now(), fc, nil)
	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	assert.Equal(t, "debug", lines[0]["level"])
	assert.Equal(t, "sql", lines[0]["message"])
	assert.Equal(t, "SELECT 1", lines[0]["sql"])
	assert.Equal(t, "req-sql", lines[0]["request_id"])
}

func TestGormLogger_ErrorIsWarnWithoutSQLWhenDisabled(t *testing.T) {
	buf := captureLogs(t, zerolog.InfoLevel)
	fc := func() (string, int64) { return "SELECT secret", 0 }

	NewGormLogger(false).Trace(context.Background(), time.Now(), fc, errors.New("connection refused"))

	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	assert.Equal(t, "warn", lines[0]["level"])
	assert.Equal(t, "connection refused", lines[0]["error"])
	assert.NotContains(t, lines[0], "sql", "本番相当（SQL ログ無効）では SQL 文を出さない")
}

func TestGormLogger_RecordNotFoundIsNotLogged(t *testing.T) {
	buf := captureLogs(t, zerolog.DebugLevel)

	NewGormLogger(false).Trace(context.Background(), time.Now(), func() (string, int64) { return "", 0 }, gorm.ErrRecordNotFound)

	assert.Empty(t, buf.String())
}

func TestGormLogger_SlowQueryIsWarn(t *testing.T) {
	buf := captureLogs(t, zerolog.InfoLevel)

	begin := time.Now().Add(-SlowQueryThreshold - time.Millisecond)
	NewGormLogger(false).Trace(context.Background(), begin, func() (string, int64) { return "SELECT 1", 1 }, nil)

	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	assert.Equal(t, "warn", lines[0]["level"])
	assert.Equal(t, true, lines[0]["slow"])
}
