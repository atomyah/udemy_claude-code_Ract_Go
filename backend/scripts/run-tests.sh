#!/bin/sh
# バックエンドの単体テストを実行し、internal 配下のカバレッジ合計を表示する。
# Makefile のレシピ行に日本語を書くと Windows 版 GNU make が文字化けさせるため、
# 日本語メッセージはこのスクリプト側に置く。
set -e

go test -v -parallel 2 -cover -coverpkg=./internal/... -coverprofile=/tmp/coverage.out ./...

echo "===== カバレッジ（internal 全体） ====="
go tool cover -func=/tmp/coverage.out | tail -1
