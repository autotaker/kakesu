---
task_id: "TASK-0003"
title: "MultiAgentV2の内部Spawn Agentを標準起動経路にする"
status: done
created_at: "2026-07-14"
---

# TASK-0003 MultiAgentV2の内部Spawn Agentを標準起動経路にする

## 目的

Codex内部の`agents.spawn_agent`で名前付きroleを選択できることを確認できたため、PLAN、DEV、REVIEW、QA、Explorerの標準起動経路を内部Spawn Agentへ切り替える。`make work-agent`と`make explorer-agent`は、内部起動を利用できない場合のfallbackとして残す。

## 背景

openai/codex Issue #32705では、MultiAgentV2において`agent_type`等のrouting入力が隠れる、`fork_turns`の既定値`all`では異種role overrideが拒否される、`task_name`はrole selectorではない、role固有sandboxが親permission profileで上書きされ得る、という相互作用が報告されている。

本リポジトリでは`hide_spawn_agent_metadata = false`と`tool_namespace = "agents"`により`agent_type`が公開され、`fork_turns = "none"`を明示した実機検証でPlanner、QA、Reviewer、DEV Luna、DEV Sol、およびPlanner配下のExplorerについて期待するmodel/effortが選択された。一方、実効sandboxは同じ証跡から確認できていないため、role TOMLの宣言だけを実効権限の証明とは扱わない。

## スコープ

### 対象

- product `AGENTS.md`へのsubagent起動手順の追加
- `docs/development/README.md`と`docs/development/agent-roles.md`の正本更新
- 必要な`agent-harness-work/AGENTS.md`の整合
- 新規用語を`docs/glossary.yml`と生成済み`docs/99-glossary-index.md`へ機械反映する用語整合
- roleと`agent_type`の対応、`task_name`との責務分離、`fork_turns = "none"`の明示
- model/effort不一致時の停止と、fallback条件の明文化
- Explorerの一質問、読み取り専用、再委譲禁止の維持
- 子AgentのGit禁止と、親のlock、scope検査、hook、commit責務の維持

### 対象外

- 製品コード、`.codex`設定、launcher、adapter、hook、lock、Task Schemaの変更
- MultiAgentV2 runtimeまたはsandbox enforcementの修正
- role TOMLだけを根拠とした実効sandboxの保証
- PLAN / DEV / REVIEW / QAのgate、独立性、mainの承認・merge責務の緩和
- `make *-agent` fallbackの削除

## 受け入れ条件

- [ ] 子Agentの標準起動経路が`agents.spawn_agent`であると`AGENTS.md`に明記されている。
- [ ] `task_name`は識別用、`agent_type`はrole選択用であり、異種roleでは`fork_turns = "none"`を明示する手順がある。
- [ ] Planner=`planner`、QA=`qa`、Reviewer=`reviewer`、DEV Luna=`dev-luna`、DEV Sol=`dev-sol`、Explorer=`explorer`の対応と期待model/effortが一致している。
- [ ] PLAN、DEV、REVIEW、QAを一体ずつ順番に起動し、DEVとReviewer、DEVとQAを分離する既存gateが維持されている。
- [ ] Explorerは一件の限定質問だけを扱い、ファイル編集、Git操作、scope拡大、再委譲を行わない。
- [ ] 子Agentはstage、commit、merge、`.git`書込みを行わず、mainまたは親がlock、scope検査、hook、commit、事後検査を所有する。
- [ ] `agent_type`が利用不能、内部Spawn Agentが利用不能、またはruntimeのmodel/effortがrole契約と不一致の場合だけ`make work-agent`または`make explorer-agent`へfallbackする。
- [ ] model/effort不一致時は子の成果を採用せず停止し、requested/observed値とruntime条件を証跡化する。
- [ ] sandboxは観測できた値だけを記録し、未観測の場合はrole TOMLだけで保証済みと表現しない。
- [ ] 対象文書間に矛盾がなく、`make check`がPASSする。
- [ ] `validate-terminology.py --write`で用語集と生成indexを更新し、inventory driftが残っていない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 承認済みPLANとQA計画に従って文書が更新されている。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] mainへの`--no-ff`マージ後にQAが完了している。
- [ ] HANDOVERがWiki Agentに取り込まれている。

## 関連コンテキスト

- openai/codex Issue #32705: MultiAgentV2のrouting入力、fork既定値、role選択、sandbox継承の既知問題
- [Codex Agent Model Routing](../../wiki/semantic/schemas/codex-agent-model-routing.md)
- [Task Delivery](../../wiki/semantic/scripts/task-delivery.md)
- [Work Repository Boundary](../../wiki/semantic/schemas/work-repository-boundary.md)
- [DECISION-0001 PLAN DEV QA Process](../../wiki/decisions/DECISION-0001-plan-dev-qa-process.md)
- [DECISION-0002 Work Repository and Wiki Ownership](../../wiki/decisions/DECISION-0002-work-repository-and-wiki-ownership.md)
- [DECISION-0003 Codex Agent Model Routing](../../wiki/decisions/DECISION-0003-codex-agent-model-routing.md)
