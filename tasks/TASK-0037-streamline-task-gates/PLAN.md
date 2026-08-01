---
task_id: "TASK-0037"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "現mainから不要なWiki自動ingest、未使用の見積算術、数値見積templateを削除する5ファイルの局所差分で、runtime・Credential・network・OS境界を変更しないため"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T04:23:26Z"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T04:23:26Z"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
---

# TASK-0037 PLAN

## 方針

この redo は現 main HEAD を baseline とし、既存の TASK-0037 実装を再実装しない。差分は、optional Wiki に反する legacy cleanup の自動 ingest、不要な見積算術 helper/test、PLAN template の見積書式だけに限定する。

標準経路は planning、単一 product candidate、candidate を第2親に持つ completion no-ff merge の三 commits とする。独立 REVIEW/QA、role・権限・秘密境界、原子的な planning/completion、candidate-bound root check の一回実行、focused test、必要時 live E2E、未検証差し替え拒否は現 HEAD の契約として維持する。`agent-routing` の canonical route 上書き拒否と、起動後 observed model/effort mismatch を warning として記録する現行 AGENTS/docs 契約は別の責務であり、変更しない。

## AC対応

| AC-ID | 現 HEAD との差分または維持判断 | 直接の failure と確認 |
|---|---|---|
| AC-1 | 現 HEAD の planning-gate transaction と変更済み subset の検証を維持する。 | planning scope/state の既存 focused tests を監査する。 |
| AC-2 | 現 HEAD の main-side HANDOVER、単一 product candidate、atomic no-ff completion を維持する。 | candidate/merge/rollback の既存 fixture を監査する。 |
| AC-3 | 現 HEAD の lock、scope、validation、commit の単回実行と staged-content digest fail-closed を維持する。 | parent/hook の同一内容・不一致の既存 focused tests を監査する。 |
| AC-4 | 現 HEAD の HANDOVER 一箇所による candidate 固定と Git 導出を維持する。 | candidate-side evidence と不一致を拒否する既存 fixture を監査する。 |
| AC-5 | 独立 Reviewer/QA、candidate-bound DEV check 監査、focused rerun を維持する。 | identity/PASS/candidate 証跡の既存 checker tests を監査する。 |
| AC-6 | 差分 B: `estimatePoints` とその算術・上限 tests を削除し、見積を実装 gate から完全に外す。backlog の `estimate_points` は viewer と参考情報として維持する。 | 予定行数/ファイル数から points を計算又は上限で停止する未使用 helper/test が残らないことを focused source/test audit で確認する。 |
| AC-7 | 差分 A: legacy qa→done cleanup で Wiki agent/receipt を自動生成・要求しない。再利用可能な知識がある場合だけ明示的な Wiki 更新を completion に含める現 HEAD の契約を維持する。 | receipt がなく、Wiki agent が利用不能でも legacy qa task が cleanup を経て done になれることを syncMain fixture で確認する。 |
| AC-8 | 現 HEAD の CF 廃止と candidate 変更時の影響ベース rerun を維持する。 | 旧 QA の無根拠転用を拒否する既存 process test を監査する。 |
| AC-9 | 現 HEAD の最小 QA 証跡と、形式だけの artifact digest 要求をしない契約を維持する。 | 最小証跡の受理と candidate 不整合の拒否を既存 checker test で監査する。 |
| AC-10 | 現 HEAD の observed model/effort warning と、role/authority/sandbox 境界不明の停止を維持する。canonical route override 拒否は caller 入力の安全境界なので維持する。 | routing と role-boundary の既存 process tests を監査する。 |
| AC-11 | 現 HEAD の task-start、planning fast-forward、単一 candidate、allow-merge 伝播、三 commits 経路を維持する。 | lifecycle/scope の既存 focused tests を監査する。 |
| AC-12 | 差分 A/B/C の focused tests と Task/root checks を実行する。数値見積や形式の追加 check は導入しない。 | 変更箇所の test failure と `make task-check TASK=TASK-0037`、candidate-bound DEV `make check` の結果で確認する。 |
| AC-13 | 現 HEAD の既存 retrospective による低価値 rule の削除/警告化を維持する。差分 A/B/C は不要 rule を削るもので、新しい rule、checklist、version、Task、commit を作らない。 | 変更差分と開発文書の既存契約を監査し、追加 gate がないことを確認する。 |

## 変更予定

| パス | 変更内容 |
|---|---|
| `scripts/task/unified-lifecycle.mjs` | `syncMain` の legacy qa→done recovery から receipt 探索と Wiki agent ingest を除く。dirty worktree の安全確認、done 状態への遷移、通常 cleanup は維持する。 |
| `scripts/task/unified-lifecycle.test.mjs` | receipt 不在かつ Wiki agent が使えない fixture で、legacy qa task の sync cleanup が receipt を要求も生成もせず done まで進むことを確認する。 |
| `scripts/task/lib.mjs` | 未使用の `estimatePoints` helper を削除する。 |
| `scripts/task/development-process.test.mjs` | `estimatePoints` の算術と上限を検査する import と tests を削除する。backlog `estimate_points` を利用する fixture/viewer 用データは変更しない。 |
| `templates/task/PLAN.md` | 概算行数列、見積 section、見積規則 checklist を除き、変更予定表を path と変更内容だけにする。 |

上記以外のファイルは変更しない。

## 実装手順

1. `syncMain` の legacy recovery から自動 Wiki ingest と receipt 条件を除く。qa→done 後の worktree cleanup 前提は変えない。
2. 同じ sync fixture に、receipt 未作成・Wiki agent 非依存で done へ遷移し、receipt が生成されないことを確認する focused case を追加する。
3. 未使用の見積算術 helper と、その算術/上限 test だけを削除する。backlog の `estimate_points` データや viewer 契約には触れない。
4. PLAN template を変更 path と内容だけの表へ縮小し、数値見積の本文・review checklist を削除する。
5. candidate では差分に対応する focused tests と root `make check` を実行し、candidate-bound root check の一回実行証跡を確認する。Main は completion transaction/merge 後に `make task-check TASK=TASK-0037` を実行する。環境依存の受け入れ条件は追加しないため live E2E は不要とする。

## 検証計画

- `syncMain` focused fixture: legacy `qa` task に receipt がない状態を作り、Wiki agent 実行を許さない環境でも sync が status を `done` にし、receipt を作らず cleanup を続行することを確認する。
- development-process focused tests: `estimatePoints` の import/test がなく、backlog `estimate_points` を使う既存 fixture が残ることを確認する。
- template audit: PLAN template に概算行数列、見積 section、見積規則 checklist がなく、変更 path/内容表だけが残ることを確認する。
- existing-contract audit: lifecycle/checker/routing の既存 tests が、candidate 固定、独立 REVIEW/QA、role/authority boundary、canonical route override 拒否、observed mismatch warning を引き続き扱うことを確認する。
- candidate 上で変更した focused tests と root `make check` を実行する。Main は completion transaction/merge 後に `make task-check TASK=TASK-0037` を実行する。QA は DEV の同一 root command を重複実行せず、candidate-bound 証跡を独立監査する。

## 代替案と不採用理由

- legacy cleanup のために receipt を空ファイルで生成する案は、optional Wiki を実質必須化するため不採用。
- `estimate_points` を backlog から削除する案は、viewer と参考情報の用途を壊すため不採用。
- agent-routing の route override 拒否や observed mismatch warning を削る案は、本 Task の残存不適合ではなく安全境界を弱めるため不採用。
- 現 HEAD の lifecycle/checker/docs を再変更する案は、既に満たす AC へ不要な回帰リスクを持ち込むため不採用。

## 未解決事項

- なし

## main Agentレビュー

- [x] 現 HEAD baseline と差分 A/B/C の境界を確認した。
- [x] 五つの変更対象以外を計画に含めていない。
- [x] 各差分に直接の failure と focused test/監査を対応させた。
- [x] QA_PLAN が TASK-first で独立作成されている。
- [x] DEV 開始を承認した。
