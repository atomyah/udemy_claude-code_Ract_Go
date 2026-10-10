#!/bin/sh
# make test の結果サマリーを表示する。
# 使い方: sh scripts/test-summary.sh <backend の終了コード> <e2e の終了コード> [frontend の終了コード]
backend_status=$1
e2e_status=$2
frontend_status=$3

print_status() {
	if [ "$2" -eq 0 ]; then
		echo "$1: 成功"
	else
		echo "$1: 失敗 (exit $2)"
	fi
}

echo ""
echo "==== テスト結果 ===="
print_status "backend " "$backend_status"
if [ -n "$frontend_status" ]; then
	print_status "frontend" "$frontend_status"
fi
print_status "e2e     " "$e2e_status"
