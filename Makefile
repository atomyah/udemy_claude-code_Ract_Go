.DEFAULT_GOAL := help
.PHONY: help dev-up dev-down dev-logs test-setup test-teardown test-backend test-e2e test

COMPOSE      := docker compose
TEST_COMPOSE := docker compose --profile test
TEST_SERVICES := api_test db_test

# 注意1: Windows 版 GNU make はレシピ行の日本語を CP932 として再エンコードするため文字化けする。
#        レシピ行は ASCII のみで書き、日本語メッセージは scripts/ 配下のファイルに置くこと。
# 注意2: Git Bash (MSYS) はレシピ内の絶対パス引数を Windows パスへ勝手に変換する。
#        コンテナ内のパスを渡すときは相対パス（WORKDIR は /app）で書くこと。

## help: 全コマンドの一覧を表示する
help:
	@cat scripts/make-help.txt

## dev-up: 開発環境を起動する
dev-up:
	$(COMPOSE) up -d

## dev-down: 開発環境を停止する
dev-down:
	$(COMPOSE) stop

## dev-logs: 開発環境のログを表示する
dev-logs:
	$(COMPOSE) logs -f

## test-setup: テスト環境（db_test / api_test）を起動する
## 直前の teardown の後始末と競合して db_test が起動失敗することがあるため、1度だけ作り直して再試行する
test-setup:
	@$(TEST_COMPOSE) up -d --wait $(TEST_SERVICES) || ( \
		echo "test-setup failed. cleaning up and retrying once..."; \
		$(TEST_COMPOSE) rm -sf $(TEST_SERVICES); \
		sleep 5; \
		$(TEST_COMPOSE) up -d --wait $(TEST_SERVICES) )

## test-teardown: テスト環境のみを停止・削除する（開発用 db / api には触れない）
test-teardown:
	-$(TEST_COMPOSE) rm -sf $(TEST_SERVICES)

## test-backend: バックエンドの単体テストを実行する（失敗してもテスト環境は必ず片付ける）
test-backend: test-setup
	@set +e; \
	$(COMPOSE) run --rm api_test sh scripts/run-tests.sh; \
	status=$$?; \
	"$(MAKE)" test-teardown; \
	exit $$status

## test-e2e: PlaywrightのE2Eテストを実行する（失敗してもテスト環境は必ず片付ける）
test-e2e: test-setup
	@set +e; \
	(cd frontend && npx playwright test); \
	status=$$?; \
	"$(MAKE)" test-teardown; \
	exit $$status

## test: バックエンド単体テストとE2Eテストを両方実行する
test:
	@set +e; \
	"$(MAKE)" test-backend; backend_status=$$?; \
	"$(MAKE)" test-e2e; e2e_status=$$?; \
	sh scripts/test-summary.sh $$backend_status $$e2e_status; \
	exit $$((backend_status + e2e_status))
