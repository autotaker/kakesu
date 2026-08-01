---
task_id: "TASK-0059"
change_class: "product"
status: "approved"
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "Makefile、実行script、process test、role registryという開発workflow toolingの限定変更であり、Kakesu product runtimeとtools/dev-agent-harness runtimeを変更しないため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T22:44:08Z"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T22:44:08Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-01T22:44:08Z"
classification_approval_reason: "Makefile、実行script、process test、.codex role registryという外部観測可能な開発workflow toolingを変更するため。"
---

# PLAN: Wiki Agentを標準spawn経路へ統合する

## 根拠と分類

本計画の唯一の要求根拠は`TASK.md`の`Planning input packet`である。Makefile、実行script、process test、`.codex` role registryという外部観測可能な開発workflow toolingの挙動を変更するため、`change_class`は`product`とする。Kakesu product runtimeとtools/dev-agent-harness runtimeは変更しない。

依存は`ready`（なし）で、dependency-ready reconciliationはN/Aである。新rule、wrapper、token、version、checklistは導入しない。既存v2形式の項目も追加しない。

## 変更境界

変更候補は次の許可pathに厳密に限定する。

- `Makefile`
- `scripts/task/run-wiki-agent.mjs`（削除）
- `scripts/task/agent-routing.mjs`
- `scripts/task/development-process.test.mjs`
- `scripts/task/unified-lifecycle.test.mjs`
- `.codex/config.toml`
- `.codex/agents/wiki.toml`（新規）
- `AGENTS.md`
- `wiki/AGENTS.md`
- `docs/development/agent-roles.md`

既存TASK-0058のWiki差分、既存receipt/Decision、receipt Schema、Wiki本文は変更しない。`wiki/AGENTS.md`はcandidate外のMain管理差分としてcompletion transactionで統合する。許可path外の差分、Kakesu product runtime差分、tools/dev-agent-harness runtime差分は受け入れない。

## 実施設計

1. 独立Wiki launcherとMake入口を除去する。
   - `scripts/task/run-wiki-agent.mjs`を削除する。
   - `Makefile`から`wiki-context`、`wiki-ingest`、`WIKI_PROFILE`、`WIKI_MODEL`、`WIKI_EFFORT`および`legacy-wiki`経路を取り除く。
   - 既存の一般的なWiki索引・検証契約は、入力パケットに反しない限り保持する。

2. 正規role registryへ標準Wiki roleを追加し、唯一の起動経路を定義する。
   - `.codex/config.toml`と新規`.codex/agents/wiki.toml`に、`wiki` role、Terra/medium、workspace-writeを宣言する。
   - role契約は編集専用、別Agent起動禁止、stage/commit/mergeおよび`.git`書込み禁止とする。
   - `scripts/task/agent-routing.mjs`と統制文書を、Mainが`agents.spawn_agent(task_name=..., agent_type="wiki", fork_turns="none", ...)`で直接起動する唯一の経路へ整合させる。Wiki専用`codex exec`、独立launcher、暗黙起動を残さない。

3. Main所有と子Agent境界を文書化する。
   - `AGENTS.md`、`wiki/AGENTS.md`、`docs/development/agent-roles.md`を同一契約へ同期する。
   - MainはWiki依頼の対象と許可pathを起動前に固定し、同時writerを作らず直列化する。終了後は差分scope、索引変更時の`make wiki-index`、`make work-check`を確認し、既存の共通lock付きpublish transactionだけでcommitする。
   - Wiki Agentにはlock、validation、scope判定、stage、commitを割り当てない。
   - Wiki依頼がないTaskはWiki Agentもreceiptもなしに完了可能と明記する。receiptを作る場合の既存Schema・検証は残すが、完了条件、暗黙処理、専用承認フローへ追加しない。

4. focused process testsを削除後の契約へ合わせる。
   - `development-process.test.mjs`と`unified-lifecycle.test.mjs`からlauncher固有fixture・期待値を除去する。
   - 正規`wiki` roleの存在と属性、launcher・Make入口・Wiki専用`codex exec`・legacy変数/経路の不在、親Mainだけの起動/lock/検証/Git所有境界、receipt任意性を検出する既存process testを更新する。
   - receiptの作成自体を必須化するテスト、または新しいwrapper/token/version/checklistを要求するテストは追加しない。
   - `wiki/AGENTS.md`はcandidateに含めず、同一candidateの独立REVIEW/QA後にMainがcompletion transactionで統合する既存境界を維持する。

## 受け入れ条件への対応

| AC | 実施・検証の対応 |
|---|---|
| AC-1 | launcher削除、Make入口・legacy変数/経路の削除、および実行・設定pathでの不在を検出するfocused process test。Task証跡内の削除理由は対象外。 |
| AC-2 | `.codex`のwiki roleとrole契約、標準spawnを唯一の起動経路とする規約・routingの同期。 |
| AC-3 | Mainの直列writer、事前scope固定、終了後検査、既存lock付きtransaction所有を統制文書とprocess testで確認。 |
| AC-4 | receipt Schema/検証を維持し、Wiki未依頼Taskのreceipt・Agent非必須性を文書とprocess testで確認。 |
| AC-5 | 影響Node tests、docs lint、`make work-check`、root `make check`、`git diff --check`を実行する。 |
| AC-6 | `wiki/AGENTS.md`をcandidate外のMain管理差分としてcompletion transactionで統合すること、許可10 pathsのみであること、TASK-0058・Kakesu product runtime・tools/dev-agent-harness runtimeに差分がないことを差分で監査する。 |

## 検証計画

製品変更として、DEV開始前に承認済みPLANと独立`QA_PLAN.md`を用意する。DEVは製品差分だけのcandidateを一度作成し、同一candidateから独立REVIEWとQAを実施する。Mainだけがcompletion transactionとmain統合を所有する。完了時は許可path差分、`git diff --check`、および以下を実行する。

- 影響するNode process tests（`development-process.test.mjs`、`unified-lifecycle.test.mjs`）
- docs lint
- `make work-check`
- root `make check`
- `git diff --check`

テストは、削除対象文字列・入口が残る回帰、role属性の欠落、子へのGit/lock/validation移譲、receiptの必須化をそれぞれ失敗として検出できることを確認する。candidateは`wiki/AGENTS.md`を含めず、Main管理差分としてcompletion transactionで統合する。索引対象の変更が生じた場合だけ`make wiki-index`を実行する。失敗はQAガイドラインに従い、実装不具合と即断せず分類する。

## リスクと復旧

- 削除漏れにより例外起動経路が残るリスクは、不在assertionと全許可pathの文字列監査で抑える。
- role登録だけでMain所有境界が曖昧になるリスクは、role、routing、3統制文書、process testsを同じ所有権文言へ同期して抑える。
- receiptを誤って必須化または弱体化するリスクは、既存Schema/検証を変更せず、任意性をnegative caseで検出して抑える。
- 想定外のKakesu product runtimeまたはtools/dev-agent-harness runtime差分、TASK-0058への遡及変更が見つかった場合は、実装を停止し、Mainが範囲を再評価する。

復旧は、candidate製品差分とcandidate外の`wiki/AGENTS.md` Main管理差分を区別して許可path単位で戻し、既存の標準role registry・Main transaction・receipt任意契約へ復元したうえで、同じfocused process testsと製品変更検査を再実行する。公開済みreceipt/Decisionは書換えない。

## 引き継ぎ条件

実装担当は承認済み本PLANと独立QA_PLANの後に、candidateへ含める許可9 pathsだけで実施する。`wiki/AGENTS.md`はcandidate外のMain管理差分としてcompletion transactionで統合する。子Agentはstage、commit、merge、`.git`書込みを行わない。Mainだけがcandidate固定、既存lock付きtransactionでのcommit、独立REVIEW/QA、完了ゲートを所有する。
