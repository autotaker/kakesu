---
task_id: "TASK-0031"
change_class: product
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20T09:55:45+09:00"
revision: 1
implementation_reviewed_at: "2026-07-20T10:14:30+09:00"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0031 QA PLAN

## TASK-first baseline

この計画は実装PLAN、checker、既存testを読む前に、TASK.md の AC-1〜AC-5 から作成した。
対象は `task-check` の安全契約preflight、PLAN の機械可読な予定変更/生成path、完了時の候補
差分照合、用語集索引、template・docs・自動testである。外部運用repositoryへのcommit束縛、Lap30
skill縮約、既存Done Taskへの遡及要求は対象外とする。

新field名とCLI引数はPLANの未決事項であるため、この計画では意味だけを固定する。識別子・
opt-in条件・宣言場所がPLANで確定した後に各fixtureの入力へ置換するが、ACの期待値を緩めない。

## 共通の実施条件

全ケースの `qa_execution_mode` は `focused-rerun` とする。Node の一時repository fixtureで
`checkTask()` / `task-check` を呼ぶため、Git履歴、file list、PLAN入力、生成pathの有無を隔離して
deterministicかつ上限付きで再現できる。実OS権限、外部service、配置、restart、cleanupは受け入れ
真実に不要である。

- QAは同一 `candidate_commit` / `candidate_tree` のDEV証跡、fixture環境、cache条件、exit、失敗時
  stderr、対象treeのdigestを記録する。候補と証跡が一致しなければ全caseをFAIL/blockedとする。
- `pnpm test:process`（又は同testのNode直接実行）は新規fixture testを必ず含む。test未追加、skip、
  weak assertion、既存testの削除・期待値緩和はnegative detectionでありPASSにしない。
- 実装が実権限、実配置、外部副作用、restart/rollback/cleanupを正しさに要求する形へ拡張した場合は
  `live-e2e` に昇格する。承認済み環境と安全なcleanupがなければblockedで、別modeのPASSで代用しない。
- field/version、許可判定、欠落判定、候補差分の親/ tree、又は旧契約の扱いが曖昧・欠落・矛盾する場合は
  fail-closedでFAILとし、DEV起因とは決めつけず `implementation_defect`、`qa_plan_defect`、
  `requirement_gap`、`environment_issue`、`regression` に分類する。

## 受け入れケース

| ケースID | AC-ID | focused-rerun: 具体的test / command | 正常・境界・negative detection | fail-closed |
|---|---|---|---|---|
| PC-001 | AC-1 | `scripts/task/development-process.test.mjs`: `safety_contract v2 preflight accepts unique declared planned and generated paths`; `pnpm test:process` | opt-in済み安全契約PLANが予定変更pathと生成pathを機械可読に宣言し、preflightが通る。許可外path、宣言path欠落、同一pathの重複を各subtestで拒否する。 | version/宣言がない・未知、予定pathが許可外、生成pathが重複/欠落、又はpreflightを通らずにDEV gateへ進めるならFAIL。 |
| PC-002 | AC-2 | `scripts/task/development-process.test.mjs`: `safety_contract v2 preflight permits docs glossary index only as declared generated path`; `pnpm test:process` | `docs/99-glossary-index.md` は新契約で明示した生成pathとしてのみ通る。未宣言、予定変更path扱い、又は無条件allowlist化をnegative fixtureで検出する。 | 索引が宣言なしで許可される、生成pathとして宣言しても拒否される、又は他の任意docs pathへ許可が拡大するならFAIL。 |
| PC-003 | AC-3 | `scripts/task/development-process.test.mjs`: `safety_contract v2 Done verifies candidate diff is declared and generated paths exist`; `pnpm test:process` | no-ff merge fixtureで、候補第2親の実差分が承認済み予定path内かつ全生成pathが存在する正常例を確認する。実差分の許可外逸脱、宣言済み生成fileの欠落、rename/copy、candidate/merge tree不一致を個別subtestで検出する。 | 予定宣言だけを信頼して実差分を読まない、生成path存在を確認しない、又は不一致をwarning扱いするならFAIL。 |
| PC-004 | AC-4 | `scripts/task/development-process.test.mjs`: `legacy safety_contract plan remains on legacy validation without v2 opt-in`; `pnpm test:process` | version opt-inなしの既存安全契約fixtureは既存allowlist/Done検査で従来どおり通る。旧Taskへ新field、生成path、又は新preflightを暗黙要求しないことを確認する。 | legacy fixtureが新契約を強制される、または新contractがversionなしに適用されるならFAIL。未知versionがlegacyへfallbackする場合もFAIL。 |
| PC-005 | AC-5 | 上記PC-001〜004を含む `pnpm test:process`; `make test-process`; `make check`; `make task-check TASK=TASK-0031` | 正常、許可外path、生成path欠落、実差分逸脱、旧互換の五つが自動testとして存在し、専用test名とfixture assertionで失敗検出能力を監査する。全体検査でtemplate/docs/terminologyとの整合も確認する。 | いずれかの代表境界が自動化されない、testがskip/only、又は `make check` / task-checkがFAILならPASSにしない。 |

## 実施時の監査

1. TASK-first期待値と完成PLANを突合し、version opt-in、予定/生成pathの区別、`docs/99-glossary-index.md` の限定許可、preflight開始点、Done時のcandidate差分照合を確認する。意味変更があればMain承認前に `expectation_changed: true` とし、PASSしない。
2. DEVのcandidate-bound実行記録と同一treeから `pnpm test:process` を独立再実行し、PC-001〜004のtest名、fixture mutation、exit、stderrを確認する。`make test-process`、`make check`、`make task-check TASK=TASK-0031` をPC-005として実行する。
3. `git diff --check <candidate-base>...<candidate>` と `git diff --name-status --find-renames --find-copies-harder <candidate-base>...<candidate>` で、TASK許可path外、rename/copy、test削除/弱体化、未予定の生成物を検査する。PLANの予定pathとの実差分対応表を残す。
4. 合格後もMainは no-ff merge の第2親candidate treeとmerge treeを比較する。不一致、またはPC-001〜005へ影響する修正があればcarry-forwardは使わず影響caseを再実行する。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-20 | QA Agent | TASK.mdだけからAC-1〜AC-5の正常、許可外、欠落、実差分逸脱、旧互換をfixtureベースのfocused-rerunとして定義。 | `main-agent approved` |
