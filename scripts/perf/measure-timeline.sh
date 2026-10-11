#!/usr/bin/env bash
# タイムライン取得API（GET /api/v1/posts）のレスポンスタイムとSQLクエリ数を計測する。
#
# 必ずテスト環境（api_test :8081 / db_test）に対して実行する。開発用 db / api には一切触れない。
# 通常は `make perf-timeline LABEL=before` から呼ぶ（テスト環境の起動・片付けも行う）。
#
# 手順:
#   1. 計測用ユーザーをAPIで新規登録し、アクセストークンを取得する
#   2. scripts/perf/seed.sql で db_test に計測用データを投入する
#   3. ウォームアップ1回の後、トークン付きで REQUESTS 回（既定 10 回）リクエストし、
#      クライアント側の応答時間（平均・最小・最大）、サーバー側の処理時間（アクセスログの latency）、
#      X-DB-Query-Count ヘッダー（1リクエストのSQL本数）を出力する
#   4. 1リクエスト分のSQLログ（api_test を SQL_LOG=true で起動した場合）を保存する
#
# 使い方: bash scripts/perf/measure-timeline.sh [ラベル]
# 環境変数: API_BASE_URL（既定 http://127.0.0.1:8081/api/v1）, REQUESTS（既定 10）, LIMIT（既定 20）,
#           PERF_OUT_DIR（結果の保存先。既定 .perf-results）
#
# 注意: Windows では "localhost" を指定すると IPv6(::1) への接続試行で毎回 約200ms 待たされ、
#       計測値がその待ち時間に埋もれる。既定値は 127.0.0.1 にしている。
set -euo pipefail

LABEL="${1:-measure}"
API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:8081/api/v1}"
REQUESTS="${REQUESTS:-10}"
LIMIT="${LIMIT:-20}"
OUT_DIR="${PERF_OUT_DIR:-.perf-results}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEST_COMPOSE="docker compose --profile test"

# 万一にも開発用APIを計測・汚染しないよう、テスト用APIのポート以外は拒否する
case "$API_BASE_URL" in
  *:8081/*) ;;
  *) echo "ERROR: API_BASE_URL はテスト用API（:8081）を指定してください: $API_BASE_URL" >&2; exit 1 ;;
esac

mkdir -p "$OUT_DIR"
RESULT_FILE="$OUT_DIR/timeline-$LABEL.txt"
SQL_LOG_FILE="$OUT_DIR/timeline-$LABEL-sql.log"

# ---- 1. 計測用ユーザーの登録 -------------------------------------------------------------
suffix="$(date +%s)"
register_body=$(printf '{"email":"perf_viewer_%s@example.com","password":"password123","handle":"perf_viewer_%s","display_name":"計測用閲覧ユーザー"}' "$suffix" "$suffix")
register_resp=$(curl -sS -f -X POST "$API_BASE_URL/auth/register" -H 'Content-Type: application/json' -d "$register_body")
TOKEN=$(echo "$register_resp" | jq -r '.access_token')
VIEWER_ID=$(echo "$register_resp" | jq -r '.user.id')
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "ERROR: 計測用ユーザーのトークン取得に失敗しました: $register_resp" >&2
  exit 1
fi

# ---- 2. テストデータ投入（db_test のみ） ---------------------------------------------------
echo "==> テスト用DB（db_test）に計測用データを投入します"
$TEST_COMPOSE exec -T db_test psql -q -U sns_test_user -d sns_test_db -v viewer_id="$VIEWER_ID" -f - < "$SCRIPT_DIR/seed.sql"

# ---- 3. 計測 -------------------------------------------------------------------------------
url="$API_BASE_URL/posts?limit=$LIMIT"
request() {
  # 出力: "<応答秒> <HTTPステータス> <クエリ数>"
  local headers
  headers=$(mktemp)
  local metrics
  metrics=$(curl -sS -o /dev/null -D "$headers" -w '%{time_total} %{http_code}' -H "Authorization: Bearer $TOKEN" "$url")
  local queries
  queries=$(tr -d '\r' < "$headers" | awk -F': ' 'tolower($1)=="x-db-query-count"{print $2}')
  rm -f "$headers"
  echo "$metrics ${queries:-N/A}"
}

echo "==> ウォームアップ（計測対象外）"
request > /dev/null

echo "==> GET $url を $REQUESTS 回計測します"
{
  echo "label: $LABEL"
  echo "date: $(date '+%Y-%m-%d %H:%M:%S %z')"
  echo "endpoint: GET /api/v1/posts?limit=$LIMIT (認証あり)"
  echo "requests: $REQUESTS"
  echo ""
  echo "#  time_ms  status  db_queries"
} > "$RESULT_FILE"

times=()
for i in $(seq 1 "$REQUESTS"); do
  read -r sec status queries < <(request)
  ms=$(awk -v s="$sec" 'BEGIN{printf "%.1f", s*1000}')
  times+=("$ms")
  if [ "$status" != "200" ]; then
    echo "ERROR: HTTP $status が返りました" >&2
    exit 1
  fi
  printf '%-2s %8s  %6s  %10s\n' "$i" "$ms" "$status" "$queries" >> "$RESULT_FILE"
  last_queries="$queries"
done

printf '%s\n' "${times[@]}" | awk -v q="$last_queries" '
  { sum += $1; if (NR == 1 || $1 < min) min = $1; if (NR == 1 || $1 > max) max = $1 }
  END {
    printf "\naverage_ms: %.1f\nmin_ms: %.1f\nmax_ms: %.1f\ndb_queries_per_request: %s\n", sum / NR, min, max, q
  }' >> "$RESULT_FILE"

# サーバー側の処理時間: Echo のアクセスログ（JSON）の latency（ナノ秒）を集計する。
# クライアント側の値と違い、ネットワーク・Docker のポート転送のオーバーヘッドを含まない。
# ホストとDocker VMの時計がずれることがあるため時刻（--since）では区切らず、
# この時点で最後に記録された REQUESTS 件（= 今計測したリクエスト）を対象にする
sleep 1
$TEST_COMPOSE logs --no-log-prefix --no-color api_test 2>/dev/null \
  | grep '^{"time"' \
  | jq -r 'select(.method == "GET" and (.uri | startswith("/api/v1/posts?"))) | .latency' \
  | tail -n "$REQUESTS" \
  | awk '
    { ms = $1 / 1e6; sum += ms; if (NR == 1 || ms < min) min = ms; if (NR == 1 || ms > max) max = ms }
    END {
      if (NR == 0) { print "server_average_ms: N/A"; exit }
      printf "server_average_ms: %.1f\nserver_min_ms: %.1f\nserver_max_ms: %.1f\nserver_samples: %d\n", sum / NR, min, max, NR
    }' >> "$RESULT_FILE"

# ---- 4. 1リクエスト分のSQLログを保存 ------------------------------------------------------
# 時計のずれの影響を受けないよう、リクエスト前のログ行数を覚えておき、それ以降の行だけを切り出す
api_logs() { $TEST_COMPOSE logs --no-log-prefix --no-color api_test 2>/dev/null; }
lines_before=$(api_logs | wc -l)
request > /dev/null
sleep 1
api_logs | tail -n +"$((lines_before + 1))" | sed 's/\x1b\[[0-9;]*m//g' > "$SQL_LOG_FILE" || true
sql_lines=$(grep -c '\[rows:' "$SQL_LOG_FILE" || true)
echo "sql_log_lines_for_one_request: $sql_lines (SQL_LOG=true のときのみ出力。詳細: $SQL_LOG_FILE)" >> "$RESULT_FILE"

echo ""
cat "$RESULT_FILE"
