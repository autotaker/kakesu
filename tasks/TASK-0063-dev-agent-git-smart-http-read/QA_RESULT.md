---
task_id: "TASK-0063"
status: complete
qa_agent: "qa-agent-terra-medium"
decision: pass
tested_at: "2026-08-02T01:50:45Z"
---

# TASK-0063 QA RESULT

## 結果

| ケース ID | コマンド/テスト | 結果 |
|---|---|---|
| QA-001 | new candidate diff、policy/handler/transaction negative assertions、および指定 race suite | `PASS` — allowlist内 canonical discovery GET と upload-pack POST のみ。receive-pack、method/path/query/media type/body、host/repository/operation逸脱は credential/network 到達前に拒否する fixture を確認。明示 `github.com:443` は policy allow と transport allow が整合する。 |
| QA-002 | new candidate diff、`capability`/`capabilitycontrol`/`connectsession` tests、および指定 race suite | `PASS` — 明示 `github-git-read` selector が peer-derived subject、同一 repository、`github.com`、5分、1回使用の scope を発行。REST/OpenAI cross-scope と subject mismatch は拒否。 |
| QA-003 | new candidate diff、`egresstransaction`/`brokerhttp` tests、および指定 race suite | `PASS` — strict Basic `x-access-token:cap_…` を consume 後に resolver 一回、real-token Basic 一回へ置換。malformed/cross-scope/reuse/resolver error は resolver/forwarder 又は応答への秘密値到達を防ぐ。 |
| QA-004 | `upstreamtransport` F-1 tests、各層 tests、および指定 race suite | `PASS` — F-1 を解消。`github.com:443` を port-free allowlisted hostnameへ正規化し、resolver は `github.com`、dial は `:443`、TLS SNI は `github.com` を使用する positive testを確認。`:444` と near-match は resolver/dial 前に拒否し、redirect/retryも導入していない。binary responseは operation 対応 status/media type/size 検査後だけ sink delivery。 |
| QA-005 | 指定 9 package focused race suite、既存 REST/OpenAI assertions と new candidate diff | `PASS` — REST/OpenAI の Bearer/JSON/transport contract を保持し、Git read への一般化・test skip/削除は検出されない。 |
| QA-006 | `git status --porcelain`、`git diff --check`、`git diff --stat`/`--name-status`、HANDOVER | `PASS` — HANDOVER正本の新candidateと worktree HEAD は一致かつ clean、20 許可パス、`1,008 insertions / 79 deletions`、対象外差分なし、diff check PASS。DEV `make check`/focused race/diff check の candidate-bound PASS 証跡を確認。Reviewerは独立gateであり、QA開始・PASS条件ではない。 |
| QA-007 | live-e2e | `BLOCKED` — 承認済み実GitHub/DNS/TLS/token/別UID/systemd/VPS環境および安全な cleanup がない。hermetic PASSで置換しない。 |

## 実行証跡

candidate worktree: `/Users/autotaker/git/agent-harness/worktrees/TASK-0063-dev-agent-git-smart-http-read`（candidate識別子の正本は `HANDOVER.md`）

1回だけ実行した bounded focused suite（exit 0）:

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/egresspolicy ./internal/capability ./internal/capabilitycontrol ./internal/connectsession ./internal/proxyca ./internal/brokerhttp ./internal/egresstransaction ./internal/upstreamforwarder ./internal/upstreamtransport
```

9 package はすべて `ok (cached)` と報告された。新candidateソース指紋に一致する Go test cache による結果であり、QA_PLAN の「一回だけ」の制約に従い `-count=1` による追加実行はしなかった。これは QA 実行証跡の性質であり、DEV不具合とは分類しない。

## 発見事項

- 初回candidateの F-1（policyが許可した default HTTPS port を transport が拒否）の再現は、新candidateで確認できない。正規化と positive/negative tests が一貫するため、修正済みと判定する。
- QAとReviewerは同一candidateを相互のPASS開始条件にせず独立に評価する。completion gate ではReviewerの独立結果がなお必要。
- QA-007 は環境・cleanup不在による `blocked` であり、candidate又はDEVの失敗ではない。

## 結論

`PASS (QA-001〜006); QA-007 remains BLOCKED live-e2e.`
