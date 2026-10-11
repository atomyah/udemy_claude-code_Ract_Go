package logger

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// maxStackFrames はスタックトレースに残すフレーム数の上限
const maxStackFrames = 32

// Stack は呼び出し元からのスタックトレースを「関数名 (ファイル:行)」の配列で返す。
// skip=0 で Stack を呼んだ関数自身から始まる。Go ランタイム内部のフレームは除く。
func Stack(skip int) []string {
	pcs := make([]uintptr, maxStackFrames)
	// runtime.Callers 自身と Stack の 2 フレームを飛ばす
	n := runtime.Callers(skip+2, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	stack := make([]string, 0, n)
	for {
		f, more := frames.Next()
		if !strings.HasPrefix(f.Function, "runtime.") {
			stack = append(stack, fmt.Sprintf("%s (%s:%d)", f.Function, f.File, f.Line))
		}
		if !more {
			break
		}
	}
	return stack
}

// stackError はエラーが発生した地点のスタックトレースを保持するエラー
type stackError struct {
	err   error
	stack []string
}

func (e *stackError) Error() string { return e.err.Error() }
func (e *stackError) Unwrap() error { return e.err }

// WithStack は呼び出し元のスタックトレースを err に付ける。
// panic の回復時など、エラーをログに出す場所と発生した場所が離れるときに使う。
func WithStack(err error, skip int) error {
	if err == nil {
		return nil
	}
	return &stackError{err: err, stack: Stack(skip + 1)}
}

// StackOf は WithStack で付けたスタックトレースを返す。付いていなければ nil
func StackOf(err error) []string {
	var se *stackError
	if errors.As(err, &se) {
		return se.stack
	}
	return nil
}
