#!/usr/bin/env bash
# フロントエンドをビルドして Firebase Hosting にデプロイする（make deploy-frontend から実行）。
#   1. .env.production の検証（API の URL が本番の https URL になっているか）
#   2. 依存パッケージのインストール（npm ci）
#   3. OpenAPI 定義（backend/docs/openapi.yaml）から API 型を生成
#   4. 本番ビルド（vite build --mode production → dist/）
#   5. firebase deploy --only hosting
#
# 環境変数:
#   FIREBASE_PROJECT  デプロイ先の Firebase プロジェクト（既定: .firebaserc の default）
#   SKIP_INSTALL=1    npm ci を省略する（node_modules が最新のとき）
#   FIREBASE_TOKEN は使わない（非推奨）。CI からは GOOGLE_APPLICATION_CREDENTIALS でサービスアカウントを渡す
set -euo pipefail

cd "$(dirname "$0")/.."

ENV_FILE=.env.production
FIREBASE_PROJECT="${FIREBASE_PROJECT:-default}"
FIREBASE_TOOLS_VERSION=15.34.2

if [ ! -f "$ENV_FILE" ]; then
  echo "エラー: frontend/$ENV_FILE がありません。docs/deploy.md を参照して作成してください" >&2
  exit 1
fi

api_url="$(grep -E '^VITE_API_BASE_URL=' "$ENV_FILE" | tail -n 1 | cut -d= -f2- | tr -d '\r')"
case "$api_url" in
  "" )
    echo "エラー: $ENV_FILE の VITE_API_BASE_URL が空です" >&2; exit 1 ;;
  *REPLACE_ME* )
    echo "エラー: $ENV_FILE の VITE_API_BASE_URL が見本の値のままです。Cloud Run の URL に置き換えてください" >&2; exit 1 ;;
  https://*/api/v1 )
    ;;
  * )
    echo "エラー: VITE_API_BASE_URL は https://<Cloud Run の URL>/api/v1 の形式にしてください（現在: $api_url）" >&2; exit 1 ;;
esac

# グローバルの firebase CLI があれば使い、なければバージョン固定の firebase-tools を npx で実行する
if command -v firebase >/dev/null 2>&1; then
  firebase_cmd=(firebase)
else
  firebase_cmd=(npx --yes "firebase-tools@${FIREBASE_TOOLS_VERSION}")
fi

echo "==> API の接続先: $api_url"
echo "==> デプロイ先 Firebase プロジェクト: $FIREBASE_PROJECT"

if [ "${SKIP_INSTALL:-0}" != "1" ]; then
  echo "==> 依存パッケージをインストールします（npm ci）"
  npm ci
fi

echo "==> API の型を生成します（npm run gen:api）"
npm run gen:api

echo "==> 本番ビルドを実行します（vite build --mode production）"
npm run build -- --mode production

echo "==> Firebase Hosting にデプロイします"
"${firebase_cmd[@]}" deploy --only hosting --project "$FIREBASE_PROJECT"

echo "==> デプロイが完了しました"
