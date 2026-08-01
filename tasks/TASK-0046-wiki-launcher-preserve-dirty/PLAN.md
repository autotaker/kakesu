---
task_id: "TASK-0046"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T09:35:30Z"
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "変更は既存のNode.js launcher、共有Git helper、temporary Git fixtureに限定され、外部連携・Schema・依存追加・権限境界の変更を含まず、受け入れ事実をhermeticに再現できるため。"
approved_dev_profile_risk_signals: []
dev_profile_promotions: []
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0046 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | edit-only の起動前に、HEADに加え index・tracked worktree・untracked file の状態をpath単位で固定し、起動後状態との差分から子所有の変更集合だけを導出する。同一pathへの上書きは集合差ではなく状態比較で子変更として扱い、開始前dirty pathと衝突した場合は安全に併存できないため成功扱いにしない。 | `scripts/task/run-wiki-agent.mjs`、`scripts/task/agent-routing.mjs`、`scripts/task/development-process.test.mjs` | snapshot helperを定義し、launcherのedit-only分岐で取得してから、allowlist判定とemit対象を子差分へ切り替える。fixtureで成功時の開始前状態と子の許可path変更を比較する。 | 状態を安全にsnapshotできないfile種別、比較不能なGit状態、または開始前dirty pathとの衝突はfail-closedにする。開始前状態は変更・stage・削除しない。 |
| AC-2 | edit-only専用restoreを共通clean-start rollbackから分離する。子の変更、追加untracked、stage、HEAD driftを除去し、固定した開始前状態をindexを含めて復元する。 | `scripts/task/run-wiki-agent.mjs`、`scripts/task/agent-routing.mjs`、`scripts/task/development-process.test.mjs` | 失敗を検出した後、edit-onlyなら専用restore、clean開始なら既存`rollbackWorkRepository`を選択する。各異常をtemporary Git fixtureで検証する。 | restore自体の例外は元の失敗を保持して区別可能なrollback失敗として合成し、成功のevidenceを出さず終了する。 |
| AC-3 | stderrを公開境界の前に純粋helperで固定prefix、上限長、token形式redactionへ正規化する。子のexit codeは構造化結果に残し、raw文字列は例外・JSONのどちらにも渡さない。 | `scripts/task/run-wiki-agent.mjs`、`scripts/task/agent-routing.mjs`、`scripts/task/development-process.test.mjs` | normalizerを追加して子非zeroの失敗生成とlaunch evidenceの双方が同じ安全な診断だけを使うようにする。短縮・redactionを直接テストする。 | normalizerが想定外入力を受けた場合はraw値を代用せず、安全な固定診断に縮退して失敗を継続する。 |
| AC-4 | edit-onlyの新しい状態保存経路をcommit-modeから切り離し、clean開始時の既存rollbackと親だけがcommitする流れを維持する。回帰を単一のtemporary Git fixture群に集約する。 | `scripts/task/run-wiki-agent.mjs`、`scripts/task/agent-routing.mjs`、`scripts/task/development-process.test.mjs` | helper単体の状態比較・復元を先に固め、launcher分岐を接続し、最後に既存clean-start/commit-mode契約と指定された境界事例を回帰化する。 | commit-modeまたはclean-startの既存期待値が変化した場合はedit-only実装をそのまま進めず、原因を切り分けて既存経路を復元する。 |

## 関連Wikiと判断

- [Development Task](../../wiki/semantic/concepts/development-task.md): 製品変更をTask branch/worktreeとPLAN・DEV・QAの順で扱う境界に従う。
- [Work Repository Boundary](../../wiki/semantic/schemas/work-repository-boundary.md) と [DECISION-0002](../../wiki/decisions/DECISION-0002-work-repository-and-wiki-ownership.md): Wiki launcherはmain管理のwork repositoryでロックを保有して実行し、Task/Wikiの所有範囲を越えない。
- [Codex Agent Model Routing](../../wiki/decisions/DECISION-0003-codex-agent-model-routing.md): 子のdriftをfail-closedに扱う既存原則は維持する。ただしdirty edit-onlyの復元対象はclean-start前提の共通rollbackと別にする。
- Wiki本文・Decision・receipt・indexの変更は計画しない。

## 補足設計

### 代替案と不採用理由

- 起動前のdirty全体をstash、commit、またはarchiveしてから子を実行する案は、ref/index副作用と復元経路を増やし、既存のcommit-mode clean-start契約を広げるため採用しない。
- edit-onlyを常にclean開始へ制限する案は、MainがTask証跡を保持したままWiki Agentへ本文更新を委譲する既存用途を失わせるため採用しない。
- 全repository cloneをrollback sourceにする案は、必要なpath状態より広く、コストと故障面を増やすため採用しない。
- raw stderrをそのままevidenceへ載せる案は、診断ログからの秘密情報露出を許すため採用しない。

### 責務・境界・不変条件

- Wiki launcher親がlock、snapshot、子結果の検査、scope判定、restore、evidence出力を所有する。子は既存どおりstage、commit、`.git`書込みを許可されない。
- edit-onlyの子所有変更は開始前状態との差分で決定し、開始前dirty pathがallowlist検査・親stage・rollback削除の対象へ混入しない。同一pathを子が上書きした衝突は、開始前bytesと子結果を推測でmergeせずrestoreして失敗にする。
- snapshotはtracked pathの存在・bytes・modeとindex状態、およびuntracked fileの存在・bytes・modeを表現する。directoryは配下file単位で扱い、安全に表せないentryはfail-closedにする。
- restoreは子が作ったuntracked fileを除去した後に開始前状態を再構成し、開始前のtracked削除とstage/unstagedの併存を含めて確認する。共通`rollbackWorkRepository`のclean-start事後条件は変更しない。
- 公開する子失敗診断はboundedかつredactedな文字列だけとし、exit code以外のraw stderrを保持・再送しない。

### 移行・互換性

- データ移行、Schema、依存、環境変数、CLI引数、Wiki形式の変更はない。
- `--commit false`だけがdirty開始を扱う。commit-modeは従来どおりclean開始を要求し、既存の親commit・post-validation経路を保つ。

## 変更予定

| パス | 変更内容 |
|---|---|
| `scripts/task/agent-routing.mjs` | edit-onlyの状態snapshot、子差分導出、復元、stderr正規化の再利用可能なpure helperを追加し、既存clean-start rollbackは維持する。 |
| `scripts/task/run-wiki-agent.mjs` | edit-only時にsnapshotと子差分のみの検査・emit・restoreを接続し、子非zeroの診断を正規化する。commit-modeの制御経路は変更しない。 |
| `scripts/task/development-process.test.mjs` | temporary Git fixtureで成功・異常時の状態保存と、stderr redactionを直接・launcher境界の両方で回帰検証する。 |

## 実装手順

1. `agent-routing.mjs`に、Git fixtureから直接検証可能なedit-only状態表現、差分導出、復元、および安全なstderr要約を設計する。clean-start用`rollbackWorkRepository`の契約は変更しない。
2. `run-wiki-agent.mjs`でedit-onlyの開始前snapshotを取得し、子終了後のscope判定を子所有集合へ置換する。失敗時は分岐したrestoreを実行し、公開エラーとevidenceにnormalizerの出力だけを渡す。
3. `development-process.test.mjs`のtemporary Git fixtureで、同一path上書き、staged+unstaged同居、tracked削除、untracked保持、子untracked除去、scope/stage/commit drift、validation失敗、stderrの切詰め/redactionを検証する。commit-modeとclean-start rollbackの既存回帰も維持する。
4. DEVはTask worktreeで対象製品差分だけを作成し、candidate-commitの一回の`make check`を実行する。独立REVIEW/QAは同じcandidateを評価し、Mainがcompletion gateと必要なマージ後確認を行う。

## 検証計画

- 開発中は `node --test scripts/task/development-process.test.mjs` を使い、pure helperとtemporary Git fixtureの失敗原因を限定する。
- candidateではDEVが `make check` を一回実行し、対象testを含む既存検査を通す。テストは`mkdtemp`配下のGit fixtureだけを操作し、Codex CLI、network、実認証情報を使用しない。
- QA_PLANはTASK-firstの独立計画として、状態保存・rollback・redaction・commit-mode回帰をcandidate-bound evidence-reviewまたはfocused-rerunに割り当てる。実環境依存の確認が必要と判明した場合は、そのケースをlive-e2eとして明示し、代替PASSを用いない。
- 失敗は実装不具合と即断せず、fixture、test、環境、既存契約のいずれに属するかをQAガイドラインに従い分類する。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0046`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
