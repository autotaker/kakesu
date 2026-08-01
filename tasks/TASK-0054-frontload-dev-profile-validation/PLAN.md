---
task_id: "TASK-0054"
change_class: "product_change"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "既存unified lifecycleのtransaction境界とvalidator正本を変更せず、最小のphase接続とprocess testでACを満たせる。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T15:29:05Z"
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0054 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `validatePlanningState` が既存 `validateDevSelection` を一回だけ呼ぶ。profile集合・理由・risk signals・promotion検証は複製せず、completion側は保持する。単一callはcounter seamを追加せず、source上の一箇所の呼出しと既存挙動で監査する。 | `scripts/task/unified-lifecycle.mjs` | planning入力を検証する既存位置で、Git操作の前に接続する。 | 既存validatorのerror codeで即時失敗し、transactionへ進まない。 |
| AC-2 | 失敗をGit変更の前に確定させ、rollbackや補償処理を追加しない。 | `scripts/task/unified-lifecycle.mjs`, `scripts/task/unified-lifecycle.test.mjs` | validator接続後、欠落profileのprocess integrationを追加する。 | main/Task branch/worktree HEAD、index、dirty planning入力の不変をassertし、commit/push/fast-forwardなしを確認する。 |
| AC-3 | 有効なprofileを既存planning fixtureの既定入力として明示し、既存transactionのcommit/fast-forward経路を変えない。 | `scripts/task/unified-lifecycle.test.mjs` | fixtureを有効化して既存成功ケースを維持する。 | 成功時は従来のplanning commit一件とTask branch fast-forwardを既存経路で検証する。 |
| AC-4 | TASK-0053と同じ三frontmatter項目がまとめて欠落するplanning failureを一件検査する。個別errorは既存`validateDevSelection` unit matrixに委ね、three-commit lifecycle testを削除・弱体化しない。 | `scripts/task/unified-lifecycle.test.mjs` | valid fixture更新後に、三項目まとめて欠落する一件と不変性assertを追加する。 | 既存validator由来のfailureと不変性を一件で観測する。 |
| AC-5 | 許可された二pathだけに限定し、追加・削除の合計を300行以下に保つ。 | 許可二pathのみ | focused process test、root `make check`、`git diff --check`を実行する。 | 失敗は該当検査結果として分類し、範囲外変更やgate追加で代替しない。 |

## 設計境界

- `validateDevSelection` はDEV profile契約の唯一の正本のままとし、planningは呼出しphaseだけを前倒しする。単一callはcounter等の新しいtest seamを設けず、source上の一箇所の呼出しと既存挙動で監査する。
- validationはGit stage、commit、push、Task branch fast-forwardより先に一回実行する。失敗時に状態を戻す処理は不要であり、dirtyなplanning入力を含めて保持する。
- completionのvalidationは削除・変更しない。candidate/completion、Wiki launcher、backlog状態遷移、Schema/frontmatter field、gate/check commandは対象外である。
- 代替のvalidator複製、新field、templateへの固定値追加、completion側の削除はいずれもACの正本性または防御層を損なうため採用しない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `scripts/task/unified-lifecycle.mjs` | `validatePlanningState` から既存validatorを一回呼び、planning transaction開始前に失敗を返す。 |
| `scripts/task/unified-lifecycle.test.mjs` | planning fixtureへ有効なprofile三項目を設定し、三項目がまとめて欠落する一件のearly failureとGit/input不変性を検証する。既存three-commit lifecycle testを維持する。 |

## 実装手順

1. planning state validationへ既存validator呼出しを接続し、Git変更前の失敗境界を維持する。
2. fixtureの有効profile三項目を設定し、三項目まとめて欠落する一件のprocess integrationと不変性観測を追加する。
3. 既存one-commit/no-ffおよびthree-commit lifecycleの検査を維持したまま、差分・範囲・標準検査を確認する。

標準経路はplanning、candidate、completionの3 commitsを維持する。本Taskはplanning前の拒否phaseだけを変え、candidateの再作成や追加commitを設計しない。

## 検証計画

- focused process test: planningの三項目まとめて欠落する一件について、既存validator由来のerror、main/Task branch/worktree HEAD、index、dirty planning入力、commit/push/fast-forward未実施を確認する。
- existing coverage: 個別errorを検証する`validateDevSelection` unit matrix、valid planningの一commit/no-ff経路、three-commit lifecycle testを維持する。single-callはsource上の一箇所と既存挙動で監査し、counter seamは追加しない。
- standard checks: root `make check`、focused process test、`git diff --check`。追加＋削除は300行以下、変更先は許可二pathだけと確認する。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0054`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
