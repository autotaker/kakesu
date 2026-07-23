---
task_id: "TASK-0003"
status: complete
completed_at: "2026-07-15"
---

# TASK-0003 HANDOVER

## 成果

- 子Agentの標準起動経路を内部`agents.spawn_agent`へ統一し、`make work-agent`と`make explorer-agent`を限定的な`fallback`として維持した。
- 製品側`AGENTS.md`、`docs/development/README.md`、`docs/development/agent-roles.md`、用語集inventory、および運用リポジトリ側`AGENTS.md`の契約を整合させた。
- 製品merge commit: `e106a1eb2efe018a19ec18a3f5c7fc9a1a3bac78`

## 主要な変更

- `task_name`は起動を追跡する識別子、`agent_type`はロール選択値として責務を分離した。
- 呼出元と異なるロールの起動では`fork_turns="none"`を必須とし、既定の`all`によるロール上書き拒否を回避する契約を明記した。
- PLAN、DEV、REVIEW、QA、Explorerのロールと期待するmodel/effortの対応を明記した。
- 起動後のmodel/effortが契約と不一致の場合は子の成果を採用せず停止し、requested/observed値とランタイム条件を証跡化してから`fallback`可否を判断する契約を追加した。
- `Explorer`は一件の限定質問、読み取り専用、再委譲禁止とした。
- 子Agentのstage、commit、merge、`.git`書込みを禁止し、親が共通ロック、スコープ検査、hook、stage、commit、事後検査を所有する境界を維持した。

## 検証結果

- 独立レビュー: PASS
- `make check`: PASS
- マージ後QA: `QA-001`〜`QA-011`すべてPASS

## 判断

- openai/codex Issue #32705で報告された、`agent_type`等の起動メタデータが隠れ得ること、`fork_turns`既定値`all`では異種ロール上書きが拒否され得ること、`task_name`はロール選択値ではないこと、ロール固有sandboxが親のpermission profileで上書きされ得ることを背景とした。
- 実機で利用可能な`agents.spawn_agent`を主経路とし、`agent_type`の欠落、内部`Spawn Agent`の利用不能、またはmodel/effort不一致の場合だけ、停止・証跡化後に`make *-agent`への`fallback`可否を判断する。
- PLAN→DEV→REVIEW→QAを一体ずつ順次進め、DEVとReviewer、DEVとQAを分離する既存ゲートは変更しない。

## 既知の制約と未解決事項

- 実効sandboxは未観測である。`.codex/agents/*.toml`の`sandbox_mode`は意図するロール契約だが、TOMLの宣言だけではランタイムの実効権限を保証しない。

## 運用上の注意

- 起動時は`task_name`をロール選択に流用せず、`agent_type`と異種ロール用の`fork_turns="none"`を明示する。
- model/effort不一致時は子の成果を採用して継続せず、requested/observed値とランタイム条件を記録する。
- sandboxの観測有無にかかわらず、子Git禁止と親のロック・コミット責務を緩和しない。

## Wikiへ引き渡す知識

### 再利用可能な知識

- MultiAgentV2では`task_name`は識別、`agent_type`はロール選択という別の責務を持つ。
- 異種ロール起動で確実に名前付きロールのmodel/effort契約を適用するには、`fork_turns="none"`を明示する。
- 内部`agents.spawn_agent`を標準経路にしても、model/effortの実測照合、ロール分離、順次ゲート、Git・ロック境界は独立した不変条件として維持する。
- ランタイムで観測できないsandbox属性は、設定ファイルの宣言だけで保証済みと扱わない。

### 反例・失敗・注意点

- `task_name`をロール選択値として扱うと、意図したモデルや推論effortを選択できない。
- 異種ロール起動で`fork_turns`を省略して既定値`all`を使うと、ロール上書きが拒否され得る。
- 子Agentの自己申告やロールTOMLだけを、model/effortまたは実効sandboxの観測証跡として扱わない。
- `make *-agent`を通常経路へ戻すと、内部`agents.spawn_agent`を主経路とする本判断と矛盾する。

### 更新候補ページ

- `wiki/decisions/DECISION-0003-codex-agent-model-routing.md`: 内部`agents.spawn_agent`を標準起動経路とする判断、`task_name`/`agent_type`の責務分離、異種ロールの`fork_turns="none"`、model/effort不一致時の停止・証跡化、限定`fallback`、sandboxの観測限界を追記する候補。
- Codex Agent Model Routingの意味Wiki: ロール対応表、Explorerの一問/read-only/no redelegation契約、子Git禁止と親ロック責務を再利用可能な契約として同期する候補。

## ブートストラップ例外

- 該当なし
