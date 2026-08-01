---
task_id: "TASK-0046"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T09:35:30Z"
revision: 1
implementation_reviewed_at: "2026-08-01T09:59:45Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0046 QA PLAN

## 方針

TASK.mdのplanning input packetだけを要件正本として、同一candidateをDEVおよびREVIEWから独立に評価する。temporary Git fixtureで完結する最小のWiki edit-only回帰testだけを一回focused rerunする。その他はcandidate source、test、DEVのcandidate-bound証跡を監査する。実Codex、network、credentialsは対象外であり、live E2Eを要しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1〜AC-4 | exact candidateで、temporary Git fixtureのみを使うWiki edit-only dirty-preservation testを一回実行する（`node --test --test-name-pattern='wiki edit-only dirty preservation' scripts/task/development-process.test.mjs`）。開始前の同一path上のstaged/unstaged bytesとmode、tracked削除、untracked file、indexを作り、成功時には子の許可Wiki pathだけが残り、開始前状態が不変であることを確認する。子非zero、HEAD変更、stage、scope違反、validation失敗では、開始前HEAD・tracked bytes/削除・mode・index・untrackedを復元し、子が増やしたuntrackedを除去することを確認する。さらに子nonzeroの固定prefix、exit code、bounded stderr要約とBearer/`sk-`/長いtoken候補のredaction、およびraw stderr非露出を検出対象とする。commit-mode clean-startの既存rollbackも同じfixture test内で回帰検出する。 | `focused-rerun` / temporary Git fixtureに閉じ、Codex・network・credentialsなしでAC-1〜AC-4の成功、異常、秘密非露出、既存clean-start回帰を決定的かつ上限付きに再現できる。これがQAによる唯一の再実行である。 |
| QA-002 | AC-1〜AC-3 | candidate diff、`run-wiki-agent.mjs`、`agent-routing.mjs`、およびQA-001のtest sourceを監査する。開始前snapshotとの差分導出がpath集合だけでなく同一path上書き、bytes、mode、存在、indexを扱い、allowlistとrollbackが子差分だけを対象にすることを確認する。untracked配下fileの復元/開始後file除去、特殊file又は復元不能時のfail-closed、rollback失敗と元失敗の区別、stderrのbounded固定redactionとraw stderr非伝播も確認する。 | `evidence-review` / implementationの境界、fail-closed処理、testの失敗検出能力、弱体化の不在はcandidate source/testとcandidate-bound結果を独立監査する。 |
| QA-003 | AC-4 | candidate diffとDEVのcandidate-bound `make check`結果を監査する。commit-modeのdirty開始拒否、clean開始時のHEAD/index/worktree/untracked rollback契約、Wiki allowlist、他launcherのclean-only契約が変更されず、許可パス外、dependency、生成物、証跡形式の変更がないことを確認する。包括checkはQAで再実行しない。 | `evidence-review` / candidate scopeと既存契約、およびDEVが一回だけ実行した包括checkの整合は証跡監査で確認し、重複包括checkを避ける。 |

実施モードは `evidence-review`、`focused-rerun`、`live-e2e` のいずれかとする。実施不能なケースは理由を記録し、別モードのPASSで置き換えない。

## 境界・異常・回帰

- QA-001の失敗はcandidate実装、test/fixture、baseline、実行環境、又はcandidate-bound証跡の不整合へ分類し、DEV faultと決めつけない。
- candidate、test、又はHANDOVERのcandidate_commitが不一致、test削除・期待値緩和、snapshot不能なfile種のfail-open、開始前状態の変化、raw stderr又は秘密候補の公開、scope外差分、又は包括check証跡不足があれば、evidence-reviewをPASSにしない。
- 実Codex、network、credentialsを使う確認はTASKの対象外であり、live-e2eを追加しない。focused rerunのPASSを外部integrationの証拠にはしない。

## 実装後の再確認

- [x] 実装差分を確認し、承認済み期待結果の変更がないことをMainが確認した。
- [x] HANDOVERのcandidate_commitに固定し、QA-001を一回だけ実行した。
- [x] QA-002〜003のcandidate-bound source/evidenceを独立監査した。
- [x] 期待結果または範囲は変更していない。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-01 | qa-agent-terra-medium | TASK-firstの独立QA計画。temporary Git fixtureのfocused rerunを一回、他をsource/evidence auditに割当。 | `main-agent-sol-high / 2026-08-01T09:35:30Z` |
