---
task_id: "TASK-0046"
title: "Wiki launcherの既存dirty差分を保全する"
status: plan
created_at: "2026-08-01"
---

# TASK-0046 Wiki launcherの既存dirty差分を保全する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TASK-0045のWiki ingestで実際に発生した、`--commit false`実行失敗時に開始前から存在したTask証跡を`reset --hard`/`clean`で消す障害を修正する。Wiki Agentが加えた差分だけをscope判定とrollbackの対象にし、開始前のtracked/staged/untracked状態を成功・失敗の両方で保持する。あわせて子Agent非zero終了を、秘密情報を漏らさない短い診断へ分類する。

### 対象と対象外

#### 対象

- edit-only Wiki launcher開始時のHEAD、index、working tree、untracked file状態のsnapshotと、子実行後との差分集合の導出。
- 子Agentが追加・変更したpathだけに対するWiki allowlist検査。開始前dirty pathを子のscope違反に数えない。
- 子失敗、commit/stage試行、scope違反、validation失敗時に、開始前のbytes、削除状態、mode、index、untracked fileを復元するrollback。
- Codex子プロセスstderrをbounded・redactedな診断へ正規化し、raw stderrを例外へ流さない処理。
- temporary Git fixtureだけを使うhermetic回帰テスト。

#### 対象外

- Wiki本文、Schema、Task lifecycle、role/model、allowlist、commit数、証跡形式、他launcherのclean-only契約の変更。
- Codex CLI、network、実Wiki Agent、実認証情報を使うE2E。
- 既存dirty repositoryでのcommit-mode Wiki ingest許可。commit-modeは引き続きclean開始だけとする。
- 汎用backup/archive機構、全repository複製、stash ref作成、追加dependency。

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: `run-wiki-agent.mjs --commit false`は開始前から存在するtrackedのstaged/unstaged変更、削除、untracked fileを子変更として扱わず、Wiki Agent成功時にそれらのbytes・mode・index状態を保持したまま、子が変更した許可Wiki pathだけを残す。
- [ ] AC-2: edit-only実行で子非zero、HEAD変更、stage、許可外path変更、又はvalidation失敗が起きた場合、子が作った変更だけを破棄し、開始前のHEAD・tracked bytes/削除・mode・index・untracked fileを同一状態へ復元する。rollback失敗は元の失敗と区別してfail-closedに報告する。
- [ ] AC-3: 子非zeroの公開診断は固定prefixとbounded・redactedなstderr要約だけを含み、Bearer、`sk-`形式、長いtoken候補を含めない。raw stderrをthrow又はJSONへ流さず、exit codeは保持する。
- [ ] AC-4: commit-mode Wiki launcherとclean開始の既存rollback契約は変えない。temporary Git fixtureによるfocused testが、同一pathへの子上書き、staged+unstaged同居、tracked削除、untracked保持、子untracked除去、scope違反、子stage/commit、stderr redactionを外部network/Codexなしで検出し、`make check`がPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0045 Wiki ingest障害 | main `0e8136a`のHANDOVERと本Task起票時の実行観測 | 既存dirty 4証跡がrollbackで消失し、復元と一時cloneを要した再現事実 |
| REF-2 | Wiki launcher | main `0e8136a`の`scripts/task/run-wiki-agent.mjs` | edit-onlyでも全dirty pathを子差分として扱い、catchで破壊的rollbackする原因箇所 |
| REF-3 | 共通rollback | main `0e8136a`の`scripts/task/agent-routing.mjs` | `reset --hard`と`clean -ffd`を前提とするclean-start契約 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| なし | `ready` | N/A | N/A |

### 許可パス

- `scripts/task/run-wiki-agent.mjs`
- `scripts/task/agent-routing.mjs`
- `scripts/task/development-process.test.mjs`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | testは`mkdtemp`配下のGit fixtureだけを書き、実Codex・network・credentialsを使わない |
| 依存状態と参照 | `ready` | TASK-0045で障害と復元結果を固定済み。追加依存なし |
| 生成物の有無と更新方法 | `ready` | JavaScript source/testのみ。生成物なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0046-wiki-launcher-preserve-dirty` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、receipt、追加形式checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

`run-wiki-agent.mjs`はedit-only時だけdirty開始を許す一方、開始前集合を記録せず、子実行後の全dirty pathをscope検査する。そのためTASK-0045の4証跡をWiki scope違反と誤判定した。catchはclean-start用の`rollbackWorkRepository`を呼び、`reset --hard`と`clean -ffd`で開始前証跡まで消した。さらに子stderrを捨てるため、一時cloneのdependency不足と状態DB read-onlyを診断するのに複数回の再実行を要した。この実害と時間損失を直接除く。

## 検討すべき設計観点

- path集合の差だけでは、子が開始前dirtyと同じpathを上書きした事実を検出できない。bytes、mode、存在、index状態まで比較する。
- edit-only用snapshot/restoreをclean-only rollbackへ混ぜず、既存launcherの単純なfail-closed契約を維持する。
- untracked directory自体ではなく配下fileを明示snapshotし、開始後に増えたfileだけを除去する。symlinkや特殊fileは安全に扱えない場合fail-closedにする。
- stderrは診断可能性と秘密非露出を両立し、固定長・固定redactionを通した文字列以外を公開しない。
- test用の過剰なCLI injectionや本番環境変数を追加せず、状態比較・復元を純粋なhelperとして直接検証する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] planning/candidate/completionの3 commit経路とcandidate一回の`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: 独立計画レビュー、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/scripts/task-delivery.md`

### 判断

- 実害が出たedit-only境界だけを修正し、一般化はhelper再利用に必要な範囲へ限定する。

### 適用しなかった重要な判断

- dirty全体をstash/commitしてからAgentを起動する方式は、ref/index副作用と追加復元経路を増やすため採用しない。
- dirty開始を全面禁止する方式は、MainがTask証跡を保持したままWiki本文だけをAgent所有で更新する用途を失うため採用しない。
