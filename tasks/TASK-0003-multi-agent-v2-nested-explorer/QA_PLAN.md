---
task_id: "TASK-0003"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-15"
revision: 2
implementation_reviewed_at: "2026-07-15"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0003 QA PLAN

## 方針

本Taskはdocs-onlyの起動ポリシー更新である。マージ済み`main`で、実装差分と6文書（product `AGENTS.md`、`docs/development/README.md`、`docs/development/agent-roles.md`、work repository `AGENTS.md`、`docs/glossary.yml`、生成済み`docs/99-glossary-index.md`）を、Task、承認済みPLAN、および製品側の`.codex/config.toml`/`.codex/agents/*.toml`という正規契約に照合する。主な合否根拠は文書上の契約と既存設定との一致であり、runtimeでの`agents.spawn_agent`実行は本Taskの合否条件にしない。

各ケースでは、該当文書の記述を引用位置付きで記録し、role/`agent_type`/model/effort、fallback条件、gate・Git・lock責務が互いに矛盾しないことを確認する。`make check`は既存の共通品質ゲートとして実行する。

## 前提と環境

- QAはDEVと独立した`qa` role（`gpt-5.6-terra` / `medium`）で実施し、マージ済み`main`を対象とする。
- Task、承認済みPLAN、レビュー結果、実装差分を確認してから実施する。期待結果または試験範囲を変更しない。
- 設定の照合元は製品側`.codex/config.toml`および`.codex/agents/{main,planner,qa,reviewer,dev-luna,dev-sol,explorer}.toml`とする。`hide_spawn_agent_metadata = false`、`tool_namespace = "agents"`、各roleのmodel/effort、Explorerの`read-only`・`max_depth = 0`・`max_threads = 1`を確認対象とする。
- 対象外のruntime、TOML、launcher、adapter、hook、lock、Task Schema、sandbox enforcementは変更されない前提で差分を確認する。`docs/glossary.yml`と`docs/99-glossary-index.md`は用語集inventory driftを解消するための承認済み機械更新であり、後者は`validate-terminology.py --write`で生成する。role TOMLの`workspace-write`/`read-only`は意図する契約であって、runtimeの実効sandbox観測の証明ではない。
- 実機spawn再検証は行わない。runtime観測が新たに提示されない限り、sandboxを保証済みと判定しない。

## 受け入れ条件との対応

| ケースID | 受け入れ条件 | 操作 | 期待結果 | 証跡 |
|---|---|---|---|---|
| QA-001 | 子Agentの標準起動経路が`agents.spawn_agent` | 6文書と差分を読み、通常の子role起動の主経路を検索する。 | product `AGENTS.md`、development文書、work `AGENTS.md`がnative `agents.spawn_agent`を主経路として整合して記載する。`make *-agent`を通常経路・必須・優先とする記述は残らない。 | 各文書の該当行、差分、検索結果。 |
| QA-002 | `task_name`/`agent_type`分離、異種roleのfork | 起動規約・例・表を照合する。 | `task_name`は固有識別、`agent_type`はrole選択と明記され、異種roleの起動には`fork_turns="none"`が必須である。`task_name`をrole selectorにする、または既定`all`に依存する経路はない。 | 該当規約・例の行番号、6文書横断の用語照合。 |
| QA-003 | roleとmodel/effortの対応 | 文書のrole表をconfig/TOMLに突合する。 | main=`main`/Sol-high、PLAN=`planner`/Terra-medium、QA=`qa`/Terra-medium、Reviewer=`reviewer`/Terra-medium、DEV Luna=`dev-luna`/Luna-xhigh、DEV Sol=`dev-sol`/Sol-high、Explorer=`explorer`/Luna-mediumが一致する。 | config/TOMLと各文書の対応表・記述の比較表。 |
| QA-004 | 順次gateとrole分離の維持 | Task/PLANと文書のプロセス記述を読む。 | PLAN→DEV→review→QAを一体ずつ順番に進めるgateが維持され、DEVとReviewer、DEVとQAが分離される。native spawn導入を理由に承認、統合、merge、FAIL分類を子へ移譲していない。 | 該当行と既存責務表の照合結果。 |
| QA-005 | Explorerの境界 | Explorerに関する全記述を照合する。 | Explorerは`explorer`、Luna/medium、一件の限定質問、read-only、編集・Git書込み・scope拡大・再委譲なし、短い根拠要約のみを維持する。複数質問、再委譲、実装を許す経路はない。 | Explorer節、TOML、検索結果。 |
| QA-006 | 子Git禁止と親のlock責務 | Git・運用証跡の規約と差分を照合する。 | 子はstage、commit、merge、`.git`書込みを行わず、lockを保持する親/mainがscope検査、hook、stage、commit、事後検査を所有する。native spawnへの変更でこの境界を緩和・移譲していない。 | product/work `AGENTS.md`、agent-rolesの該当行、差分。 |
| QA-007 | 限定されたfallback | fallback記述を6文書で検索し、PLANの三条件と比較する。 | fallbackは`agent_type`の欠落/利用不能、internal spawn不可、runtimeのmodel/effort不一致だけに限定される。該当時のみ`make work-agent`または一問専用`make explorer-agent`を使用し、運用証跡書込みでは親lock責務を維持する。 | fallback条件の一覧と文書別照合。 |
| QA-008 | mismatch時の停止・証跡化 | model/effort検査と異常時記述を読む。 | requested/observed model・effortとruntime条件を記録し、不一致なら子成果を採用せず停止する。別の`task_name`やrole名で回避・継続する記述はない。 | mismatch節、異常経路の照合結果。 |
| QA-009 | sandboxの観測限界 | sandboxの全記述を検索しTOMLとPLANに照合する。 | role TOMLは設計上のsandbox契約としてのみ扱われ、spawn metadataで未観測の実効sandboxを保証・証明・確認済みと断定しない。Git/lock境界をsandbox観測の有無から変更しない。 | sandbox記述の検索結果、TOMLとの対比。 |
| QA-010 | 文書整合と共通検査 | 対象6文書、Task、PLAN、設定を横断レビューし、`make check`を実行する。 | 役割名、primary/fallback、fork、gate、Explorer、Git/lock、mismatch、sandboxの各契約に矛盾がなく、承認済みの用語集機械更新以外に対象外ファイルの変更がない。`make check`がPASSする。 | 照合チェックリスト、`make check`の終了コードと出力要約。 |
| QA-011 | 用語集inventory driftの機械更新 | `docs/glossary.yml`を正本として確認し、`validate-terminology.py --write`を実行後に`docs/99-glossary-index.md`の差分と生成結果を確認して、`make check`を再実行する。 | glossaryとindexが用語inventoryとして整合し、indexは検証スクリプトの生成結果と一致する。生成済みindexに手編集の痕跡がなく、`--write`後に追加差分が生じない。再実行した`make check`がPASSする。 | 実行コマンド、終了コード、`--write`前後の差分、glossary/index照合、再実行ログ要約。 |

## 境界・異常・回帰

- `agent_type`が非公開/欠落、internal spawn不可、model/effort mismatchの各異常は、fallbackの正当な入口として文書化されているかを確認する。それ以外（慣例、都合、sandbox未観測、通常の証跡書込み）はfallback理由にならない。
- role名だけの一致、`task_name`による擬似的なrole選択、`fork_turns`省略、既定`all`の許容、mismatch後の成果採用はFAILとする。
- Explorerの質問数、read-only、再委譲禁止、深さ0のいずれかが緩和される回帰はFAILとする。
- 子のGit操作許可、親lock・scope検査・hook・commit責務の消失/子移譲、DEVとreviewer/QAの兼任許可、gate順序の緩和はFAILとする。
- runtime実効sandboxをTOMLだけで保証する表現はFAILとする。一方、未観測sandboxそのものはdocs-only TaskのFAIL条件ではなく、未保証として正しく表現されているかで判定する。
- `docs/99-glossary-index.md`を手編集した、または`validate-terminology.py --write`後に未反映差分が残る場合はFAILとする。glossary/indexの不整合は用語集inventory driftとして記録し、起因を差分、既存状態、または検査基盤へ分類する。
- `make check`失敗は、変更差分、既存不具合、環境/依存関係、検査基盤のいずれかに分類する。再現根拠なしにDEV不具合へ帰責しない。

## 実施不能時の扱い

- 文書・設定・差分のいずれかにアクセスできない場合は、未確認のケースID、阻害要因、必要な最小入力を`QA_RESULT.md`へ記録し、PASSとはしない。
- `make check`を実行できない、または失敗した場合は、コマンド、終了コード、要約、再現可否、差分との因果を記録する。分類不能ならmain Agentへ判断を委ね、DEV不具合と断定しない。
- 実機spawnが利用できなくても、本QA計画のdocs-only照合は継続する。実機未実施をruntime成功/実効sandbox保証の根拠にせず、追加のruntime検証が必要ならTask範囲外の追跡事項としてmain Agentへ報告する。

## 実装後の再確認

- [x] コミット`a4cefff`の実装差分と`REVIEW_RESULT.md`（PASS、`make check` PASS）を確認した。
- [x] native spawn、role契約、限定fallback、Explorer/Git/lock境界、sandbox未保証、および用語集生成の操作手順が現行実装と一致することを確認した。
- [x] 期待結果・試験範囲の変更はないことを確認した（`expectation_changed: false`を維持）。
- [x] 期待結果または範囲を変更していないため、main Agentの追加承認は不要であることを確認した。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-15 | QA Agent | TASK-0003のdocs-only起動契約に対する実装前QA計画 | `main-agent-sol-high` |
| 2 | 2026-07-15 | QA Agent | 用語集inventoryの機械更新・生成整合・`make check`再実行を追加（期待結果は不変） | `main-agent-sol-high` |
| 3 | 2026-07-15 | QA Agent | コミット`a4cefff`とPASSレビューに基づく実装後再確認（期待結果・試験範囲は不変） | `not required` |
