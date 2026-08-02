---
task_id: "TASK-0064"
title: "文書lintの失敗を一回で集約する"
status: draft
created_at: "2026-08-02"
---

# TASK-0064 文書lintの失敗を一回で集約する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

`make lint-docs`の独立した既存検査をすべて一回の起動で実行し、先行検査がFAILしても後続の指摘を同じ出力で得られるようにする。TASK-0061/0063で繰り返した「terminology修正後に初めてtextlint指摘が見える」直列手戻りを、検査の削除・緩和・新規ルール追加なしで解消する。

### 対象と対象外

#### 対象

- terminology validator、Markdown textlint、`git diff --check`の既存3検査を、shell展開を使わない小さなrunnerから固定順で全て実行する。
- 一つ又は複数の検査がFAILしても残りを実行し、各commandのstdout/stderrをその場で表示し、最後にnon-zeroで終了する。
- 全検査PASS時だけzeroで終了し、既存`make lint-docs`と`make check`の入口・検査内容を維持する。
- fake runnerを使うbounded unit testで、先頭FAIL後も後続が実行されること、複数FAIL、全PASS、spawn errorを確認する。

#### 対象外

- lint rule、用語集、対象Markdown列挙、指摘のseverity、`make check`の他targetを変更しない。
- 検査の並列化、cache、retry、自動修正、skip、changed-files限定、外部依存又は新packageを追加しない。
- dev-agent-harness製品コード、runtime、Schema、Task gateの承認条件、candidate commit回数を変更しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: `make lint-docs`はterminology、textlint、`git diff --check`を固定順で一回ずつ実行し、先行FAILでも後続を省略しない。
- [ ] AC-2: 一件以上のFAIL又はcommand起動失敗で最終exitはnon-zero、全PASSでzeroとなり、全commandの既存stdout/stderrが利用者へ見える。
- [ ] AC-3: command/argsは固定配列でshellを介さず、検査削除・緩和、retry/cache/parallel/autofix、秘密又は入力値の新規ログを導入しない。
- [ ] AC-4: unit testはfirst-fail-continues、multiple-fail、all-pass、spawn-errorをfailure-detectし、root `make check`、`git diff --check`がPASSする。
- [ ] AC-5: candidateは承認済み3パス・約250行以内で、dev-agent-harness製品コード、Schema、依存、生成物を変更しない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | root `Makefile`の`lint-docs` target | main `1b393fe` | 現在の3検査と直列停止の再現根拠 |
| REF-2 | TASK-0061/0063 HANDOVER | main `1b393fe` | 同じterminology→textlintの候補固定前手戻りが反復した根拠 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| なし | `ready` | N/A | N/A |

### 許可パス

- `Makefile`
- `scripts/run-doc-lints.mjs`（新規）
- `scripts/task/development-process.test.mjs`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 通常product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | low-risk local process utilityのため`dev-luna`、Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | 既存3commandとNode test runnerがmain `1b393fe`に存在する |
| 生成物の有無と更新方法 | `ready` | MakefileとNode source/testのみ。dependency、code generationなし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0064-aggregate-doc-lint-diagnostics` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

`lint-docs`はMake recipeの各行を順に実行するため、terminology validatorがFAILするとtextlintとdiff checkを実行しない。TASK-0061とTASK-0063の両方でREADMEのterminology修正後に別のtextlint指摘が現れ、重いcandidate `make check`を余計に一回繰り返した。検査自体は品質に寄与するため維持し、診断を一回で集約する実行方法だけを直す。

## 検討すべき設計観点

- runnerは検査結果の意味を解釈せず、固定commandを全て一回実行してexitだけを集約する。
- shell command文字列を受けず、repository inputからcommandを組み立てない。
- output captureによる遅延を増やさず、子processのstdioを継承して進捗と指摘を即時表示する。
- `make check`の検査順序と品質gateは変えず、docs検査内部のfail-fastだけをall-diagnosticsへ変更する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- 未調査

### 判断

- 未調査

### 適用しなかった重要な判断

- なし
