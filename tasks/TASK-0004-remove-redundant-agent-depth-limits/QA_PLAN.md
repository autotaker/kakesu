---
task_id: "TASK-0004"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-14"
revision: 1
implementation_reviewed_at: "2026-07-14T21:38:56+10:00"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0004 QA PLAN

## 方針

live model の応答品質や実在Codex state DBには依存せず、product `main` の canonical TOML、`agent-routing` の決定的fixture、Explorer launcher の spawn stub、generated work adapter の完全一致検査を用いる。通常roleの「local depth key がない」構造不変条件と、Explorerの「local `max_depth = 0` によるno-child」不変条件を別々に観測し、いずれも起動・同期前にfail closedとなることを確認する。

QAはmain Agentが`--no-ff`で統合し、adapterを同期した後のproduct `main`とwork adapterを対象にする。Explorerはcustom-agent delegationではなく、`scripts/task/run-explorer-agent.mjs`のexplicit launcher contractとして検証する。実Codex呼出し、秘密値、会話全文、raw logは受入判定の根拠にしない。

## 前提と環境

- 製品repositoryは`../agent-harness`、運用repositoryは`../agent-harness-work`である。前者がcanonical `.codex`、routing parser、test、製品文書を所有し、後者がTask証跡とmain Agent同期済みadapterを所有する。
- Node.js 20以上、Git、共有pre-commit hook、product `make check`、`make -C ../agent-harness work-check`が利用できること。fixtureはtemporary canonical/work repositoryを使用し、global Codex設定、未文書include、model aliasを入力にしない。
- 実装後にREVIEW_RESULT、マージcommit、adapter同期commitを確認して、現行のtest名・command名へ対応付ける。この対応付けは期待結果・対象範囲を変えない限りrevisionを上げない。
- TASK-0004の承認済みDEV profileは`sol-high`である。QAはprofile、PLAN、adapterを変更または承認せず、証跡と実装結果を照合するだけとする。

## 受け入れ条件との対応

| ケースID | 受け入れ条件 | 操作 | 期待結果 | 証跡 |
|---|---|---|---|---|
| QA-001 | 通常6 roleにagent-localな深度上限がなく、project-scoped depth 2が正本である | `node --test scripts/task/agent-routing.test.mjs`のcanonical contract fixture、およびcanonical TOMLの直接照合を実行する。main/planner/qa/reviewer/dev-luna/dev-solへ`max_depth`を再導入するtemporary fixtureも実行する。 | 正常canonicalでは6 role全てに`[agents].max_depth` keyがなく、project configだけが`agents.max_depth = 2`および`max_threads = 2`を持つ。各通常roleへの値を問わないkey再導入はparserが同期・起動前に専用errorで拒否する。 | TAP case名/結果、role file名、rejection code。 |
| QA-002 | depth 2 / thread 2のトポロジーと既存max_threadsが回帰しない | `validateDelegation` fixtureでroot→Explorer（depth 1）、root→role→Explorer（depth 2）、depth 3、3 thread、Explorer fan-outと0/複数questionを検査する。 | depth 1/2と一問は許可される。depth 3、3 thread、Explorer fan-out、単一でないquestionは起動前に拒否され、通常roleのlocal depth欠落によって許可chainは変化しない。 | fixture assertion、拒否code、簡潔なlaunch evidence。 |
| QA-003 | Explorerだけがrole-local no-childと既存explicit launcher contractを維持する | Explorer TOML/prompt parser fixture、Explorer child-spawn負例、`run-explorer-agent.mjs`のspawn stub/dry-run fixtureを実行する。Explorerの`max_depth`欠落・非0化と`max_threads`変更のfixtureを含める。 | Explorerだけが`max_depth = 0`と`max_threads = 1`を持つ。欠落・緩和・child spawnはfail closed。launcherはfixed Luna/medium、read-only、closed stdin、write scopeなし、single bounded question、`commit:null`、短い根拠要約/file reference contractを維持する。 | TOML/prompt assertion、fixed CLI・stdio assertion、rejection code、redacted dry-run JSON。 |
| QA-004 | canonical parser、digest、generated adapterが二種類のdriftを区別して同期前に拒否する | temporary canonical fixtureで通常roleへの`max_depth`再導入、Explorer depth 0の削除・変更、project depth/thread変更を注入し、`readCanonicalContracts`、digest、adapter render、`syncWorkAdapter(..., { check: true })`または同等checkを実行する。 | 正常時はadapterがproject depth/thread 2とrole registryを決定的に解決する。通常role prohibited keyとExplorer required keyの各driftは別の理由でnonzeroとなり、render/sync前にfail closedする。global config等による回避はない。 | fixture別PASS/FAIL、normalized digest、drift field/rejection code、`changed=false` check結果。 |
| QA-005 | Explorer no-childの多層防御とparent-owned commit境界に回帰がない | routing/development-process fixtureでfixed prompt、launcher入力、closed stdin、write scope、child stage/commit、scope/hook/validation failureを検査する。 | prompt、read-only sandbox、explicit launcher、no-child設定、負例が一層でも欠ければ検査がFAILする。Explorer/childはGit write・scope拡大・再委譲できず、failure時は親のcommitがなく`commit:null`となる。 | TAP summary、ordered event assertion、HEAD before/after、launch JSON。 |
| QA-006 | 製品文書が通常roleのproject-scoped上限とExplorer固有no-childを区別する | `docs/development/agent-roles.md`をレビューし、削除済みの通常role local depthを現行contractとして説明する文言を検索する。 | 文書はproject `max_depth = 2` / `max_threads = 2`を全体トポロジー上限、Explorer `max_depth = 0`をrole固有no-childとして明示し、model、effort、sandbox、registryの不要な変更を含まない。 | reviewed path、検索語と結果、差分照合。 |
| QA-007 | routing・process・adapter・repository boundaryの回帰がない | merge済みmainで`node --test scripts/task/agent-routing.test.mjs`、`node --test scripts/task/development-process.test.mjs`、`make check`、adapter complete-match check、`make -C ../agent-harness work-check`を実行する。 | 全commandがexit 0。生成work adapterはmain Agent同期済みcanonicalと完全一致し、製品/運用repositoryの所有境界、lock、parent-owned commit、TASK-0003の未承認native経路非依存に回帰がない。 | command、exit status、短いtest summary、digest、work-check output。 |

## 境界・異常・回帰

- expected/actualの不一致は、`implementation defect`、`QA plan defect`、`requirement or plan defect`、`environment issue`、`regression`に分類する。複合原因は無理に一つへ圧縮しない。DEVへの自動帰責、差し戻し先、revert、バグ化は行わずmain Agentの判断へ委ねる。
- 通常roleのlocal-depth再導入、Explorer depth 0の欠落・緩和、project depth/thread変更、depth 3、3 thread、Explorer child/fan-out、複数question、global/alias依存、adapter driftは全てfail closedであることを確認する。
- Explorerの実効境界はdepth値のみでPASSとしない。fixed prompt、read-only、explicit launcher、closed stdin、empty write scope、`commit:null`、child-spawn負例の全層を確認する。
- product `make check`とadapter wrapperはlockまたはbuild出力の書込みを必要とする場合がある。QA sandboxで実行不能でも、直接のread-only fixtureを可能な範囲で行い、main Agentがlock解放後・権限を持つ環境で同一必須gateを再実行した結果と区別する。

## 実施不能時の扱い

- product main、同期済みadapter、shared hook、fixture toolchain、lock、またはbuild output権限が利用不能なら、影響したケース、再現command、阻害した依存、環境分類をQA_RESULTに記録する。未実行gateをPASSに読み替えない。
- direct fixtureまたはread-only adapter checkで代替観測できる部分は実行する。ただし必須wrapperの最終PASSは、適切なlock/書込み権限を持つmain Agentの再実行結果を別証跡として確認する。
- 実装結果が本計画と異なるときはcanonical contract、approved PLAN、fixtureを順に照合する。期待結果または試験範囲を変更する必要がある場合のみ、QA_PLANを改訂しmain Agentの承認を得るまで既存期待を置き換えない。

## 実装後の再確認

- [x] 実装差分とレビュー結果を確認した。review対象は`32a1c38ace6b1395b82c5b001777355b459a9558`、判定は`pass`である。
- [x] merge済みproduct `main`の`514facbb461927c0e5fc376a56ab8f975c054940`とmain-owned generated work adapterを確認した。adapterはdigest `5566794aaf22a890ef432e4e45b63a3a07e1bcb3eaf0cf620a47883c705c3445`で完全一致し、再生成はno-opである。
- [x] QA-001〜QA-007の操作手順を現行実装に照合し、変更不要であることを確認した。
- [x] 期待結果または試験範囲の変更がないことを確認した。
- [x] 期待結果または範囲の変更がないため、追加のmain Agent承認は不要であることを確認した。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-14 | qa-agent-terra-medium | approved PLAN、TASK-0002のapproved QA evidence、関連Wikiに基づく実装前QA計画 | `approved` |
