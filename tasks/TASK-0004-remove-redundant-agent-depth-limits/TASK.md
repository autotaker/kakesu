---
task_id: "TASK-0004"
title: "通常Agentの重複した深度上限を削除しExplorerの再委譲禁止を維持する"
status: complete
created_at: "2026-07-14"
---

# TASK-0004 通常Agentの重複した深度上限を削除しExplorerの再委譲禁止を維持する

## 目的

通常の非Explorer Agent定義に重複しているagent-localな深度上限を削除し、委譲トポロジーの上限をproject-scoped設定へ一元化する。同時に、Explorer自身だけに必要な再委譲禁止を、Explorer固有設定、固定prompt、明示launcher、負例試験で引き続きfail closedにし、設定の簡素化が権限緩和にならない状態を作る。

## 背景

TASK-0002では、rootからrole Agentを経由してExplorerへ到達できる全体上限として、project-scopedな`agents.max_depth = 2`と`agents.max_threads = 2`を導入した。さらに、main、Planner、QA、Reviewer、DEVの各role定義にも同じ形の`[agents]`設定を置き、通常roleへ`max_depth = 1`、Explorerへ`max_depth = 0`を設定した。

その後の承認済みQA PLAN revision 2とQA結果では、Explorerの実行経路はcustom-agent delegationではなく`run-explorer-agent.mjs`による明示launcherであり、一質問、Luna / `medium`、read-only、closed stdin、write scopeなし、`commit:null`をlauncher contractとして検証する形に確定した。通常roleに複写されたagent-local `max_depth = 1`は、このExplorer実行境界を保証せず、project全体の深度上限と責務が重複している。六つの通常roleへ同じ値を保持すると、Codexの深度意味論または将来の明示的なnested実行を検証するときに、どの上限が正本かが曖昧になり、設定driftや意図しない起動拒否の原因になる。

一方、Explorerの`max_depth = 0`は、Explorerから別Agentを起動させないrole固有の防御であり、通常roleの重複上限とは性質が異なる。本Taskは、project全体の最大深さ2と最大thread数2を維持したまま通常roleの重複だけを除き、Explorerの再委譲禁止が設定簡素化に巻き込まれないことを自動検査で証明する。TASK-0003のPLAN内native Explorer起動は別Taskの未承認計画であり、本Taskではその実装や受け入れを前提にしない。

## スコープ

### 対象

- 製品リポジトリのcanonical role定義にあるmain、Planner、QA、Reviewer、DEV Luna、DEV Solのagent-localな深度上限の削除
- project-scopedな最大深さ2を通常Agentの委譲トポロジー上限として一元的に扱う設定、解決、検査
- Explorer固有の`max_depth = 0`、read-only、一質問、scope拡大禁止、再委譲禁止を維持する多層contract
- canonical設定の構造、生成work adapter、digest / drift検査を、通常roleの深度上限不在とExplorerのno-childを区別して検証する処理
- 正常系、設定drift、深さ超過、Explorerからのchild spawnを検出する自動テスト
- 通常roleの深度上限とExplorerのno-childの正本を正確に説明する必要最小限の製品文書
- 製品`main`への統合後にmain Agentが行う生成work adapterの同期と、両repository rootでの回帰確認

### 対象外

- project-scopedな`agents.max_depth = 2`または`agents.max_threads = 2`の緩和、削除、増加
- 通常roleまたはExplorerの`max_threads`値の変更
- Explorerの`max_depth = 0`、read-only sandbox、一質問制約、固定model / effort、編集・Git書込み・scope拡大・再委譲禁止の緩和
- Explorer以外の新しい子role、fan-out、generic delegation、natural-languageによるrole選択の追加
- TASK-0003が扱うPLANからのMultiAgentV2 native Explorer起動の実装、承認、移行
- main、PLAN、DEV、QA、REVIEWのmodel routing、責務分離、PLAN / QA gate、親所有commit、Git運用の変更
- 利用者global設定、未文書include、model aliasへの依存
- 運用リポジトリの通常Wiki本文、Decision、Schemaの変更

## 受け入れ条件

- [x] canonicalなmain、Planner、QA、Reviewer、DEV Luna、DEV Solの各role定義は、agent-localな`max_depth`または同等の重複した深度上限を持たず、project-scopedな深度contractに従う。
- [x] 製品repositoryと生成work adapterは、全体の委譲トポロジー上限として`agents.max_depth = 2`を引き続き解決し、rootから深さ3以上へ到達するchainを許可しない。
- [x] Explorer roleは固有の`max_depth = 0`を保持し、Explorerからのchild spawnまたは再委譲を設定検査と負例試験で起動前に拒否する。
- [x] Explorerのno-childは深度値だけに依存せず、固定prompt、read-only sandbox、明示launcher、closed stdin、write scopeなし、`commit:null`の既存contractでも維持される。
- [x] projectおよび各roleの`max_threads`値と、深さ3、3 thread以上、Explorer fan-out、一件でないbounded questionを拒否する既存contractに回帰がない。
- [x] canonical parser、digest、生成adapterまたは同等のdrift検査は、通常roleへのagent-local深度上限の再導入と、Explorerの`max_depth = 0`の欠落・緩和を区別して検出し、起動または同期前にfail closedにする。
- [x] rootからの既存`run-explorer-agent.mjs`経路と各work roleへ提示される明示Explorer launcher経路は、固定model / effort、一質問、read-only、簡潔な根拠要約とfile referenceという既存の実効contractを維持する。
- [x] 製品文書は、通常roleがproject-scopedな深度上限に従うことと、Explorerだけがrole固有のno-child上限を持つことを区別し、削除済みの重複設定を現行contractとして説明しない。
- [x] 自動テストは、通常roleの深度上限不在、project全体のdepth 2、Explorerのdepth 0 / no-child、設定driftの正負例を決定的に証明し、live modelの応答品質に依存しない。
- [x] 製品`make check`、routing関連試験、生成adapterの完全一致検査、`make -C ../agent-harness work-check`がPASSし、製品・運用repositoryの所有境界と親所有commitに回帰がない。

## 検討すべき設計観点

- project-scopedな`agents.max_depth = 2`と、各role定義内の`[agents].max_depth`について、Codex runtimeがどの起点・相対深度・優先順位で評価するかをfixtureで固定し、通常roleから削除しても許可chainと拒否chainが変わらないこと
- 通常roleの「全体上限に従う」という継承contractと、Explorerの「自身は子を持てない」というrole固有contractを、同じ深度設定として一括処理せず別の不変条件として表現すること
- canonical parserとdigestが値だけでなく、通常roleでは深度keyが存在しないこと、Explorerでは`max_depth = 0`が存在することを検査し、手動再導入や欠落をdriftとして検出する方法
- product canonical設定を先に変更し、main Agentが所有するwork adapterを決定的に再生成する順序と、二つのrepository root間で一時的な不一致を受け入れない同期・rollback方法
- `max_depth`だけを対象とし、`max_threads`、model、effort、sandbox、role registry、Explorer launcherの既存値を不要に書き換えない最小差分
- Explorerの再委譲禁止を、role設定、固定prompt、launcher入力検査、tool公開範囲、負例試験のどこで検出し、いずれか一層の欠落をテストが確実にFAILさせるか
- TASK-0003の未承認PLANや将来のnative nested経路に依存せず本Taskを完結させつつ、後続Taskがproject深度とExplorer no-childの正本を再利用できる境界
- `make check`とadapter drift試験がglobal設定、live Codex、実在するstate DBへの書込みに依存せず、管理sandboxでも決定的に実施できる検証設計

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] 必要な実装、テスト、文書が完成している。
- [x] 独立レビューと`make check`がPASSしている。
- [x] マージ後QAが完了している。
- [x] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- [Codex Agent Model Routing](../../wiki/semantic/schemas/codex-agent-model-routing.md): 製品repositoryのproject-scoped TOMLを正本とし、Explorerはrootまたはroot→role→Explorerで起動できる、depth 2 / threads 2の範囲内の一質問・read-only・no-child調査として扱う。固定prompt、closed stdin、負例試験、launcher traceを含む多層境界を検証する。
- [Task Delivery](../../wiki/semantic/scripts/task-delivery.md): PLAN承認、実装前QA計画、独立review、mainへの`--no-ff`マージ、マージ後QA、HANDOVER ingestの順序を維持する。
- [Work Repository Boundary](../../wiki/semantic/schemas/work-repository-boundary.md): 製品repositoryがコード・テスト・製品文書を、運用repositoryがTask証跡とWikiを所有する。運用repositoryへの書込みはlockで直列化し、Agentは所有範囲だけを直接commitする。
- [QA FAIL Attribution](../../wiki/semantic/case-patterns/qa-fail-attribution.md): runtime深度意味論、設定drift、QA fixture、管理sandboxの失敗を区別し、FAILをDEVへ自動帰責しない。

### 判断

- [DECISION-0001 PLAN DEV QA Process](../../wiki/decisions/DECISION-0001-plan-dev-qa-process.md): 設定整理でも既存の役割分離、承認、独立review、マージ後QAを省略しない。
- [DECISION-0002 Work Repository and Wiki Ownership](../../wiki/decisions/DECISION-0002-work-repository-and-wiki-ownership.md): role設定と製品文書は製品repository、Task証跡は運用repository、work adapter同期はmain Agentの責務として維持する。
- [DECISION-0003 Codex Agent Model Routing](../../wiki/decisions/DECISION-0003-codex-agent-model-routing.md): project-scoped TOMLをrole routingの正本とし、Explorerを明示launcherによる一問のread-only調査、depth 2、threads 2、no-childとする判断を維持する。通常roleの重複上限を削除する設計は、この正本とExplorer固有境界を損なわないことを示す。
- [TASK-0002 approved PLAN](../TASK-0002-codex-agent-model-routing/PLAN.md) と [approved QA PLAN](../TASK-0002-codex-agent-model-routing/QA_PLAN.md): projectのdepth 2とExplorerのno-childを受け入れ期待とし、revision 2でExplorer実行をcustom-agent delegationではなく明示launcher contractとして検証する整理を根拠にする。
- [TASK-0002 QA RESULT](../TASK-0002-codex-agent-model-routing/QA_RESULT.md): QA-004 / QA-005でdepth / thread fixtureとExplorer launcherを別々にPASS判定した証跡を、変更前baselineとして使う。

### 適用しなかった重要な判断

- なし。DECISION-0003のdepth 2、threads 2、Explorer no-childは維持し、通常role定義から重複した深度値を除くことによって同Decisionを置換または緩和しない。TASK-0003のdraft PLANは承認済み判断ではないため、本Taskの要件根拠または実装前提には用いない。
