---
task_id: "TASK-0044"
status: complete
completed_at: "2026-08-01T08:27:42Z"
candidate_commit: "caa8071c382a92013008f6d9a900a952c1b26dc9"
---

# TASK-0044 HANDOVER

## 成果

- root `Makefile`の`check` prerequisiteを一行だけ変更し、既存`lint-docs`をbuild/test/言語別lintより前へ移した。
- 新しいrule、script、test又はcommandは追加せず、standalone `lint` targetと全recipeを維持した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|
| base/candidateの`make -n check`比較 | PASS。双方24 commandでmultiset不変。candidateはdocs lint 3 commandを先行し、viewer生成と最終diff checkを末尾に維持 |
| candidate launcherのroot `make check`（一回） | PASS |
| `git diff --check` | PASS |

## 主要な変更

- `check: build test lint`を`check: lint-docs build test lint-core lint-memory lint-governance`へ変更した。
- 差分は`Makefile`だけ、1追加・1削除である。

## 検証結果

- 通常のcandidate checkはdocs lintから開始し、全既存検査をPASSした。
- dry-runではdependency install、terminology validator、`pnpm lint:docs`、文書用diff checkの後にproduct build/testが続き、viewer data生成と最終diff checkが最後に残る。
- fault injectionはDEV時点でdependency markerがなかったため未実施。通常candidate check後にmarkerが存在するため、QAが同一candidateへ一回だけ実施する。

## 判断・既知の制約

- 標準non-parallel `make check`だけを順序保証の対象とし、`make -j`向けのdependency edge又は恒久順序testは追加していない。
- 新しい再利用可能な製品意味はないため、Wikiは更新しない。
