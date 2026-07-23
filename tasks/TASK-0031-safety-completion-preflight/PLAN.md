---
task_id: "TASK-0031"
change_class: product
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "checker、CLI、Make、テンプレート、文書をまたぐfail-closedな完了経路変更であり、予定pathと実差分の対応を一貫して設計する必要があるため"
approved_dev_profile_risk_signals:
  - cross_cutting
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20T09:55:45+09:00"
scope_reconciled_by: "main-agent-sol-high"
scope_reconciled_at: "2026-07-20T10:04:00+09:00"
scope_reconciliation_reason: "既存用語検査が文書・template差分から決定的に生成する用語集と索引を候補へ含めるため。AC・設計・QA期待値は不変更"
planned_implementation_files: 2
planned_implementation_lines: 130
estimate_points: 1
---

# TASK-0031 PLAN

## 受け入れ条件の具体化

| AC | 設計判断 | 主な変更path |
|---|---|---|
| AC-1 | v2の機械可読path契約をpreflightでfail-closedに検査する。 | `check-task.mjs`、`PLAN.md` template |
| AC-2 | `docs/99-glossary-index.md`を生成専用allowlistに置く。 | `check-task.mjs`、開発文書 |
| AC-3 | v2 Doneは第2親候補の実差分を承認済みpath集合へ束縛し、生成宣言の欠落を拒否する。 | `check-task.mjs`、process test |
| AC-4 | version未指定の既存安全契約は現行Done検査を保ち、v2のみ新契約に入る。 | `check-task.mjs`、process test |
| AC-5 | 正常・許可外・生成欠落・実差分逸脱・legacy互換をfixtureで自動化する。 | process test、Makefile |

## 設計決定

1. 安全契約PLANのv2 opt-inを `safety_contract_version: 2` とし、`safety_contract_planned_paths` と `safety_contract_generated_paths` のリポジトリ相対file-path配列を必須化する。空配列は当該種別の変更なしを表す。
2. pathは空、絶対、`..`、directory/globを拒否し、配列内・配列間の重複を拒否する。通常予定pathは既存統制path allowlist、生成pathは生成専用allowlistにそれぞれ一致しなければならない。初期の生成専用allowlistは `docs/99-glossary-index.md` とする。
3. `check-task.mjs --phase preflight` はGit不要でv2 PLAN契約を検査する。`make task-preflight TASK=...` をその公開入口とする。無versionのv2 fieldと未知versionはfail-closedにする。
4. Doneは既存のno-ff、tree/digest、rename/copy、空差分、非`AMD`拒否を維持する。v2だけは全候補差分pathをPLAN二配列のunionに含め、宣言済み生成pathの全てが候補差分にあることを要求する。通常予定pathは未変更でもよい。
5. version fieldのないlegacy安全契約（TASK-0024を含む）は現行固定allowlist Done検査を変更しない。v2の新fieldを無versionで混在させる状態は拒否する。

## 変更予定

| ファイル | 種別 | 概算行数 | 内容 |
|---|---|---:|---|
| `scripts/task/check-task.mjs` | implementation | 110 | v2 validator、preflight CLI、Doneの予定集合/生成path照合、legacy分岐。 |
| `Makefile` | implementation | 3 | `task-preflight` PHONYとCLI呼出し。 |
| `scripts/task/development-process.test.mjs` | test | 150 | v2/legacy fixtureと正負ケース。 |
| `templates/task/PLAN.md` | template | 8 | v2 fieldと記入説明。 |
| `docs/development/development-process.md` | documentation | 14 | preflight、実差分照合、legacy動作。 |
| `docs/development/task-management.md` | documentation | 12 | version、生成path、運用境界。 |
| `docs/glossary.yml` | generated documentation | 0 | 用語検査の決定的な生成結果。 |
| `docs/99-glossary-index.md` | generated documentation | 0 | 用語検査の決定的な生成索引。 |

見積もり対象は実装2ファイル、130行である。

```text
file_score = ceil(2 / 3) = 1
line_score = ceil(130 / 200) = 1
estimate_points = 1
```

## 実装順

1. checkerにversion/path契約の共通validatorとglobal/生成allowlistを追加する。
2. preflight CLIとMake targetを追加し、既存`task-check`の既定挙動を変えない。
3. v2 Doneを候補差分と生成pathへ束縛し、legacy分岐を保持する。
4. fixtureを拡張して正常、許可外、field欠落/重複、生成path欠落、実差分逸脱、TASK-0024互換を検証する。
5. templateと文書を実装に同期し、用語生成で索引が更新される場合は生成結果を含める。

## 失敗時・互換性

preflight失敗はDEV開始前にPLANを修正・再承認する。Doneの計画外差分は後付けallowlistにせず、PLAN再承認または別Taskへ分離する。v2は明示opt-inであり、既存の`make task-check`とversion未指定の安全契約Taskは後方互換に保つ。

## 検証計画

- `node --test scripts/task/development-process.test.mjs`：v2正負例とlegacy互換。
- `make task-preflight TASK=TASK-0031`：新CLI/Make入口。
- `make task-check TASK=TASK-0031` と `make test-process`：既存入口とprocess test。
- `make lint-docs`、`git diff --check`：文書用語、索引freshness、空白。必要時は `scripts/validate-terminology.py --write` 後に再実行する。
- `make check`：製品変更として全build/test/lintを確認する。

## main Agentレビュー

- [x] 受け入れ条件が検証可能である。
- [x] v2 opt-in、生成索引の境界、実差分照合、legacy互換が明確である。
- [x] QA計画を作成できる。
- [x] 見積もりが規則どおりである。
- [x] DEV開始を承認した。
