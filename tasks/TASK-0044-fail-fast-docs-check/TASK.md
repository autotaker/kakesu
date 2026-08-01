---
task_id: "TASK-0044"
title: "make checkの文書lintをfail-fast化する"
status: done
created_at: "2026-08-01"
---

# TASK-0044 make checkの文書lintをfail-fast化する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

標準`make check`で既存の`lint-docs`をbuild、product test、言語別lintより先に完了させる。TASK-0042とTASK-0043ではREADMEの既存用語lint違反が包括検査の終盤で初めて判明し、失敗することが決まっている状態で約40秒のbuild/testを繰り返した。検査基準を増減せず、安価な既存文書lintを先頭へ移して同じfailureを数秒で返す。

### 対象と対象外

#### 対象

- root `Makefile`の`check` orchestrationだけを変更し、`lint-docs`を完了後に既存の`build`、`test`、`lint-core`、`lint-memory`、`lint-governance`、viewer data生成、最終`git diff --check`を実行する順序。
- 標準の非parallel `make check`で、既存各検査を一回ずつ実行し、`lint-docs`失敗時はproduct build/testへ到達しないfail-fast動作。
- `make -n check`によるcommand集合・順序の比較と、`UV=false make check`相当の上限付きfault injectionによる文書lint段階での即時停止確認。
- 最終candidate byteに対する通常のroot `make check`。

#### 対象外

- 新しいlint、rule、glossary項目、恒久test、script、checklist、version field、warning/error条件の追加又は削除。
- `lint-docs`、`lint`、`build`、`test`、各subtargetのcommandや意味の変更。
- `make -j`又は外部orchestratorによるparallel実行順の新規保証。標準`make check`だけを対象にする。
- product source、tools/dev-agent-harness、Task lifecycle script、hook、CI、dependency、generated product、Wikiの変更。

### 受け入れ条件

- [x] AC-1: `make -n check`で`validate-terminology.py`、`pnpm lint:docs`、文書用`git diff --check`が、core/memory/governanceのbuild/test/lintより前に現れる。viewer data生成と最終`git diff --check`は従来どおり最後に残る。
- [x] AC-2: `check`が実行する既存target/command集合は順序以外で増減せず、`lint`その他の公開targetは変更されない。新しいrule、script、test、glossary、文書は追加されない。
- [x] AC-3: 文書lint commandを意図的に即時失敗させた標準`make check`はproduct build/test commandへ到達せずnonzeroで終了する。fault injectionはrepository fileを変更せず、実product command、network、dependency更新を実行しない。
- [x] AC-4: 通常のcandidate launcherによるroot `make check`、`git diff --check`がPASSし、base...candidateの変更はroot `Makefile`だけ、追加＋削除10行以下である。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | root `Makefile` | main `70e14d5`時点の`check: build test lint`、`lint: ... lint-docs` | 現行command集合と遅い実行順 |
| REF-2 | TASK-0042 HANDOVER | completion `387fe9f` | README用語lintがcandidate前に二度遅延FAILした再発記録 |
| REF-3 | TASK-0043 HANDOVER | completion `70e14d5` | 同じREADME用語lintが包括検査終盤で再発した記録 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0043 | `ready` | REF-3 | 反復failureを具体的根拠とし、検査基準でなく実行順だけを変える |

### 許可パス

- `Makefile`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | dry-runとcommand overrideのfault injectionだけ。実network、dependency更新、外部作用なし |
| 依存状態と参照 | `ready` | TASK-0042/0043で同じ遅延failureが反復し、現在のMakefile順序をREF-1で固定 |
| 生成物の有無と更新方法 | `ready` | root Makefile一つだけ。生成物なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0044-fail-fast-docs-check` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

- TASK-0043はplanning開始時点でready。実装や品質判定は変更せず、同Taskで観測した反復遅延を実行順だけで解消する。

## 背景

root `Makefile`は`check: build test lint`であり、`lint-docs`は`lint`の最後にある。このため文書だけの既知FAILでもGo/Python/Rust build/test、tabletop、process test、各言語lintを先に完了する。人が候補前に手動実行する運用では再発したため、標準entrypoint自身をfail-fastにする。

## 検討すべき設計観点

- `lint-docs`を新たに複製せず、`check`の最初のpreconditionとして一回だけ完了させる。
- `lint` aggregateはstandalone利用者向けにそのまま残す。`check`だけが`lint` aggregateを展開し、重複する`lint-docs`を除いた既存3 lint targetを後段で実行する。
- fault injectionは`UV=false`等のcommand overrideで最初の文書lint commandを失敗させ、build/test commandが出ないことだけを観測する。実fileへlint違反を作らない。
- 永続的な順序assertion testは、この単純なMakefile構造以上の保守対象になるため追加しない。

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] planning/candidate/completionの3 commit経路と`make check`を満たしている。
- [x] 同一candidateの独立REVIEW/QAを完了し、post-merge `task-check`をcompletion後に実行する。
- [x] 新しい品質rule又はSemantic知識を追加していないため、glossary/Wiki/恒久testを更新していない。

## 関連コンテキスト

### 意味 Wiki

- 更新なし。これは既存検査の実行順最適化であり、新しい再利用可能な製品意味を追加しない。

### 判断

- 反復した遅延failureを、手動preflight指示ではなく標準`make check`のfail-fast順序で解消する。

### 適用しなかった重要な判断

- 新しいdocs precheck、候補専用rule、恒久順序test、glossary例外は追加しない。
