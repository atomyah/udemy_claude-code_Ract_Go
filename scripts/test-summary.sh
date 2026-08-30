#!/bin/sh
# make test の結果サマリーを表示する。
# 使い方: sh scripts/test-summary.sh <backend の終了コード> <e2e の終了コード>
backend_status=$1
e2e_status=$2

echo ""
echo "==== テスト結果 ===="
if [ "$backend_status" -eq 0 ]; then
	echo "backend: 成功"
else
	echo "backend: 失敗 (exit $backend_status)"
fi
if [ "$e2e_status" -eq 0 ]; then
	echo "e2e    : 成功"
else
	echo "e2e    : 失敗 (exit $e2e_status)"
fi
