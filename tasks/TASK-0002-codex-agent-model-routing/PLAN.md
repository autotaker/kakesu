---
task_id: "TASK-0002"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-14"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "launcher、sandbox/commit所有権、兄弟2 repository root、project設定、起動証跡、テストを横断し、権限境界の契約変更を伴うため"
approved_dev_profile_risk_signals:
  - cross_cutting
  - sandbox_commit_ownership
  - dual_repository_root
  - configuration_contract
  - process_evidence
planned_implementation_files: 15
planned_implementation_lines: 1086
estimate_points: 8
---

# TASK-0002 PLAN

## DEV profile選定証跡

```yaml
profile: sol-high
model: gpt-5.6-sol
reasoning_effort: high
decision_rule: "高リスクsignalが一つでもある、横断的、または判定不能ならsol-high"
reason: "launcher、sandbox/commit所有権、兄弟2 root、設定、証跡、テストを横断する契約・権限境界変更"
```

`luna-xhigh` は、対象が明確かつ局所的で、機械的に検証可能であり、契約、Schema、security、concurrency、migrationのいずれにも影響せず、曖昧さもない場合だけ選べる。いずれかのsignal、横断性、または不確実性があれば決定的に `sol-high` とする。DEV中に新しいsignalを発見した場合、DEVは停止し、変更前profile、`sol-high`、signal、理由、main承認者・時刻をPLANのpromotion履歴へ追記して承認を受けてから再起動する。降格はしない。

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| 固定role routing | routing試験でmain=`gpt-5.6-sol/high`、planner/qa/reviewer=`gpt-5.6-terra/medium`、explorer=`gpt-5.6-luna/medium`、DEVの両profileを完全一致で確認する。 | product canonical role TOML |
| DEV routingとPLAN証跡 | `check-task` がprofile、reason、risk signals、promotion履歴を検査する。TASK-0002は上記 `sol-high` を使う。 | PLAN frontmatterと選定証跡 |
| nestingとexplorer | main/root はexplorerを直接起動できる。root→role→explorer は深さ2として許可し、role→explorer は第二階層を維持する。root→explorer は深さ1として許可する。explorer→child、深さ3、複数質問、child spawnは起動前FAILする。 | `agents.max_depth = 2`、launcher policy、試験 |
| explorer権限 | explorerは `sandbox_mode = "read-only"`、write scopeなし、commit/spawn/scope拡大禁止、bounded question一件、要約とファイル参照のみの固定契約である。 | explorer TOML、固定prompt、dry-run/evidence試験 |
| 両rootの設定 | productのcanonical role contractsと、main-owned生成adapterを両rootから解決する。adapter生成後のdrift validationがmodel、effort、sandbox、depth、spawn制約、role名の完全一致を証明する。 | generation/governance flowとcontract試験 |
| parent-owned commit | 子stdinはclosed。子成功後だけ、lock保持中の親がdirty scopeを検査、stage、hook付きcommit、再検査する。子に`.git`書込みを与えない。 | process integration試験 |
| fail-closed互換性 | fixed roleへの`PROFILE`/`MODEL`/effort上書きはcanonical値と一致しなければ起動前FAILする。一致値だけ許容する。文書化済みのnon-role legacy/Wiki経路だけを維持し、role固定を経由しないことを正常・異常系で試験する。 | argument validation試験 |
| concise launch evidence | dry-runと実行でrole/profile、model、effort、cwd、sandbox/write scope、allowed paths、child exit、親commit（なしも含む）を一行JSONで出力し、秘密値・raw logは保存しない。 | launch evidence試験 |

## 関連Wikiと判断

- [Development Task](../../wiki/semantic/concepts/development-task.md) と [Task Delivery](../../wiki/semantic/scripts/task-delivery.md) に従い、routingはPLAN承認、実装前QA計画、独立review、main Agentだけの`--no-ff` merge、マージ後QAという既存gateを変更しない。`Development Task`がいうtopic branch/worktreeは製品repositoryの実装作業場所を指し、運用repositoryの`main`一本・ロック直列化と混同しない。
- [Work Repository Boundary](../../wiki/semantic/schemas/work-repository-boundary.md) と [DECISION-0002 Work Repository and Wiki Ownership](../../wiki/decisions/DECISION-0002-work-repository-and-wiki-ownership.md) に従い、製品コードとcanonical role contractは`agent-harness`、Task証跡は`agent-harness-work`に置く。work側adapterはmain Agentが共通lock下で所有する別commitとし、DEVまたはWiki Agentへ設定・証跡所有権を移さない。
- [DECISION-0001 PLAN DEV QA Process](../../wiki/decisions/DECISION-0001-plan-dev-qa-process.md) の責務分離は、固定routing、parent-owned commit、限定read-only explorerによって強化するが、main Agentの承認・merge判断をrole Agentまたはexplorerへ委譲しない。routing試験またはマージ後QAのFAILは [QA FAIL Attribution](../../wiki/semantic/case-patterns/qa-fail-attribution.md) に従って分類し、DEVへ自動帰責しない。
- [Wiki Ingestion](../../wiki/semantic/scripts/wiki-ingestion.md) はQA完了後に`HANDOVER.md`をdigest付きで取り込む終了手順であり、PLAN作成時には適用しない。本TaskでSemantic Wiki、Decision、Schemaを変更しない。

### 競合と実装時検証

- accepted Decisionとの直接の競合はない。DECISION-0001は役割とgateを固定し、DECISION-0002はrepository・Wikiの所有境界を固定するため、本設計はその内側で起動contractを追加する。
- product実装は決定的な`make work-config-sync`（または同等名のtarget）を追加する。productの`--no-ff` merge後、post-merge QA前にmain Agentがこれを実行し、生成されたwork-repository adapterを共通lock下で`ACTION=governance`により別commitする。bootstrap中のTASK-0002 Agentは、すでに検証済みの一時的な明示profileを継続して用いる。未文書のCodex TOML include機構やglobal設定へ依存しない。
- `agents.max_threads = 2` と`max_depth = 2`は、root→explorerおよびroot→role→explorerの統合試験で検証する。CLIの意味論がこのcontractと不一致ならreview前にFAILし、実装で修正する。未解決のPLAN判断として先送りしない。
- DEV開始に未解決の設計判断は残らない。

## 設計

### 決定した構成

製品repositoryの `.codex/config.toml` と `.codex/agents/*.toml` を唯一のcanonical custom-agent configurationとする。`.codex/config.toml` は `agents.max_depth = 2` と、root→role→explorer chainを保証する保守的な最小値 `agents.max_threads = 2` を明示し、global config、未文書include key、未文書model aliasを参照しない。role contractは次の実在するproduct artifactで明示する。

| artifact | 固定contract |
|---|---|
| `../agent-harness/.codex/config.toml` | project settings、`agents.max_depth = 2`、`agents.max_threads = 2`、明示role registry |
| `../agent-harness/.codex/agents/main.toml` | `gpt-5.6-sol` / `high`、root orchestration |
| `../agent-harness/.codex/agents/planner.toml` | `gpt-5.6-terra` / `medium` |
| `../agent-harness/.codex/agents/qa.toml` | `gpt-5.6-terra` / `medium` |
| `../agent-harness/.codex/agents/reviewer.toml` | `gpt-5.6-terra` / `medium` |
| `../agent-harness/.codex/agents/dev-luna.toml` | `gpt-5.6-luna` / `xhigh`、低risk DEVだけ |
| `../agent-harness/.codex/agents/dev-sol.toml` | `gpt-5.6-sol` / `high`、high/unknown risk DEV |
| `../agent-harness/.codex/agents/explorer.toml` | `gpt-5.6-luna` / `medium`、`sandbox_mode = "read-only"`、子spawnなし |

launcherのcanonical contractは `agent-routing.mjs` が読み、CLIへは明示したmodelとreasoning effortだけを渡す。main/rootと各role Agentは一件のbounded explorer questionを起動できる。rootからのdirect explorerは深さ1、role Agentからのexplorerは深さ2である。explorerは設定、固定prompt、launcherの三層でchild spawningを禁止し、childを一切起動できない。通常の委譲はoffであり、`agents.max_threads = 2`、会話全文は保存しない。

二重rootの正本は解決済みである。product `.codex` がcanonicalであり、work repositoryの `.codex/config.toml` は既存initialization/governance workflowがcanonical contractから生成するmain-owned project-scoped adapterとする。このadapterはmainが共通lock下で別コミットとして更新する。DEVはproduct Task worktreeだけを編集し、work-repository configurationを編集・commitしない。adapterはincludeやmodel aliasを使わず、生成された明示TOMLでrole contractを展開する。deterministic drift validationはcanonical filesとadapterを正規化比較し、全role contractが同一でなければFAILする。

`run-work-agent.mjs` は子Agentを`.git` write権限なしのsandboxで実行しstdinをclosedにする。成功後だけ親がlockを保持してscopeを検査し、許可変更のみstage、既存pre-commit hook、`validate-work`、commit、commit後`validate-work`を順に行う。scope、hook、validation、child exitのいずれかがFAILならcommitしない。evidenceにはfailureと`commit:null`を残す。

fixed roleの`--profile`、`--model`、effort上書きはcanonical値と不一致なら起動前FAILし、一致する冗長な明示値だけ許す。roleを持たない既存generic launcherとWiki launcherは明示されたlegacy経路に限定して既存overrideを維持する。固定roleを経由させず、これをテストで証明する。

### 代替案と不採用理由

- role/model値を各launcher・両rootへ手で複写する案はdriftを招くため不採用。生成adapterと決定的drift validationを採用する。
- 未文書のTOML includeまたはmodel aliasに依存する案は不採用。明示生成TOMLを使う。
- 子Agentに`.git` writeを許す案はcommit authorityを委譲するため不採用。
- root-direct explorerを禁止する案は受け入れ条件と衝突するため不採用。
- explorerの再委譲またはfan-outを許す案は深さ、scope、token消費を統制できないため不採用。
- DEVを常にLunaにする案は横断的な権限・設定契約を過小評価するため不採用。

### 責務と不変条件

- main AgentはPLAN承認、DEV profile/promotion承認、work-repository adapterの生成・別コミット、branch/worktree、merge/revert、FAIL分類を所有する。
- Plannerは本PLANと機械検査可能なDEV選定証跡を所有する。承認欄はmainだけが更新する。
- DEVは承認profileでproduct Task worktreeだけを編集・検証し、profile昇格とwork adapterを自己承認・編集しない。
- launcher parentはlock、scope検査、stage/hook/commit、構造化evidenceを所有する。role childはallowed filesの編集・検証だけを行い、commit/stage/lock解放をしない。
- explorerはbounded read-only調査と短い根拠要約だけを行う。
- role、model、effort、深さ、explorer権限はcaller引数で緩和できない。parent commitは子exit=0、scope、hook、validationがすべて成功した場合だけ発生する。

### 失敗時・移行・互換性

fixed-role不一致override、未知DEV profile、PLAN証跡欠落、depth違反、explorer child spawn、scope外差分、adapter drift、child nonzero、hook/validation失敗はfail closedとする。部分stage、commit、lock release後の再試行はしない。rollbackはproductのrouting/configurationとmain-owned adapterをそれぞれの所有commitでrevertし、旧launcherを復元する。既存commit履歴、Task証跡、Wikiを移行しない。

## 変更予定

見積もり対象は実装、configuration、build integrationである。テストと文書は対象外である。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `../agent-harness/.codex/config.toml` | configuration | 40 | canonical project settings、depth/thread policy、role registry。 |
| `../agent-harness/.codex/agents/main.toml` | configuration | 22 | main contract。 |
| `../agent-harness/.codex/agents/planner.toml` | configuration | 22 | planner contract。 |
| `../agent-harness/.codex/agents/qa.toml` | configuration | 22 | QA contract。 |
| `../agent-harness/.codex/agents/reviewer.toml` | configuration | 22 | reviewer contract。 |
| `../agent-harness/.codex/agents/dev-luna.toml` | configuration | 25 | Luna DEV contract。 |
| `../agent-harness/.codex/agents/dev-sol.toml` | configuration | 25 | Sol DEV contract。 |
| `../agent-harness/.codex/agents/explorer.toml` | configuration | 28 | read-only/no-child explorer contract。 |
| `.codex/config.toml` | configuration | 30 | main-owned generated work-repository adapter。 |
| `../agent-harness/scripts/task/agent-routing.mjs` | implementation | 260 | canonical contracts、depth/override/PLAN/drift/evidence validation。 |
| `../agent-harness/scripts/task/run-work-agent.mjs` | implementation | 240 | closed stdin、child sandbox、parent scope/stage/hook/commit flow。 |
| `../agent-harness/scripts/task/run-wiki-agent.mjs` | implementation | 80 | legacy non-role/Wiki override validationとevidence統合。 |
| `../agent-harness/scripts/task/check-task.mjs` | implementation | 120 | DEV profile、reason、promotion履歴のmachine validation。 |
| `../agent-harness/scripts/task/work-pre-commit.mjs` | implementation | 110 | parent-owned stagingとallowed-path rejectionを維持するhook検査。 |
| `../agent-harness/Makefile` | build integration | 40 | routing/drift testsとrole launcher target。 |
| `../agent-harness/scripts/task/agent-routing.test.mjs` | test | - | routing、root/role explorer、depth、drift、evidence試験。 |
| `../agent-harness/scripts/task/development-process.test.mjs` | test | - | profile、promotion、parent commit、scope、closed stdin試験。 |
| `../agent-harness/docs/development/agent-roles.md` | documentation | - | profile、explorer、adapter契約。 |
| `../agent-harness/docs/development/development-process.md` | documentation | - | approval、promotion、parent commit、adapter governance。 |
| `../agent-harness/AGENTS.md` | documentation | - | direct child commit禁止とlauncher-owned commit規約。 |

実装/configuration/build integrationは15ファイル、1,086行。`file_score = 5`、`line_score = 6`。許容scale値 `1, 2, 3, 5, 8, 13` のうち6以上の最小値は8のため、`estimate_points: 8` とする。

## 実装手順

1. product canonical `.codex/config.toml` と7 role TOMLを追加し、固定profile、`agents.max_depth = 2`、thread limit、explorer no-child contractを定義する。
2. routingに明示contract、root-direct/role-child explorer規則、override・PLAN evidence検査、deterministic adapter drift validationを追加する。
3. 既存initialization/governance flowでmain-owned work adapterを生成し、mainが共通lock下で独立commitする。DEVはproduct worktreeのみを変更する。
4. role launcherをparent-owned commit flowへ変更し、closed stdin、scope検査、hook、evidenceを実装する。legacy non-role経路を分離する。
5. routing/drift unitとprocess integration試験、Makefile、運用文書を更新する。
6. product `make check`、work `make -C ../agent-harness work-check`、hook付き所有commit、独立review、merge後QAを既存gate順に実施する。

## 検証計画

- `node --test scripts/task/agent-routing.test.mjs`：全role/DEV profile、root→explorer、root→role→explorer、depth 3拒否、explorer child spawn拒否、read-only、一質問制約、override、adapter drift、evidenceを確認する。
- `node --test scripts/task/development-process.test.mjs`：`sol-high`/`luna-xhigh`、reason/risk欠落、未知profile、Luna rule違反、promotion履歴、scope外変更、closed stdin、child failure、parent-only commit、hook failureを確認する。
- fixtureでchildがcommitできず、親が許可ファイルだけをcommitし、commit hashをJSON evidenceに載せることを確認する。commitなし・拒否時は`commit:null`であることを確認する。
- drift fixtureでproduct canonical role filesとwork adapterの一値でも異なるcontractをFAILし、両rootから同一contractを解決することを確認する。
- productで`make check`、work repositoryで`make -C ../agent-harness work-check`と対象Task検査を実行する。Reviewerはdiff、evidenceの秘密値非含有、hook非迂回を独立確認し、QAはmerge済みmainで受け入れケースを再実行する。

## main Agentレビュー

- [x] 受け入れ条件が検証可能である。
- [x] 設計観点と代替案を検討している。
- [x] QA計画を作成できる。
- [x] 見積もりが規則どおりである。
- [x] DEV profile `sol-high`、選定理由、promotion規則を承認した。
- [x] DEV開始を承認した。
