---
task_id: "TASK-0029"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
revision: 1
implementation_reviewed_at: "2026-07-20 corrected candidate 864f455b563a6fffb043ed297d5cb10e3849b988"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0029 QA_PLAN

## 独立性と実行ゲート

初版は `TASK.md` だけから作成した独立QA計画である。2026-07-20に完成したPLANとの突合では、TASK-firstの受け入れ条件、境界、モードを維持した。PLANが具体化したrecovery integrity fail-fast、workspace参照、event sequence gapなし、SQLite v3 migration、検証コマンドを追加観測として採用した。実装又は後続PLANが受け入れ条件、範囲、期待結果、またはモードを意味的に変更する必要があれば、本書を改訂してMain再承認を要する。実装で吸収してはならない。

DEV開始条件は、TASK-0027/0028の独立REVIEW/QA PASS、Main merge、および各 `candidate_tree == merge_tree` の確認である。特にTASK-0028 merge後の実際のStore/lifecycle API、schema migration version、typed conflict、event sequence契約が、本計画の前提と異なるなら、DEVを開始せずPLAN/QA_PLANを改訂・再承認する。

QAは、DEVが `HANDOVER.md` に固定した同じ `candidate_commit` と `candidate_tree` から、REVIEWのPASSを開始条件にせず独立に開始する。QA開始時にcandidateのcommit/tree不一致、未固定、または影響範囲不明があれば、全caseをPASSにせずMainへ報告する。

## 共通fixtureと観測

- 一時ディレクトリ内のfile-backed SQLite DBを使う。接続を明示的にcloseし、同じDB pathをreopenする。in-memory DBだけでrecoveryを代替しない。
- TASK-0027/0028で作成済みのTask、有効なactive owner、workspace参照をfixtureに用いる。owner割当/解放、lifecycle遷移、owner移管を本Taskの更新操作で再実装・再定義しない。
- contract/progressの初期version、各payload、schema ID/revision/digest、event sequence、current read model、historyの全recordを操作前後に取得し比較できるようにする。
- 各更新は一つのexpected versionから開始し、永続化状態を新規reader/新規connectionでも観測する。失敗操作後はcurrent、history、eventの全てを再読する。
- DEV証跡にはcase ID、candidate commit/tree、正確なcommand/test、fixture/SQLite pathの種別、cache条件、exit、生成物またはtest output digest、未実施理由を記録する。QAはその完全性、候補への束縛、negative caseの失敗検出能力、テスト削除/弱体化の不存在を独立確認する。

## ケース

| ID | 受け入れ条件・境界 | 操作と期待結果 | qa_execution_mode / 理由 | fail-closed |
|---|---|---|---|---|
| Q29-01 | contract expected-versionの正常CAS、immutable snapshot、current read model、`ContractChanged` | 現current versionをexpectedとしてcontractを更新する。new versionはちょうど旧+1で、旧snapshotは同一内容のまま残り、新snapshot/current/eventが一つずつ同一transactionで可視になる。eventは正しい順序で対応schema refを持つ。 | `focused-rerun` — temporary file SQLiteと固定fixtureで、atomic persistenceの受け入れ真実をhermetic/deterministic/boundedに再現できる。 | 一時SQLite integration testでない、current/history/eventのいずれか未観測、candidate証跡不一致、又はtransaction境界不明ならPASSにしない。 |
| Q29-02 | progress expected-versionの正常CAS、append-only history、current progress、対応event | 現current versionをexpectedとしてprogressを更新する。new versionは旧+1、旧historyは不変、新history/current/eventが一つずつ同一transactionで可視になり、schema ID/revision/digestを保持する。 | `focused-rerun` — Q29-01と同じbounded SQLite fixtureで完全再現できる。 | append-only性、version+1、event、schema refのいずれかを独立に検証できないならPASSにしない。 |
| Q29-03 | stale/future/skipped、同一version内容差替え、busyの拒否 | stale expected、future expected、currentより飛んだnew version、及び同一versionで異なるpayloadを試す。SQLite busy/lockも成功又は自動retryに変換せずtyped conflict/storage errorとして返す。各失敗後のcurrent/history/event sequenceは操作前と完全一致する。silent overwrite、generic patch APIは存在しない。 | `focused-rerun` — 境界入力とreadbackはdeterministicかつbounded。 | error型/意味が未確定、失敗後の三読モデルを比較しない、自動retry/上書きを許す、又は失敗event sequence gapを検査しないならPASSにしない。 |
| Q29-04 | 同expected versionの競合とlost update防止 | 同一current versionをexpectedにした二更新を並行開始する。正確に一方だけ成功し、敗者はtyped conflictとなる。最終current、history増分、event増分は成功一回分だけでpartial writeがない。contractとprogressをそれぞれ対象にする。 | `focused-rerun` — thread/task barrier等のbounded deterministic fixtureで競合受け入れ真実を再現できる場合のみ許可する。 | fixtureが競合を実際に同一expected versionへ到達させる保証を持たない、成功数が正確に一つでない、busy/lockをsuccess又はsilent retryに変換する、又は部分状態を再読しないならPASSにしない。再現のdeterminismが証明できなければ `live-e2e` に昇格しblockedとする。 |
| Q29-05 | schema referenceの不変保存と全履歴不変 | contract/progress/eventの初期recordと更新recordについてschema ID/revision/digestを検査し、保存portで必要な非空/形式境界を拒否することを確認する。更新後に旧contract snapshot、旧progress history、既存eventのpayload/version/schema refが削除・上書きされていないことを全件比較する。runtime JSON Schema意味検証やschema resolution libraryを導入していないことも差分で監査する。 | `focused-rerun` — file SQLite fixtureの全record比較はbounded/deterministicであり、schema参照の永続化を直接確認できる。 | record種別の一つでもschema ref又は入力境界未検査、旧recordの同値比較欠落、Schema変更又はvalidator/dependency追加が未再計画ならPASSにしない。 |
| Q29-06 | 故意SQL失敗時のatomic rollback | transaction中にhistory append、current pointer更新、event appendのいずれか後段でSQL failureを注入する。操作はstorage errorとして返り、current/history/eventが全て操作前と完全一致することを検査する。contract/progressの各更新経路を対象にする。 | `focused-rerun` — deterministic fault injectionとfile SQLite readbackでrollback真実をboundedに再現できる。 | failureが実SQL pathへ注入されない、rollback後の三状態全比較がない、失敗をsuccess/自動retryへ変える、又はcleanupが安全でないならPASSにしない。 |
| Q29-07 | close/reopen durability、current read model、integrity fail-fast | Q29-01/02の更新とhistory/eventを保存後、DBをcloseし同一pathをreopenする。Task current state、active owner、workspace参照、current contract/progress、contract/progress全history、event sequenceとschema refsがclose前のread modelと完全一致することを比較する。current/historyのversion/digest/schema refに欠落・矛盾・重複・非単調を故意に作った時はtyped storage/corruption errorでfail-fastし、部分read modelを返さない。event replayのみを正本とせず、persisted current read modelを直接読む。 | `focused-rerun` — file-backed temporary SQLite、明示close/reopen、固定データ、bounded corruption fixtureによりdurability/recoveryを完全再現できる。 | in-memoryのみ、同一connection再利用、owner/workspace/current/history/eventの比較欠落、integrity異常でpartial readを返す、event replayだけでの再構築、又はcleanup不能ならPASSにしない。 |
| Q29-08 | TASK-0028依存の非回帰とscope外不変 | TASK-0028が固定したowner排他/lifecycle API、migration version、typed conflict、event sequence契約に対し、更新の前後でactive owner、workspace参照、Task lifecycleが不当に変化しないことを確認する。SQLite migration v3が0027/0028の既存dataを保存することも確認する。差分を監査し、owner assignment/release/transfer、Agent Run、Task tree、Workspace/Resource cleanup、transport/CLI、Inbox/Outbox、Plane間原子性、event-sourced再構築、backup/replication/PITR、generic patch、自動retry、runtime schema validation/Schema変更/未承認依存追加がscope外のままであることを確認する。 | `evidence-review` — scope外不変はcandidate-bound diff、TASK-0028 merge証跡、対象test証跡の独立監査で判断する。 | TASK-0028 merge/tree一致証跡欠落、migrationの既存data保存証跡欠落、差分範囲不明、scope外変更、テスト弱体化、又はcandidate-bound evidence欠落ならPASSにしない。高リスク実装部分をDEV自己証跡だけでPASSにしない。 |
| Q29-09 | 候補全体の回帰・証跡完全性 | candidateで `cd core && go test -count=1 ./internal/control`、`go test -count=20 ./internal/control`、`go test -race ./internal/control`、`go test ./...`、`go vet ./...`、`make check`、`make task-check TASK=TASK-0029`、`git diff --check` を実行する。QAは全caseのDEV証跡を再突合し、negative testが期待しない実装を検出できること、受け入れ条件・テスト・schema/config/dependencyが弱体化されていないことを確認する。 | `focused-rerun` — 全コマンドがcandidate上でhermetic/deterministic/boundedに完了することを確認した上でQAが独立実行する。そうでなければ実行不能理由を記録しPASSにしない。 | exit非0、candidate/tree/digest不一致、必要なcase evidence欠落、checkの環境依存又はunbounded性、テスト削除/弱体化、影響不明はPASSにしない。 |

## 判定、blocked、FAIL分類

- 全caseは同じcandidate commit/treeへ束縛され、Q29-01〜09がPASSして初めてQA PASS候補となる。TASK-0028の依存条件未充足、またはmerge後APIとの前提差は `requirement_gap` としてPLAN/QA_PLANへ差し戻す。
- SQLite busy/lock、並行実行の不安定さ、fault injection不能、fixture破損、またはCI/host問題は、再現証跡を残して `environment_issue` と分類する。Task/PLANに合意済みの期待結果から実装が逸脱した時だけ `implementation_defect`、既存挙動の破壊は `regression`、本書の誤った期待/手順は `qa_plan_defect` とする。最終分類と差し戻し先はMainが決定する。
- `live-e2e` が必要と判明したcaseは、承認済み隔離環境、操作、観測、rollback/cleanupが準備されるまでblockedである。別モードのPASSで代替しない。
- REVIEWによるcandidate変更後、Mainだけが再実行範囲を決める。並行性、persistence、error/rollback、Schema参照、受け入れ条件、QA_PLAN、テスト、設定、依存の変更は `qa_carry_forward` を禁止し、影響caseを再実行する。merge後はMainが `merge_tree == candidate_tree` を確認し、不一致なら影響を再評価する。

## 必須QA結果証跡

`QA_RESULT.md` には、各caseのモード、理由、実コマンド、fixture、cache、exit、candidate commit/tree、artifact digest、観測結果、未実施/blocked理由、FAIL分類を記録する。Mainはcandidate treeとmerge treeの比較、環境依存caseのpost-merge扱い、candidate変更時のrerun又はcarry-forward不許可理由を記録する。

## 改訂履歴

- 2026-07-20: TASK-first初版。`TASK.md`だけを根拠に作成。
- 2026-07-20: PLAN完成後突合。TASK-firstの受け入れ条件・scope・モードに矛盾なし。PLANで明文化されたSQLite v3既存data保存、workspace参照、schema ref入力形式境界、busyのfail-closed、event sequence gapなし、recovery integrity corruption fail-fast、正確なverification commandを追加した。実装/結果証跡は参照していない。
