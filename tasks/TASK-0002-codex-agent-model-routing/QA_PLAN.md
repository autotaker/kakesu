---
task_id: "TASK-0002"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-14"
revision: 2
implementation_reviewed_at: "2026-07-14T12:10:41+10:00"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0002 QA PLAN

## 方針

live model の応答品質には依存せず、canonical role contract、launcher の dry-run、Node の unit/integration fixture、Git fixture を用いて、起動前拒否・権限境界・親所有 commit を決定的に検証する。実行が必要な子は固定 fixture command とし、実際の Codex 呼出し、秘密値、会話全文、raw log を証跡に残さない。QA は merge 済み product `main` と、main Agent が別 commit した生成済み work adapter を対象とする。

## 前提と環境

- 製品リポジトリは `../agent-harness`、運用リポジトリは `../agent-harness-work`。製品実装と canonical `.codex` は前者、Task 証跡と生成 work adapter は後者に属する。
- Node.js 20 以上、Git、共有 pre-commit hook、product `make check`、`make -C ../agent-harness work-check` を利用できる。試験は一時 fixture repository/worktree を作成し、実在する利用者 global Codex 設定を参照しない。
- 起動試験は実装予定の `node --test scripts/task/agent-routing.test.mjs`、`node --test scripts/task/development-process.test.mjs` と同等の明示 fixture を使う。各 fixture は stdin、sandbox/write scope、hook、child exit、commit hash を注入可能にする。
- TASK-0002 の承認済み DEV profile は PLAN frontmatter と選定証跡の `sol-high` である。QA は profile を変更・承認せず、承認済み証跡との照合だけを行う。

## 実装レビュー済み command 対応（`1b4013bb9563adb53939b1b9fd632b440f1ee918`）

受け入れ期待は変更しない。以下は実装後の fixture / command 名への対応だけを明確化する。Explorer は custom-agent delegation ではなく、明示 launcher `scripts/task/run-explorer-agent.mjs`（または `make explorer-agent`）で起動する。したがって、深さ・thread の fixture は既存 role contract の検証、Explorer の実行契約は launcher の fixed CLI / closed-stdin 試験として扱う。

| ケースID | 実装済みの主な検証 command / fixture | contract 対応 |
|---|---|---|
| QA-001 | `node --test scripts/task/agent-routing.test.mjs` の `role routing matches the canonical model and effort contracts` | `.codex/agents/{main,planner,qa,reviewer,dev-luna,dev-sol,explorer}.toml` の canonical contract を全件読む。 |
| QA-002 | `make task-check TASK=TASK-0002 WORK_ROOT=../agent-harness-work` と両 test file の DEV selection fixture | TASK の承認済み `sol-high` 証跡と、unknown / missing / risky Luna の起動前拒否を照合する。 |
| QA-003 | `agent-routing.test.mjs` の `DEV selection permits Luna only without risk and validates promotion`、`development-process.test.mjs` の `DEV profile evidence rejects unknown, missing, and risky Luna selections` | Luna eligibility、Sol 選択、promotion 必須 field、降格拒否を検証する。 |
| QA-004 | `agent-routing.test.mjs` の `explorer delegation accepts only depth two, two threads, and one question` と `node scripts/task/run-explorer-agent.mjs --root <root> --question '<question>' --dry-run true` | 前者は role depth/thread contract、後者は明示 Explorer launcher の一問・read-only evidence contract を検証する。 |
| QA-005 | `agent-routing.test.mjs` の `Explorer launcher requires one question and invokes the fixed CLI contract with closed stdin` | `codex exec -C <root> --sandbox read-only -m gpt-5.6-luna -c model_reasoning_effort="medium"`、closed stdin、write scopeなし、commit null を検証する。 |
| QA-006 | `node scripts/task/agent-routing.mjs --work-root ../agent-harness-work --mode check` と `agent-routing.test.mjs` の `work adapter generation is deterministic and detects drift` | generated adapter digest / drift fail-closed を検証する。 |
| QA-007 | `agent-routing.test.mjs` の `fixed role overrides fail closed and redundant canonical values pass` と legacy Wiki launcher fixture | fixed role の override 拒否と、role を経由しない Wiki legacy route を分離して検証する。 |
| QA-008 | `agent-routing.test.mjs` の Explorer launcher fixture と `development-process.test.mjs` の `launchers close child stdin and reserve commits for the lock-owning parent` | Explorer と work role launcher の `stdio:["ignore","pipe","pipe"]` を検証する。 |
| QA-009 | `agent-routing.test.mjs` の `parent commit preconditions reject child commit, stage, failure, and scope drift` | child の edit と parent の scope validation / commit ownership を検証する。 |
| QA-010 | `development-process.test.mjs` の `failure rollback restores HEAD, index, worktree, untracked files, and lock` | child nonzero、scope、stage/commit、hook、validation failure の rollback / no-commit を検証する。 |
| QA-011 | `agent-routing.test.mjs` の `launch evidence is concise, closed-stdin, and redacts secret-like values` | one-line evidence、redaction、raw log 非保存を検証する。 |
| QA-012 | `development-process.test.mjs` の `DEV gate rejects missing role separation and worktree assignment` と `make task-check TASK=TASK-0002 WORK_ROOT=../agent-harness-work` | repository boundary、role separation、承認済み証跡 gate を検証する。main-only `--no-ff` merge は main の merge gate で別途実施する。 |
| QA-013 | `make check`、`node --test scripts/task/agent-routing.test.mjs`、`node --test scripts/task/development-process.test.mjs`、`make -C ../agent-harness work-check` | merge 済み main と main-owned generated adapter での必須 regression gate を検証する。 |

## 受け入れ条件との対応

| ケースID | 受け入れ条件 | 操作 | 期待結果 | 証跡 |
|---|---|---|---|---|
| QA-001 | main、planner、qa、reviewer、explorer、dev-luna、dev-sol の固定 model / effort | `node --test scripts/task/agent-routing.test.mjs --test-name-pattern='role routing'` を実行し、canonical product role TOML と両 root の解決結果を fixture assertion で比較する。 | main=`gpt-5.6-sol/high`、planner/qa/reviewer=`gpt-5.6-terra/medium`、explorer=`gpt-5.6-luna/medium`、dev-luna=`gpt-5.6-luna/xhigh`、dev-sol=`gpt-5.6-sol/high` の完全一致以外は FAIL。 | TAP の case 名・PASS、canonical role file 名、redacted routing dry-run JSON。 |
| QA-002 | TASK-0002 の承認 profile と invalid/unapproved DEV 拒否 | `check-task` の TASK-0002 fixture を実行し `sol-high`、reason、risk signals を確認する。profile を `luna-xhigh`、未知値、reason/risk 欠落へ各々差し替えた fixture を起動する。 | 現行 PLAN の `sol-high` は PASS。未承認、未知、証跡欠落は子起動前に nonzero FAIL し、child result と commit はともに null。 | check-task assertion、拒否理由 code、launch JSON の `child_result:null` と `commit:null`。 |
| QA-003 | Luna eligibility と Luna→Sol promotion 履歴 | 明確・局所的・機械検証可能で高 risk signal なしの PLAN fixture と、曖昧・横断的・contract/Schema/security/concurrency/migration signal を一つずつ持つ fixture を検査する。続けて Luna から Sol への変更前 profile、signal、理由、main 承認者、時刻を含む promotion fixture を検査する。 | 全低 risk 条件を満たす場合だけ `luna-xhigh` を PASS。signal 又は不明が一つでもあれば `sol-high` 以外を FAIL。promotion は全履歴フィールドがあれば PASS、欠落・Sol から Luna への降格は FAIL。 | 判定 fixture 名と結果、machine-readable selection/promotion record、拒否 code。 |
| QA-004 | main→explorer、main→role→explorer、深さ/thread と bounded 調査 | `validateDelegation` fixture で root→explorer（depth 1）と root→role→explorer（depth 2）、depth 3、explorer→child、同時 3 thread、質問 0/2 件を検査する。Explorer 実行は `node scripts/task/run-explorer-agent.mjs --root <root> --question '<question>' --dry-run true` で確認する。 | 二つの許可 chain は role contract として PASS。`max_depth=2`、`max_threads=2` を超える入力、explorer の child/fan-out、単一でない bounded question は起動前 FAIL。Explorer は explicit launcher 経路であり、通常委譲は既定 off。 | canonical config / fixture assertion、launcher dry-run JSON、拒否 code、`commit:null`。 |
| QA-005 | explorer の read-only、簡潔な結果、再委譲禁止 | `agent-routing.test.mjs` の Explorer launcher fixture と、fixed prompt / CLI assertion を実行する。実 Codex を呼ばず、spawn stub が一問への短い evidence summary を返す。 | explorer は explicit launcher から `sandbox_mode=read-only`、write scope なし、closed stdin、commit null で起動する。prompt は編集、Git write、scope 拡大、再委譲を禁止し、短い要約と file reference だけを要求する。 | fixed CLI / sandbox / stdin assertion、prompt policy assertion、redacted child result JSON。 |
| QA-006 | product canonical config と generated work adapter の同一性、global/alias 非依存 | product root と work root から設定解決 fixture を実行し normalized role contract を比較する。adapter の model、effort、sandbox、depth、spawn constraint、role 名を一値ずつ変えた drift fixture、global config と undocumented alias を注入した fixture を実行する。 | 両 root は同一の明示 contract を解決する。全 drift、global config 依存、未文書 alias は FAIL。adapter は生成済み明示 TOML であり undocumented include を必要としない。 | normalized contract digest、drift field/rejection code、resolved cwd を含む dry-run JSON。 |
| QA-007 | fixed-role override 拒否と legacy Wiki/non-role 互換 | main/planner/qa/reviewer/dev/explorer launcher に canonical と不一致の `--profile`、`--model`、effort を渡し、一致する冗長値も渡す。明示 legacy generic launcher と Wiki launcher の既存 override fixture を role を付けずに実行する。 | fixed role の不一致は起動前 FAIL、一致値のみ PASS。文書化済み legacy non-role/Wiki 経路の override は維持するが、fixed role routing を経由・緩和しない。 | argument-validation result、legacy route identifier、PASS/FAIL launch JSON（秘密値なし）。 |
| QA-008 | non-TTY / closed stdin | child fixture が TTY を要求せず EOF を観測するよう、pipe/closed stdin で parent launcher integration test を実行する。stdin 読取り待ち fixture も実行する。 | child は non-TTY かつ stdin closed で実行され、入力待ちで hang しない。stdin が必要な child は timeout/nonzero となり parent は commit しない。 | fixture exit/timeout assertion、evidence の `stdin:"closed"`、`commit:null` 又は成功 commit hash。 |
| QA-009 | parent-owned commit 正常系 | allowlist 内の一ファイルだけを編集して exit 0 の child fixture を parent launcher で実行する。fixture child に Git stage/commit 権限がないことも確認する。 | lock 保持中に parent だけが scope validation、stage、hook、validation、commit、commit 後 validation を順に実行する。child は edit のみで Git write 不可、親 commit hash が一件だけ発生する。 | ordered event fixture、lock identifier、Git log/commit hash、launch JSON の child result と commit。 |
| QA-010 | child/scope/hook/validation failure 時の no-commit | child nonzero、scope 外 edit、hook nonzero、pre/post validation nonzero の各 fixture を parent launcher で実行する。 | 全 failure は nonzero、stage/commit を行わず、既存 HEAD は不変。lock は parent cleanup 後に解放され、evidence の `commit:null` が明示される。 | fixture 別 failure code、HEAD before/after assertion、event trace、launch JSON。 |
| QA-011 | concise launch JSON evidence | QA-001〜QA-010 の成功・拒否 fixture から evidence を schema assertion で検査する。secret-like key、token 値、大量 raw log を含む fixture も投入する。 | 一行 JSON は role/profile、model、effort、cwd、sandbox/write scope、allowed paths、child result、commit hash 又は null を含む。secret/raw log は除外/ redaction される。 | JSON schema assertion、redaction assertion、成功と失敗の sample evidence。 |
| QA-012 | repository boundary、role separation、PLAN/DEV/QA gate、main-only merge | product/work root、role ownership、PLAN approval、DEV profile、QA plan、review result を欠落/不正化する process fixture と、role child が merge を試す fixture を実行する。main による `--no-ff` merge fixture も実行する。 | product code と work evidence の所有境界を越える変更、未 gate の進行、child/role の merge は拒否。main のみが独立 review PASS 後に `--no-ff` merge できる。 | boundary/gate rejection code、Git graph assertion（no-ff merge）、role/allowed-path evidence。 |
| QA-013 | regression | product `make check`、`node --test scripts/task/agent-routing.test.mjs`、`node --test scripts/task/development-process.test.mjs`、`make -C ../agent-harness work-check` を merge 済み main と生成 adapter で実行する。 | 全 command が exit 0。既存 process、hook、Task/Wiki/work validation を回帰させず、生成物/worktree を work repository に混入させない。 | command name、exit status、短い test summary、work validation output。 |

## 境界・異常・回帰

- 失敗は責任追及ではなく修正対象の分類に使う。実装と canonical contract の不一致は `implementation defect`、QA case の誤った期待は `QA plan defect`、TASK/PLAN の曖昧・欠落は `requirement or plan defect`、toolchain/fixture/lock の再現不能は `environment defect`、既存 PASS の破損は `regression` とする。複合原因は複数分類を記録する。
- QA Agent は FAIL を DEV に自動帰責せず、fixture、command、expected/actual、classification を QA_RESULT に記録する。差し戻し先、revert、バグ化、期待値・範囲変更は main Agent が決定する。
- adapter drift、global/alias、固定 role override、未承認 profile、depth/thread、explorer write/spawn、closed-stdin child、scope/hook/validation/child failure は全て fail-closed とし、部分 commit、child commit、raw log 保存を許容しない。
- regression は product `make check` と work-check を必須とする。live Codex service の可用性やモデル応答差は acceptance 判定の前提にせず、必要なら別途 environment observation として記録する。

## 実施不能時の扱い

- product main、生成 adapter、shared hook、又は fixture toolchain が利用不能なら、影響したケースを未実施にし、再現 command、欠けた依存、environment classification を記録する。dry-run/evidence fixture で代替できるケースは実行するが、実行できない gate を PASS 扱いしない。
- 実行結果が本計画の期待と異なる場合、まず canonical contract、PLAN、fixture を照合する。期待結果又は対象範囲を変える必要があれば QA_PLAN を単独で改訂し、main Agent の承認を得るまで既存期待を置き換えない。

## 実装後の再確認

- [ ] merge 済み product `main`、main-owned generated work adapter、実装差分、REVIEW_RESULT を確認する。
- [ ] QA-001〜QA-013 の command/fixture 名を現行実装へ対応付け、live model 依存が混入していないことを確認する。
- [ ] TASK、approved PLAN、canonical contract と期待 model/effort、profile、権限、commit contract を再照合する。
- [ ] 期待結果または試験範囲の変更有無を確認する。変更時は main Agent の承認を得て revision と理由を記録する。
- [ ] merge 後 main で QA-001〜QA-013 と product `make check` / work-check を再実行し、QA_RESULT に evidence と classification を記録する。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-14 | qa-agent-terra-medium | approved PLAN と local Wiki に基づく実装前 QA 計画 | `approved` |
| 2 | 2026-07-14 | qa-agent-terra-medium | `1b4013bb9563adb53939b1b9fd632b440f1ee918` と REVIEW_RESULT を照合し、QA-001〜QA-013 の実装 command 対応、特に explicit Explorer launcher を明確化。期待結果・範囲は不変。 | 不要（`expectation_changed=false`） |
