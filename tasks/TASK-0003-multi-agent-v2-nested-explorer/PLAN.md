---
task_id: "TASK-0003"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-15"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "変更は起動ポリシーを説明する文書だけであり、runtime、TOML、launcher、hook、lock、Git権限を変更しない。対象文書の既存契約へ指定済みの実機観測を整合させる、局所的で機械レビュー可能な編集であるため。"
approved_dev_profile_risk_signals: []
planned_implementation_files: 6
planned_implementation_lines: 220
estimate_points: 2
---

# TASK-0003 PLAN

## 再設計の結論

本Taskは MultiAgentV2 の実装、adapter、runtime evidence、sandbox 制御を変更しない **docs-only の起動ポリシー更新** とする。既存の製品設定では `hide_spawn_agent_metadata = false` と `tool_namespace = "agents"` により `agents.spawn_agent` の `agent_type` が利用可能であり、実機で `planner` / `qa` / `reviewer` / `dev-luna` / `dev-sol` と、`planner` からの `explorer` の model/effort 選択が確認済みである。この観測を、全ロールに適用する明示的な起動契約として文書化する。

`sandbox_mode` は起動メタデータとして観測できない。そのため `.codex/agents/*.toml` の sandbox 指定は設計上の契約として参照できても、実効 sandbox を本Taskで保証・証明したとは扱わない。runtime がその契約を露出・検査できるようにする変更は対象外である。

## DEV profile選定証跡

```yaml
profile: luna-xhigh
model: gpt-5.6-luna
reasoning_effort: xhigh
decision_rule: "runtime・権限境界・設定生成を変更しない局所的な docs-only Task は luna-xhigh を使う"
reason: "4つのポリシー文書と、そこから検出される用語集・生成indexへ同一の起動契約を反映するだけであり、コード、TOML、script、lock、hook、Git責務を変更しない"
```

DEV は文書間の既存契約が矛盾する、または記載に runtime/configuration の変更が必要だと判明した時点で停止し、main Agent に `sol-high` への再評価とTask再設計を報告する。DEV はこのPLANの範囲で sandbox の保証表現を追加してはならない。

## 受け入れ条件の具体化

| 条件 | 完了時の文書上の観測方法 |
|---|---|
| 主ランチャー | product `AGENTS.md` と development 文書が、子Agent起動では `agents.spawn_agent` を主ランチャーと明記する。 |
| role選択 | `task_name` は一意な識別子、`agent_type` は role selector であることを明記し、role→agent_type の対応表を置く。 |
| 異種roleのfork | 呼出元と異なる role を起動する時は `fork_turns="none"` を必須とし、理由を role override の runtime 拒否回避として記載する。 |
| Explorer | 各 role の Explorer は `agent_type: "explorer"`、一件の限定質問、read-only 調査、再委譲なし、短い根拠要約のみ、という既存契約を維持して記載する。 |
| gateとGit境界 | PLAN→DEV→review→QA の順次 gate、DEV/reviewer/QA 分離、子の stage/commit/merge/`.git` 書込み禁止、親の lock/scope/hook/commit 責務が、起動方式変更後も不変と明記される。 |
| fallback | `agent_type` が利用不能、internal `spawn_agent` が利用不能、または runtime の model/effort が指定roleと不整合の場合に限り、`make work-agent` または `make explorer-agent` を fallback にする。通常経路で make launcher を必須・優先と記載しない。 |
| mismatch時の扱い | 起動結果で期待する model/effort と異なる場合は、その子の成果を採用せず、実行を停止して role、requested/observed model・effort、runtime 条件を証跡化する、と記載する。sandbox は未観測なら未保証として同じ証跡に記録する。 |
| 文書整合 | `AGENTS.md`、`docs/development/README.md`、`docs/development/agent-roles.md`、必要な work repository `AGENTS.md` が矛盾しない。既存の `make work-agent` の親lock・scope検査・hook・commit責務は削除・緩和しない。 |
| 用語整合 | 新規・更新された用語は `uv run --project memory python scripts/validate-terminology.py --write` により `docs/glossary.yml` と `docs/99-glossary-index.md` へ機械反映され、用語検証でinventory driftが残らない。 |

## 設計

### 正規の起動ポリシー

1. main Agent はTask、承認済みPLAN/QA計画、順次gate、分離要件を確認して role を選ぶ。
2. 子を起動する時、`task_name` は人間とruntimeが追跡する固有名であり、roleを選ぶ値には使わない。`agent_type` に正規roleを指定する。
3. 異種roleでは必ず `fork_turns="none"` を渡す。これにより親の model/effort 文脈を継承して role override が拒否される既知の挙動を避ける。
4. 子が返す model/effort を、選択した role の `.codex/agents/*.toml` 契約と照合する。一致しなければ停止・証跡化し、代替のrole名や task_name による回避を試みない。
5. `agent_type` 非公開/欠落、internal spawn 未提供、または前項のruntime不整合時だけ、既存の `make work-agent TASK=... ACTION=...` または一問専用の `make explorer-agent QUESTION=...` を fallback として利用する。運用リポジトリに書く fallback は従来どおり lock を保持する親が実行する。

| responsibility | `agent_type` | model / effort | 起動上の注意 |
|---|---|---|---|
| main | `main` | Sol / high | 承認・統合・mergeを子へ委譲しない。 |
| PLAN | `planner` | Terra / medium | PLAN evidenceのみ。 |
| DEV (低risk) | `dev-luna` | Luna / xhigh | 承認済み luna-xhigh PLANのみ。 |
| DEV (高risk/不明) | `dev-sol` | Sol / high | 承認済み sol-high PLANのみ。 |
| reviewer | `reviewer` | Terra / medium | DEVから独立。 |
| QA | `qa` | Terra / medium | DEVから独立。 |
| bounded research | `explorer` | Luna / medium | 一問、read-only、no redelegation。 |

role TOML の `sandbox_mode` は意図する role contract である。spawn metadata に sandbox がない現状では、その値を実効権限の証明と表現しない。子の Git禁止と親だけが持つ lock、scope検査、hook、stage、commit、rollbackの責務を、sandbox 観測の有無から推論して変更しない。

### 代替案と不採用理由

- `task_name` を role selector とする案は runtime の role選択契約ではないため不採用。
- `fork_turns` 既定の `all` で異種roleを起動する案は role override が拒否され得るため不採用。
- 常に `make *-agent` を主入口に残す案は、利用可能な `agents.spawn_agent` と実機検証済みの role選択を活かせず、ユーザー要望に反するため不採用。ただしfallbackとして維持する。
- sandbox を role TOML だけで実効保証とする案は観測証拠がないため不採用。
- 子へGit、commit、merge、lockを渡す案は既存の責務境界に反するため不採用。

## 変更予定

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `AGENTS.md` | documentation / policy | 50 | subagent の標準起動例、`task_name`/`agent_type` の意味、異種roleの `fork_turns="none"`、Explorer、一致検査・停止・fallback、既存Git境界を追記する。 |
| `docs/development/README.md` | documentation | 30 | 開発プロセスの入口として、native spawn優先と例外的fallback、および順次gateが変わらないことを短く案内する。 |
| `docs/development/agent-roles.md` | documentation / policy | 80 | role→agent_type対応、呼出し契約、Explorer制約、model/effort mismatch証跡、sandboxの観測限界、work launcherとの責務分担を正本として詳記する。既存の「Explorerは明示launcherのみ」の文言を native spawn優先・fallback launcherへ置換する。 |
| `../agent-harness-work/AGENTS.md` | documentation / policy | 20 | 運用リポジトリに固有の起動案内が存在する場合に限り、product側ポリシーへの参照と、証跡書込み時の `make work-agent` fallback/親lock責務を整合させる。固有の重複ポリシーがない場合は変更しない。 |
| `docs/glossary.yml` | generated documentation inventory | 15 | 新規tokenと既存用語のinventoryを、指定の用語検証スクリプトの `--write` 出力としてのみ更新する。直接編集しない。 |
| `docs/99-glossary-index.md` | generated documentation index | 25 | `docs/glossary.yml` と同じ `--write` 実行で再生成する。直接編集しない。 |

見積もり対象は6文書、最大220行である。`file_score = ceil(6 / 3) = 2`、`line_score = ceil(220 / 200) = 2` のため、許容scaleの最小値 `2` を維持する。用語集とindexは生成文書であり、既存スクリプト以外で編集しない。テストコード、TOML、launcher、adapter、hook、work Task証跡は変更しない。

## 実装手順

1. 現在の product `AGENTS.md`、開発README、Agent責務文書、`.codex/config.toml`、各role TOMLを照合し、role名・model・effort・Explorer制約を転記対象として確定する。
2. product `AGENTS.md` に、native `agents.spawn_agent` の呼出し規約と、異種role / Explorer / mismatch / fallback / Git境界を追記する。
3. READMEには詳細規約へのリンクと、native優先・make fallback・既存gate不変だけを記し、規約の複製を最小化する。
4. `agent-roles.md` を主たる詳細契約として更新し、古い「明示launcherのみ」の表現を置換する。sandboxについては「role TOMLの意図する設定であり、今回の観測では実効保証でない」と明記する。
5. work repository `AGENTS.md` に起動方法の重複規約がある場合だけ、運用証跡書込みの lock所有親・scope/hook/commit責務を保持したfallback説明へ整合させる。
6. 用語集を直接編集せず、`uv run --project memory python scripts/validate-terminology.py --write` を一度実行して、`docs/glossary.yml` と `docs/99-glossary-index.md` を機械更新する。
7. DEV は差分を読み、用語、role表、fallback条件、禁止事項が全対象文書で一致することを確認する。既存検査がdocsを対象にする範囲で `make check` を実行し、失敗は文書変更との因果を分けて記録する。

## QA計画可能な検証手順

1. 文書レビューで、標準経路が `agents.spawn_agent`、role selectorが `agent_type`、識別子が `task_name` と明示されていることを確認する。
2. 表または例で、main/planner/qa/reviewer/dev-luna/dev-sol/explorer の正規 `agent_type` と契約model/effortを照合する。
3. 異種role例すべてが `fork_turns="none"` を必須とし、Explorerが一問/no redelegation/read-onlyであることを確認する。
4. `agent_type` 欠落、internal spawn不可、model/effort mismatch の三条件以外で make launcherへfallbackしていないこと、および mismatch時に停止・証跡化すると書かれていることを確認する。
5. sandboxをmetadataまたはTOMLだけで保証したという断定がなく、未観測であることと既存の子Git禁止・親lock責務が明記されていることを確認する。
6. PLAN→DEV→review→QA、独立性、子Git禁止、親のscope/hook/commit責務に緩和がないことを確認する。
7. `uv run --project memory python scripts/validate-terminology.py --write` の後に同じ検証を実行し、新規tokenとinventory driftがないこと、`docs/glossary.yml` と `docs/99-glossary-index.md` が生成出力以外で編集されていないことを確認する。
8. `make check` を実行し、docs-only差分で既存品質ゲートを通ることを確認する。失敗時はQAガイドラインに従い文書・環境・既存不具合を分類し、DEV不具合と自動帰責しない。

## 不変条件・対象外

- 製品コード、`.codex/config.toml`、`.codex/agents/*.toml`、launcher、adapter、digest、hook、lock、Task schemaは変更しない。
- `docs/glossary.yml` と `docs/99-glossary-index.md` は手編集せず、用語検証スクリプトの生成出力以外では変更しない。
- sandboxのruntime enforcementやmetadata露出を追加しない。
- make launcherの親lock、scope検査、hook、stage、commit、rollback責務を削除・子へ移譲しない。
- mainだけが承認、commit、merge、FAIL分類を所有する。DEV、reviewer、QAの分離と既存gateを変更しない。
- Explorerは一件の限定したread-only質問だけを扱い、実装、方針決定、再委譲をしない。

## main Agentレビュー

- [x] docs-only への再設計と対象外が妥当である。
- [x] native spawn優先、`agent_type` role選択、`task_name`識別、異種role `fork_turns="none"` が明確である。
- [x] fallback、mismatch停止、sandbox観測限界、Git/lock責務が検証可能に記述されている。
- [x] QA計画を作成できる。
- [x] DEV profile `luna-xhigh`、見積もり2 pointsを承認した。
- [x] DEV開始を承認した。
