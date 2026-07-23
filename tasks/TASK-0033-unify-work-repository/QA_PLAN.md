---
task_id: "TASK-0033"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-23T10:04:33+10:00"
revision: 2
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0033 QA PLAN

## 方針

この計画は TASK の `planning input packet` のみを入力とする。各観測は、DEV が固定する同一の `candidate_commit` と `candidate_tree` に束縛し、HANDOVER のケース別コマンド、fixture/environment、cache、exit、digest、未実施理由を独立に突合する。cutover 後はコード candidate commit/digest と bootstrap evidence commit/digest の組を同一評価対象として突合する。候補、bootstrap、tree、移行元 REF-2、または PR 管理対象 path の digest が一致しない場合は、どの `evidence-review` も PASS にしない。

## 前提と環境

- QA 開始条件は、candidate-bound HANDOVER と対象 Task の `make task-check`、`make check` の実行記録があること。REVIEW の結果は開始条件にしない。
- fixture は一時的な bare remote と単一 checkout を使い、REF-1/REF-2 相当の固定入力、clean/dirty、競合、赤 CI、既取込/未取込の状態を明示する。ネットワーク、認証、GitHub 設定を模擬してはならないケースは `live-e2e` とする。
- GitHub actor `autotaker`、merge commit、auto-merge と required context（`Full check`、`Task check`、`Scope check`）は固定済みである。ruleset 実設定と candidate 上の check run は QA-005〜QA-007 の `live-e2e` で確認し、固定値との差異があれば Main は dependency-ready reconciliation と QA_PLAN 再承認なしに DEV/QA を進めない。
- public repository では認証情報、`auth.json`、token、Actions secret の値をログ・fixture・成果物へ出さない。存在検査は secret-free な path/設定/ログの否定観測で行う。
- bootstrap は移行 validator 固定後に Main が直接 push した evidence commit であること、移行元 work repository が freeze され以後の新規証跡書込みがないこと、製品 main が正本へ切替済みであることを確認する。失敗時は証跡正本を曖昧にせず、開始前状態へ rollback できる根拠がなければ blocked とする。

## 受け入れ条件との対応

| ケースID | AC-ID | 観測方法 | `qa_execution_mode` / 理由 | 必要証跡 | fail-closed |
|---|---|---|---|---|---|
| QA-001 | AC-1 | candidate と単一 checkout で運用入力・schema/viewer/hook を読取・更新する対象テストを独立再実行し、実行可能 path、文書、設定を `../agent-harness-work` と外部 `WORK_ROOT` 必須参照について機械探索する。 | `focused-rerun` / 固定 candidate と隔離 fixture で完全再現できる。 | 探索 pattern と対象外、単一 root fixture、更新後の checker exit、生成 index/digest。 | 参照残存、外部 root を与えないと失敗、欠落 path、探索対象不明、test 弱体化は FAIL。 |
| QA-002 | AC-2 | clean main fixture で `task-start` を一回実行し、生成、検査、commit/push、branch、sparse worktree を観測する。個別 publish 入口を途中失敗させ、partial state の検出・回復を確認する。 | `focused-rerun` / bare remote を含む bounded deterministic fixture で成功と失敗原子性を再現できる。 | 実行前後 HEAD/remote refs、生成物 digest、worktree list、hook/checker exit、失敗注入と cleanup 記録。 | dirty 開始、partial commit/push、作成物と branch/worktree の不整合、retry 無制限は FAIL。 |
| QA-003 | AC-3 | bootstrap evidence commit の main 直接 push、work repository freeze、製品 main への証跡正本切替を確認したうえで、作成済み Task worktree の sparse list と filesystem を確認して main 管理証跡/運用 index 非 checkout を検証する。main root への明示 routing、action ごとの allowlist/共有 path、lock 競合、non-fast-forward race（2回上限）を fixture で独立再実行し、bootstrap failure 時の rollback 根拠も確認する。 | `focused-rerun` / Git lock と remote race は隔離 bare remote の決定的競合注入で受理真実を再現できる。 | bootstrap evidence commit/digest と移行 validator exit、freeze 時点/書込監査、製品 main 正本確認、rollback 手順、sparse-checkout 設定・列挙、main root への書込先、action別 staged path、lock owner/log、各 retry の recheck/exit、remote ref と digest。 | bootstrap digest 不一致、freeze 後の work 書込、正本不明、rollback 根拠なし、除外 path が checkout、許可以外が stage/commit、共有 path を別 action が更新、lock bypass、競合で自動解決/3回目 retry は FAIL。 |
| QA-004 | AC-4 | main 直 push workflow を fixture event と workflow 定義の双方で検査し、証跡 scope、Schema、link、状態遷移の positive/negative と製品 path 混入を独立再実行する。CI 実行ログ/定義から write token・commit/push・再帰 trigger がないことを確認する。 | `focused-rerun` / read-only checker と event matrix は hermetic fixture で bounded に再現できる。 | workflow digest、event/permission/concurrency 定義、positive/negative fixture exits、CI command log、対象 path manifest。 | product path を許容、schema/link/state error を通過、write command/token、bot push による自己再起動、検査未実行は FAIL。 |
| QA-005 | AC-5 | コード candidate commit/digest と bootstrap evidence commit/digest の composite candidate、candidate tree、PR 管理対象 path digest から REVIEW/QA 証跡を独立監査する。branch が bootstrap 後の製品 main を取り込んだ後、PR 差分から main 管理証跡が消えたことを確認し、main 直管理 path を PR に混入させた negative を実行する。両 PASS の後、承認済み GitHub test repository で `task-pr` を実行し ready PR、merge-commit auto-merge を観測する。 | `live-e2e` / PR 作成、actor 権限、auto-merge は実 GitHub auth と repository 設定に依存する。 | composite candidate の全 commit/digest、candidate/tree と path digest、branch の main 取込前後 diff、両 result の開始独立性、PR URL/number、ready 状態、auto-merge method/status、negative PR 拒否、cleanup。 | pending/無効 auth、composite candidate/tree/digest 不一致、branch 更新後も main 管理証跡が PR diff に残る、片方 PASS 前の PR、direct path 混入許容、merge method 不明は blocked/FAIL。 |
| QA-006 | AC-6 | QA-005 の test PR に required checks が実際に付与され、merge candidate で full/task/scope check が走ることを確認する。1件を失敗させ auto-merge 保留、全成功後のみ merge を観測する。 | `live-e2e` / GitHub ruleset/required check と auto-merge 判定は外部 service の実設定でのみ確認できる。 | repository settings snapshot（secret-free）、check run 名/commit SHA/exit、失敗時 merge 保留、成功時 merge event、cleanup。 | required check 名/対象 SHA 不一致、失敗しても merge、check を bypass、権限/設定未確認は blocked/FAIL。 |
| QA-007 | AC-7 | merged PR に対する `pull_request.closed` workflow を承認済み test repository で一度発火し、main の `merged_commit`/状態更新を確認する。同一 event 再送、未 merge close、push event、`workflow_run` 不在、Task/PR concurrency を negative で観測する。 | `live-e2e` / webhook event、GitHub concurrency と main 更新は環境固有の外部作用である。 | event payload/PR、main before/after digest、更新 commit、duplicate/no-op log、workflow graph、concurrency run IDs、cleanup/rollback。 | 非 merged close が書込、二重 commit、push CI 書込、workflow_run 連鎖、concurrency 未適用、cleanup 不明は blocked/FAIL。 |
| QA-008 | AC-8 | clean/dirty/conflict/赤 CI、未取込/既取込 Task を持つ main fixture で `sync` を独立再実行する。実認証済みの隔離 local Codex 環境では Wiki ingest、done 化、receipt、worktree/branch cleanup、再実行 no-op を確認し、`FAST=1` が同期のみを行うことを確認する。 | `live-e2e` / local Codex Pro 認証による実 Wiki 取込と cleanup は外部認証・副作用を伴う。 | local environment identity（secret-free）、remote/HEAD、ingest receipt/digest、Task/backlog/Wiki 状態、worktree/branch before/after、FAST result、rollback/cleanup。 | dirty/conflict/red CI で継続、FAST が ingest/done、既取込で再更新、認証情報を保存、cleanup/rollback 不明は blocked/FAIL。 |
| QA-009 | AC-9 | REF-2 の固定 snapshot にある historical Done Task 32件と current TASK-0033 evidence、および candidate の migration manifest から、backlog、Wiki、判断、Lap30、viewer、schema/hook の件数と content digest を独立に再算出する。bootstrap evidence commit/digest がこの照合結果に束縛され、work repository freeze 後に製品 main を正本としていることを確認する。既存 Done/公開 Wiki/Lap30 の semantic diff、新旧 checker、bootstrap rollback 根拠を確認する。 | `focused-rerun` / 固定入力の count/digest と semantic diff は bounded deterministic に再現できる。 | REF-2 revision、historical Done Task 32件と current TASK-0033 evidence を区別したカテゴリ別 count/path manifest、旧新 digest、semantic-diff 出力、両 checker exit、bootstrap evidence commit/digest、freeze/正本切替監査、rollback 手順、除外リスト。 | 件数/digest 不一致、bootstrap が照合外、freeze 後の work 書込、正本切替不明、rollback 根拠なし、許容外意味差、未追跡 migration 入力、旧成果物を上書き、checker 省略は FAIL。 |
| QA-010 | AC-9 | QA-009 の照合 PASS 後のみ、承認済み GitHub public repository で旧 `autotaker/kakesu-work` の archive を実行し、公開 repository に secret/AI auth が置かれないことを archive 前後の設定・tree・Actions artifact/log で観測する。復旧手順（unarchive と migration rollback decision）を dry-run で検証する。 | `live-e2e` / archive は外部の不可逆に近い repository 操作であり、public visibility/security boundary は実環境確認が必要である。 | archive actor/時刻、archive 前 QA-009 digest、public tree/Actions/secret-free scan、archive API/UI result、unarchive authority と rollback 手順。 | digest 照合前 archive、secret/auth 値または配置検出、public repo 誤対象、unarchive/rollback 権限・手順不明は blocked/FAIL。 |

## 境界・異常・回帰

- `evidence-commit` は action 名と許可 path の対応を表で照合し、対象 Task 以外、shared path、生成 index、PR path の各境界を negative にする。lock 取得失敗、remote non-fast-forward、rebase 後 digest 変化は安全停止を期待し、成功へ丸めない。
- sparse worktree では path 非存在だけでなく、`git sparse-checkout` 設定、親 main root の意図しない書換え、Agent の明示 root 指定を観測する。
- workflow は `push`、`pull_request`、`pull_request.closed` の event matrix、permissions、concurrency、bot/actor、書込み command を相互照合する。read-only CI、recursive trigger 防止、postmerge idempotency は個別に negative を持つ。
- public repo security scan は文字列/ファイル名/fixture/Actions log/artifact を対象にし、実 secret 値は表示・保存・送信しない。検出時は値を伏せて FAIL とし、archive は実行しない。
- 高リスク、証跡欠落、digest/commit/tree 不一致、test 削除・弱体化、影響不明は `evidence-review` PASS にしない。本計画の `live-e2e` は隔離・承認済み環境と安全な cleanup がない限り blocked とする。
- `qa_carry_forward` は `QA_RESULT.md` の CF-1〜CF-7 が全て満たされる場合だけ Main が選択できる。認証認可、秘密、Schema/設定/依存、並行性、lifecycle/persistence/error/fail-closed、試験弱体化、期待/範囲変更、QA FAIL、案/tree 不一致は carry-forward を禁止する。

## 実施不能時の扱い

- GitHub actor/auto-merge/ruleset/required check、隔離 test repository、または local Codex Pro の認証・安全 cleanup が unavailable なら、該当 `live-e2e` は blocked のままとし、fixture や証跡監査で PASS を代替しない。
- 実行できない場合はケース ID、candidate、環境、未実施 command、阻害要因、未確認の受入真実、cleanup 状態を `QA_RESULT.md` に残す。FAIL の原因（実装、fixture、認証/権限、GitHub service、手順/証跡）は根拠とともに分類し、DEV 責任と自動帰責しない。

## 案とマージ後確認

- DEV は candidate ごとに `candidate_commit`、`candidate_tree`、PR 管理対象 path digest と、cutover 後は bootstrap evidence commit/digest を含む composite candidate、全ケースの command/test、fixture/environment、cache、exit、artifact digest、未実施理由を HANDOVER へ固定する。
- QA は同一 composite candidate から REVIEW と並行・独立に開始する。修正で candidate または bootstrap evidence が変われば、Main が再束縛し、影響ケースを再実行する（本 Task の security、auth、workflow、schema、concurrency、state 変更には carry-forward を適用しない）。
- `merge_tree == candidate_tree` でも QA-005〜QA-008、QA-010 は環境依存のため、merge 後にケース単位で確認する。merge tree 不一致なら全ケースの影響を fail-closed に再評価する。

## 実装後の再確認

- [ ] 実装差分と candidate-bound HANDOVER を確認した。
- [ ] 操作手順を現行実装に合わせた。
- [ ] 期待結果または試験範囲の変更有無を確認した。
- [ ] 期待結果または範囲を変更した場合、main Agent の承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-22 | QA Agent | TASK packet 起点の独立初版 | `2026-07-23 main-agent-sol-high approved` |
| 2 | 2026-07-23 | QA Agent | cutover bootstrap の composite candidate、freeze、PR差分negative、rollback観測を追加 | `2026-07-23 main-agent-sol-high approved` |
