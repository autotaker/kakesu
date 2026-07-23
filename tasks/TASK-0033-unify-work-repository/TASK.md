---
task_id: "TASK-0033"
title: "運用リポジトリを製品リポジトリへ統合する"
status: dev
created_at: "2026-07-22"
---

# TASK-0033 運用リポジトリを製品リポジトリへ統合する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

`agent-harness-work`を独立運用する二重リポジトリ構成を廃止し、製品・Task証跡・Wiki・自動化を`agent-harness`へ統合する。証跡はmain worktreeへ直接かつ安全にcommit/pushし、コードはsparseなTask worktreeで変更し、独立REVIEW/QA後のPRをCI成功時にmerge commitでauto-mergeする一貫した運用を提供する。

### 対象と対象外

#### 対象

- `agent-harness-work`の最新スナップショットから`backlog.yaml`、`tasks/`、`wiki/`、`lap30/`、運用Schema、viewer、hooksと必要な運用定義を`agent-harness`へ移行する。
- `agent-harness`を唯一の正本とし、外部`WORK_ROOT`および`../agent-harness-work`への実行時依存を除去する。
- `make task-start`によるmain上のTask起票・commit/pushと、`worktrees/TASK-*`配下へのTask branch/sparse worktree作成を一括化する。
- Agentがmain worktreeのTask証跡を直接編集し、Main所有のaction別`evidence-commit`が許可scope、共通lock、軽量検査、commit、最大2回のrebase/push retryを所有する。
- コードworktreeからmain直管理pathをsparse-checkoutで除外し、Review/QAはworktreeのcandidateを評価しながら結果をmain worktreeの正本へ記録する。
- REVIEW/QA PASS後のready PR作成、merge commit方式のauto-merge、PR CI、merge後の冪等な状態更新を自動化する。
- ローカル`make sync`でmain同期、merge済みTaskのWiki取込、done化、worktree/branch cleanup、commit/pushを行い、`FAST=1`でAI Wiki取込を省略できるようにする。
- 移行完了後に`autotaker/kakesu-work`をarchiveし、履歴統合は要求しない。

#### 対象外

- `agent-harness-work`のGit履歴を`agent-harness`へ取り込むこと。
- OpenAI API従量課金、GitHub-hosted runnerへのChatGPT `auth.json`配置、self-hosted runnerの導入。
- GitHub Actions上でAI Wiki取込を実行すること。Wiki AgentはローカルCodex Pro認証を使う。
- 既存Done Task、公開済みWiki判断、Lap30 eventの意味を遡及変更すること。
- REVIEW/QAの独立性、Main所有の統合判断、固定roleのmodel/effort契約を緩和すること。

### 受け入れ条件

- [ ] AC-1: `agent-harness`の単一checkoutだけでbacklog、Task証跡、Wiki、Lap30、viewer、運用Schema/hookを検査・更新でき、実行コード・文書・設定に`../agent-harness-work`または外部`WORK_ROOT`を必須とする参照が残らない。
- [ ] AC-2: cleanなmainから`make task-start ID=... SLUG=... TITLE=...`を一度実行すると、Task/backlogの生成・検査・commit/pushと、`task/TASK-*` branchおよび`worktrees/TASK-*` sparse worktreeの作成が完了し、個別公開入口を誤用しても不完全な状態を残さない。
- [ ] AC-3: コードworktreeでは`backlog.yaml`、`tasks/**`、`wiki/**`、運用indexがcheckoutされず、Agentは明示されたmain rootのTask正本だけを更新する。action別`evidence-commit`は対象Taskの許可pathだけをcommitし、共有pathは専用actionに限定し、local lockとremote raceをfail-closedに処理する。
- [ ] AC-4: mainへの直接pushではread-only軽量CIが証跡scope、backlog/Task/Wiki Schema、リンクと状態遷移を検査し、製品path混入をFAILにする。CIはcommit/pushせず、書込みworkflowの再帰起動を生じない。
- [ ] AC-5: REVIEWとQAは同じPR管理対象pathのcandidate commit/digestから独立に評価され、両方PASS後だけ`make task-pr TASK=...`がbranchをpushしてready PRを作成し、merge commit方式のauto-mergeを有効化する。PRへmain直管理pathが混入した場合は拒否する。
- [ ] AC-6: PR CIはmerge候補に対する`make check`、対象Task検査、scope検査をrequired checkとして実行し、全て成功するまでauto-mergeしない。
- [ ] AC-7: `pull_request.closed`かつmergedの一回限りのpost-merge workflowだけが`merged_commit`と`merged`状態をmainへ記録し、記録済みTaskではno-opとなる。push CIは書き込まず、`workflow_run`連鎖を使わず、Task/PR単位concurrencyにより無限ループと重複commitを防ぐ。
- [ ] AC-8: `make sync`はmainの安全な同期、remote prune、merge済み未取込TaskのローカルCodex Wiki取込、done化、cleanなworktree/branchの削除、証跡commit/pushを冪等に行い、dirty/conflict/赤CIでは停止する。`FAST=1`はWiki取込とdone化を保留して同期だけを行う。
- [ ] AC-9: 既存32 Done Taskと進行中TASK-0033、backlog、Wiki、判断、Lap30、viewerの最新スナップショットが欠落や意味変更なく移行され、新旧双方の検査結果と件数/digestが照合された後に`autotaker/kakesu-work`がarchiveされる。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | product `main` | `e3f6da75b597372db484ff722c2ccb414bf802fa` | 統合先の開始点 |
| REF-2 | work repository `main` | `d030db5dc2974056387616d047197823b94602ce` | Task起票前の移行元スナップショット |
| REF-3 | public GitHub repository | `autotaker/kakesu`, default branch `main`, 2026-07-22 GitHub API確認 | PR/Actions/公開repoの安全境界 |
| REF-4 | 現行開発プロセス | product `e3f6da7`の`AGENTS.md`と`docs/development/*` | PLAN/DEV/REVIEW/QA、Main Git所有、現行完了checker |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| 既存Task | `ready` | TASK-0001〜TASK-0032はwork `d030db5`でdone | 移行時の件数・digest照合 |
| GitHub PR/auto-merge権限 | `ready` | 昇格環境で`gh auth status`、repository GraphQL、`gh repo edit`を実測 | actor `autotaker`、merge commit可、auto-merge有効。required contextは`Full check`、`Task check`、`Scope check`に固定しworkflow導入時にrulesetへ設定 |

### 許可パス

- `AGENTS.md`
- `Makefile`
- `.gitignore`
- `.github/**`
- `.githooks/**`
- `.agents/skills/run-efficient-task-delivery/**`
- `.codex/**`
- `backlog.yaml`
- `tasks/**`
- `wiki/**`
- `lap30/**`
- `viewer/**`
- `schemas/**`（既存製品Schemaとの衝突を避ける移行先をPLANで固定）
- `scripts/task/**`
- `templates/task/**`
- `docs/development/**`
- `docs/glossary.yml`
- `docs/99-glossary-index.md`
- `package.json`
- `pnpm-lock.yaml`
- `project.yaml`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | ready | `make task-preflight TASK=TASK-0033`を2026-07-22にexit 0で実測。現行`make task-check`、`make check`、`make work-check`と、統合後に追加する単一repo checkerをcandidate上で実行する |
| 権限 | ready | 2026-07-23に昇格環境で`gh auth status`がactor `autotaker`と`repo`/`workflow` scopeを確認。GraphQLでmerge commit可、`gh repo edit --enable-auto-merge`後に`autoMergeAllowed:true`を確認 |
| 依存状態と参照 | ready | REF-1〜REF-4、actor、merge方式、auto-merge、required context名を固定。rulesetの実設定とcheck runはQA-005〜QA-007のlive-e2eで確認する |
| 生成物の有無と更新方法 | ready | backlog viewer、Wiki index/receipt、glossary index、Task templates、workflowを決定的生成または固定入力で検査する。AI Wiki取込はlocal `make sync`のみ |
| 割当ワークツリー | ready | 現行work repoの`make worktree-create TASK=TASK-0033`で承認PLAN後にTask branch/worktreeを割り当て、統合後挙動はfixture内の一時repoで検証する |
| Lapログの書込・Schema・`repository annotation` | `not_started`（計測対象外） | ユーザーはLap計測を要求していない。既存Lap30 Schema/JSONLは移行digest照合対象とし遡及変更しない |

### 未決事項

- なし。運用Schemaの正規配置はPLANで`schemas/operations/`に固定した。

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- 2026-07-23 Main: sandbox内`gh auth status`の無効token表示はmacOS Keychain非公開による観測差であり、昇格環境ではactor `autotaker`の認証、`repo`/`workflow` scope、public `autotaker/kakesu`への管理権限を確認した。
- repositoryはmerge commit対応、auto-merge有効へ変更済み。required check contextを`Full check`、`Task check`、`Scope check`に固定し、workflow導入後にrulesetへ設定する。
- planning時のpending値がreadyになってもAC、設計scope、QAの期待または範囲は変わらない。PLAN/QA_PLANは同じ内容をMainが承認する。

### Cutover bootstrap clarification

- TASK-0033自身のmain管理証跡をコードPRへ混入させないため、DEVが移行validatorを固定した後、Mainは`backlog.yaml`、`tasks/**`、`wiki/**`、`lap30/**`、単一root用`project.yaml`を移行元digestへ照合し、製品mainへ証跡commitとして直接pushする。
- このbootstrap commit以降は製品mainを証跡正本とし、`agent-harness-work`への新規証跡書込みを停止する。Reviewer/QAはコードcandidate commit/digestとbootstrap evidence commit/digestの組を同一評価対象として独立に扱い、結果を製品mainへ記録する。
- Task branchはbootstrap後の製品mainを取り込み、PR差分からmain管理証跡を消す。scope検査にTASK-0033専用のPR例外は設けず、以後のPRと同じ禁止規則を適用する。
- これはAC-3、AC-5、AC-9の既存意図を実現する順序の明確化であり、受け入れ期待または範囲を変更しない。PLAN/QA_PLANはこの順序へ改訂・再承認してからDEVを開始する。

## 背景

製品と運用証跡が別Git repositoryに分かれているため、Agent routing adapter、共通lock、worktree lifecycle、二重push、merge後Wiki取込の調整が必要となり、通常のTask完了に過剰な運用コストが生じている。ユーザーとの要件確認で、証跡は単一repoのmain worktreeへ直接pushし、コードだけをTask worktree/PRへ分離する方針、軽量main CI、merge commit auto-merge、ローカルCodex認証によるWiki取込、無限ループ防止を合意した。

## 検討すべき設計観点

- trackedな証跡pathをコードworktreeから除外するper-worktree sparse-checkoutと、main root正本への明示routing。
- main Agentが保持する共通lock、action別allowlist、remote non-fast-forward時の最大2回rebase/recheck、競合時の安全な停止。
- 全repository tree一致に代わるPR管理対象pathのcandidate digestと、証跡更新を除外したREVIEW/QA/merge後同一性。
- `pull_request`、`push main`、`pull_request.closed` workflowの責務分離、冪等key、concurrency、bot commitの再帰防止。
- public repoかつChatGPT Proのため認証情報をGitHub Actionsへ置かず、AI Wiki取込をlocal syncへ限定する安全境界。
- 既存運用Schemaと製品Schema、`.codex`/`.agents`、hooks、package scriptsの正本統合と重複排除。
- 移行中の旧Make入口、`WORK_ROOT`、既存32 Task、Wiki receipt、Lap30、viewerの互換性とcutover/rollback。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、`merge_tree`確認、環境依存ケース、Wiki取り込みが完了している。
- [ ] 安全契約変更の場合: 独立計画レビュー、契約検査、`no-ff merge`、案/merge tree一致が完了し、製品REVIEW/QA PASSやWiki receiptを代用証跡として作成していない。

## 関連コンテキスト

### Semantic Wiki

- [Development Task](../../wiki/semantic/concepts/development-task.md): Task証跡を製品リポジトリ外へ保持する現行モデルを、単一repository内のmain正本とsparseコードworktreeへ移行する対象として見直す。
- [Work Repository Boundary](../../wiki/semantic/schemas/work-repository-boundary.md): 製品/運用の二重repository、`project.yaml`による相対参照、運用mainへのlock付き書込みという現行境界を置換する。
- [Task Delivery](../../wiki/semantic/scripts/task-delivery.md): PLAN→DEV→REVIEW→QAの独立gateを維持しつつ、Task branch/worktree、main統合、Wiki ingestの実行経路を単一repository向けに更新する。
- [Wiki Ingestion](../../wiki/semantic/scripts/wiki-ingestion.md): HANDOVER digestによる冪等取込、Decision不変性、Schema検査を維持し、local `make sync`での取込へ移す。
- [Safety Contract Completion Preflight](../../wiki/semantic/schemas/safety-contract-completion-preflight.md): 安全契約変更を選ぶ場合のplanned/generated path宣言とfail-closedな完了検査に従う。
- [QA FAIL Attribution](../../wiki/semantic/case-patterns/qa-fail-attribution.md): CI、認証、lockなどの環境依存失敗をDEV不具合と自動帰責せず、Mainが根拠に基づき差し戻し先を決める。

### Decision

- [DECISION-0001 PLAN DEV QA Process](../../wiki/decisions/DECISION-0001-plan-dev-qa-process.md): 独立PLAN、REVIEW、QAとMainの統合判断を維持する。
- [DECISION-0002 Work Repository and Wiki Ownership](../../wiki/decisions/DECISION-0002-work-repository-and-wiki-ownership.md): 本Taskが置換対象とする、`agent-harness-work`を独立正本とする現行判断。置換時もWiki/Schemaの所有分離、lock、digest付きingestという安全性を保持する。
- [DECISION-0004 MultiAgentV2 Role Startup](../../wiki/decisions/DECISION-0004-multiagentv2-role-startup.md): `agent_type`によるrole選択、cross-roleの`fork_turns="none"`、親のlock/scope/hook/commit所有を維持する。

### 適用しなかった重要なDecision

- [DECISION-0003 Codex Agent Model Routing](../../wiki/decisions/DECISION-0003-codex-agent-model-routing.md) はDECISION-0004によりsupersededであり、二重root adapterの設計根拠としてのみ参照する。旧Decisionを復活させず、移行後は外部work repository adapterを廃止する。
