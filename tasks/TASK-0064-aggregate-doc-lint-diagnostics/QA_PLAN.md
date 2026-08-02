---
task_id: "TASK-0064"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T02:05:12Z"
revision: 1
implementation_reviewed_at: "2026-08-02T02:19:15Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0064 QA PLAN

## 方針

TASK.md の Planning input packet を唯一の期待値根拠とする。candidate commit に対し、fake runner を用いる bounded・deterministic な unit test を独立に一回再実行し、all-diagnostics の制御フロー、終了状態、spawn 設定を検証する。既存 `make lint-docs` と `make check` の入口・検査内容、および candidate の許可パス・規模は candidate-bound diff と DEV 証跡を監査する。root `make check` は DEV 実行証跡を監査し、QA では重複実行しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1, AC-2 | candidateの`scripts/task/development-process.test.mjs`に追加したdoc lint runner casesをNodeで一回実行する。fake runnerのfirst-fail-continuesケースで、terminology → textlint → `git diff --check`が固定順・各一回で呼ばれ、先頭non-zero後も後続を呼ぶこと、複数failでも全command実行後にnon-zeroになることを確認する。 | `focused-rerun` / 既存`test:process`列挙済みunit testはfake processに閉じ、実network・OS権限・外部状態に依存しないbounded deterministicな受け入れ真実を再現できる。 |
| QA-002 | AC-2 | 同じ bounded unit test の all-pass と spawn-error ケースを確認する。all-pass は全 command 実行後 zero、spawn error は記録して後続を実行し最終 non-zero とする。各 fake command の stdout/stderr が親へ即時継承される設定であることを test assertion と candidate source の両方で確認する。 | `focused-rerun` / exit 集約、起動失敗継続、stdio 継承はいずれも fake runner の観測値と明示 assertion で hermetic に検証可能である。 |
| QA-003 | AC-3 | candidate diff と fake runner の spawn 呼出記録を監査する。3 command が固定 argv 配列であり、`shell: false`（又は shell 未指定で false の既定値を assertion）で起動され、repository input から command を組み立てず、retry/cache/parallel/autofix、秘密又は入力値の新規ログを追加しないことを確認する。 | `evidence-review` / セキュリティおよび禁止機能の不在は candidate-bound source/diff と test assertion の独立監査が適切であり、live environment は不要。 |
| QA-004 | AC-1, AC-3 | candidate diff を main の既存 `Makefile` と対照する。`make lint-docs` と `make check` の入口を維持し、terminology validator、Markdown textlint、`git diff --check` という既存3検査の内容・順序を変更せず、runner 化は docs lint 内の fail-fast 除去だけであることを確認する。 | `evidence-review` / Make target と固定 command の同一性は candidate-bound diff の比較で確認でき、QA による root `make check` 再実行は不要である。 |
| QA-005 | AC-4 | DEV HANDOVER の candidate に紐づく root `make check` PASS 証跡を確認し、実行対象が candidate commit と一致することを監査する。加えて candidate 上で `git diff --check` を一回実行して whitespace error がないことを確認する。 | `evidence-review` / root `make check` は既に DEV の candidate 証跡で確認し、同一の高コスト root suite を QA で重複実行しない。diff check は短時間・決定的な差分健全性検査として独立実行する。 |
| QA-006 | AC-5 | candidate commitをmain基準でscope auditする。変更パスが`Makefile`、`scripts/run-doc-lints.mjs`、`scripts/task/development-process.test.mjs`の3パスだけで、追加・変更行が約250行以内であること、dev-agent-harness製品コード、runtime、Schema、依存、生成物に変更がないことを確認する。 | `evidence-review` / 許可パス・変更規模・除外対象はcandidate-bound diffで直接監査できる。 |

実施モードは `evidence-review`、`focused-rerun`、`live-e2e` のいずれかとする。実施不能なケースは理由を記録し、別モードのPASSで置き換えない。

## 境界・異常・回帰

- 先頭 command が non-zero でも、残る2 command が順番どおり各一回起動される。first failure だけを返して後続を省略してはならない。
- 2件以上の command failure でも、全3 command 完了後の最終 status は non-zero である。failure の数によって実行回数を増減させない。
- 全3 command が成功した時だけ最終 status は zero である。
- command spawn 自体が error になっても、runner はその失敗を non-zero として集約し、未実行の後続 command を続ける。
- 子 command の stdout/stderr は capture・遅延表示ではなく親 stdio を継承する。固定 argv と shell 無効化を維持する。
- 既存検査の削除、緩和、並列化、cache、retry、自動修正、skip、changed-files 限定、新規 package・外部依存を回帰として扱う。
- 許可3パス以外、又は約250行を明らかに超える candidate は AC-5 failure として扱い、QA PASS を出さない。

## 実装後の再確認

- [ ] candidate commit 識別子と DEV の `make check` 証跡が一致することを確認した（QA では root `make check` を再実行しない）。
- [ ] QA-001 と QA-002 の bounded unit test を candidate 上で一回再実行し、各ケースの failure-detecting assertion を確認した。
- [ ] QA-003、QA-004、QA-006 の candidate-bound diff/evidence audit と QA-005 の `git diff --check` を完了した。
- [ ] 実装差分とレビュー結果を確認した。
- [ ] 操作手順と期待結果を現行実装に合わせた。
- [ ] 期待結果または範囲を変更した場合、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-firstの独立QA計画。Main確認でunit testを既存`test:process`列挙済みpathへ補正。 | `approved` |
