---
task_id: "TASK-0075"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T09:28:29Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0075 QA PLAN

## 方針

この計画はTASKの受け入れ条件を基準に、DEV開始前にQAが独立作成する。安全契約のcompletion分岐と既存product分岐を、隔離されたGit fixtureで再現する。実装後は候補commitに固定したfocused process testを一度だけ再実行し、root `make check`の候補bound証跡を監査する。実OS権限、外部サービス、配置、restartを扱わないため`live-e2e`は割り当てない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1, AC-4 | safety_contract fixtureで、承認済みTASK-first QA_PLAN、canonical `HANDOVER.candidate_commit`、Mainの4 safety checksと時刻を満たし、`REVIEW_RESULT.md`/`QA_RESULT.md`がpendingのままcompletion preflightとcompletionが進行することを確認する。結果PASSの生成・要求がないことと、Main分類・PLAN/QA_PLAN承認の欠落を拒否することを確認する。 | `focused-rerun` / completionの安全契約分岐を、有限でdeterministicなローカルGit fixtureが完全に再現できる。 |
| QA-002 | AC-2, AC-3 | `MERGE_HEAD`中とcommit後main履歴の両fixtureでcandidateがexact two-parent no-ff mergeのsecond parentであることを確認する。candidate欠落・不一致・fast-forward/余分parent、未宣言path、main-managed path、rename/copy、製品pathをそれぞれ拒否することを確認する。 | `focused-rerun` / Git parentとcandidate diffはfixture内で直接観測でき、外部状態を必要としない。 |
| QA-003 | AC-2, AC-4 | safety_contract fixtureのHANDOVERに`merged_commit`、`safety_candidate_tree`、`safety_merge_tree`、`safety_check_digest`がなくても検証がGit導出で成立することを確認する。同時に新field、version、receipt、digest、追加transactionを要求・生成しない差分と証跡を監査する。 | `focused-rerun` / legacy fieldの不在と新規要求の不在は、isolated fixtureとcandidate diffで決定的に検出できる。 |
| QA-004 | AC-5 | product fixtureで固定candidateへの独立REVIEW/QA、candidate-bound DEV `make check`監査、exact no-ff second parentが依然必須であることを確認し、いずれかの欠落を拒否する。root `make check`のPASS出力・実行commitを候補bound証跡として監査する。 | `evidence-review` / product回帰の受入れ証拠はDEV候補上のroot checkを独立監査し、focused rerunが安全契約分岐をproduct PASSへ置換しないことを確認する。 |

実施モードは `evidence-review`、`focused-rerun`、`live-e2e` のいずれかとする。実施不能なケースは理由を記録し、別モードのPASSで置き換えない。

## focused process test

実装後、candidate_commitを固定して次を一度だけ実行する。

```sh
node --test scripts/task/development-process.test.mjs scripts/task/unified-lifecycle.test.mjs
```

この1コマンドはQA-001〜QA-004のfixture、`MERGE_HEAD` pre-mergeとpost-merge main履歴、legacy safety field不在、product completion回帰を対象にする。失敗時は実装不具合と決めつけず、fixture/期待値/実装/環境のいずれかへ分類し、候補を再固定するまでPASSにしない。

## 証跡監査

- root `make check`はDEVがcandidate_commit上で実行した結果を確認し、コマンド、終了状態、実行commit、対象差分との対応を記録する。
- `make check`の失敗・未実施・candidate不一致はQA-004をPASSにしない。安全契約のQA-001〜QA-003は、focused rerunのPASSを製品REVIEW/QA PASSの代替証跡にしない。
- candidate diffはTASKの許可5パスだけであること、既存Task証跡・Lap30 Schema/JSONLを遡及変更しないこと、追加field/version/receipt/digest/transactionがないことを監査する。

## 境界・異常・回帰

- safety_contractでpendingの製品用結果を理由に停止すること、または製品用PASSを生成することは失敗とする。
- Gitから導出可能なcandidate/merge情報をHANDOVERやbacklogへ重複転記することは失敗とする。
- `MERGE_HEAD`の候補、post-mergeのmain second parent、candidate diff宣言のいずれかが不一致なら失敗とする。
- product completionが独立REVIEW/QA又はcandidate-bound DEV checkを省略できるようになれば回帰とする。

## 実装後の再確認

- [ ] 実装差分とレビュー結果を確認した。
- [ ] 操作手順と期待結果を現行実装に合わせた。
- [ ] 期待結果または範囲を変更した場合、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-first初版。Mainがproduct分類、scope、受け入れ経路を確認。 | `main-agent-sol-high / 2026-08-02T09:28:29Z` |
