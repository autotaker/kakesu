---
task_id: "TASK-0073"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T08:33:50Z"
revision: 1
implementation_reviewed_at: "2026-08-02T08:54:54Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0073 QA PLAN

## 方針

TASK本文だけを期待値の正本とする。candidate固定後、専用worktreeの`tools/dev-agent-harness`で次のfocused suiteを一回だけ実行する。

```sh
GOCACHE=/Users/autotaker/git/agent-harness/.build/go-cache go test -count=1 -race ./internal/approvaldecision
```

suiteはreal `approvalstate` / `approvalchallenge` integrationとpackage-private failure fakeを含み、network、実credential、WebAuthn、Tailscaleを使わない。root `make check`、harness check/distcheck、docs lint、`git diff --check`はDEVのcandidate-boundコマンド/結果を監査し、QAでは再実行しない。candidate識別子はHANDOVERだけを正本とし、本書やQA_RESULT frontmatterへ重複記録しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測 | モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | production constructorがstore、manager、trusted verifierを固定し、Beginがstoreのpending recordからrequest ID/digestを導出してIssueする。callerはdigest/verifier/time/challengeを注入できない。 | `focused-rerun` / public API、ordering fake、real integrationで決定的に検出できる。 |
| QA-002 | AC-2 | Completeがfixed verifierによるConsumeを先に行い、verified bindingだけからApprove/Denyを一度選び、durable transition成功後だけrecordとcredential stable IDを返す。 | `focused-rerun` / call order、exact values、approve/denyをfixtureで検出できる。 |
| QA-003 | AC-3 | verifier failure/panic/replay、expiry/terminal/digest mismatch、state persistence/poison failure、複数approve/deny競合でfallback・challenge復活・成功推測をせず、最初のdurable pending transitionだけが成功する。 | `focused-rerun` / bounded negative/failure/race fixtureで再現できる。 |
| QA-004 | AC-4 | real package integrationがBegin→Complete→durable Get、one-shot replay拒否、response loss後のGet照合を通す。private fakesが順序/回数を検出し、assertion/result copy、固定非漏えいerror、race不在を確認する。 | `focused-rerun` / real wiringとfailure seamの両方が必要である。 |
| QA-005 | AC-5 | package/READMEがtrusted verifier seamを実WebAuthn/Tailscale identityと扱わず、audit、grant、push authorizationへ昇格しない。 | `evidence-review` / candidate diffとREADMEを監査する。実環境項目のPASSにはしない。 |
| QA-006 | AC-6 | 許可3パス、約700〜1,100 additions、stdlib-only、新規dependency/config/generated artifactなし。DEVのfocused/harness/root/docs/diff結果が同じcandidateに結び付く。 | `evidence-review` / diffと簡潔なHANDOVER証跡を監査し、全体checkを重複実行しない。 |

## live-e2e

実WebAuthn assertion/credential、Tailscale identity/Serve/Grant、HTTP/UI/session/CSRF、audit、grant、実pushは`blocked/not run`とする。承認済み環境、権限、test identity、外部作用の遮断とcleanup手順が揃う後続Taskまで、focused/evidence PASSで代替しない。

## 判定

- focused suiteは一回のexitと各fixtureの検出能力でQA-001〜004を判定する。retryや別package testを追加しない。
- failureはcandidate実装、test/evidence、baseline/environment、仕様不整合へ分類し、直ちにDEV faultと決めつけない。
- 高リスク経路のfixture欠落、test弱体化、許可path/dependency逸脱、candidate-bound証跡不足はPASSにしない。
- 実装後に期待値を変更する場合だけMain承認とfrontmatter更新を要求する。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium / main-agent-sol-high | TASK-first QA計画を重複のない6ケースへ縮約 | `main-agent-sol-high` |
