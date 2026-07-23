---
task_id: "TASK-0027"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
revision: 2
implementation_reviewed_at: "2026-07-20"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0027 QA PLAN

## TASK-first baseline

この計画はPLAN、実装差分、既存testを参照せず、完成した`TASK.md`だけからDEV開始前に作成した。TASK-0027はGo CoreのSQLite依存、migration、永続Schema、transaction、rollbackという高リスク境界を変更する製品Taskである。QAの期待値は、temporary SQLite実DBで受け入れ真実を再現し、部分状態や再open時の隠れた副作用を機械的に検出することに置く。

同一schema versionの再openはmigrationとpragma初期化の再実行安全性に加え、初回作成で永続化したcreation read modelをclose/reopen後に同一内容で取得できることを対象とする。一方、owner lifecycle、contract/progress更新履歴、versioned recoveryはTASK-0029/0028の対象なので本TaskのPASS条件へ拡張しない。既存draft-v0 Schemaの意味検証又は変更、transport、CLI、Inbox/Outbox、Agent Runも検査対象外とする。

## mode選択と共通fail-closed規則

| `qa_execution_mode` | このTaskでの使用条件 | QA最低操作 | fail-closed条件 |
|---|---|---|---|
| `focused-rerun` | migration、pragma、永続Schema、transaction、rollbackをtemporary SQLite DBでhermetic・deterministic・boundedに完全再現できるcase | QA所有の一時directoryと新規DBを用い、`go test -count=1`で対象testとnegative mutationを独立再実行する | 実DBを通らないmockだけ、test cache依存、fixture残留、failure injectionが対象transactionを通らない、又は結果をtable/read modelから観測できない場合はPASS不可 |
| `evidence-review` | dependency/license、Core単独所有、変更scope、candidate-bound test証跡を差分と静的証跡で完全監査できるcase | candidate commit/tree、差分、command、環境、cache、exit、artifact digest、negative検出能力、test弱体化を照合する | 証跡欠落・digest不一致・candidate/tree不一致・scope逸脱・test削除/弱体化・影響不明ならPASS不可。挙動判断が必要なら`focused-rerun`へ昇格する |

本TaskのSQLite caseはpersistence/error/fail-closed境界なので`evidence-review`だけでPASSさせない。実OS権限、実配置、外部service、実restart/rollback、環境固有integrationはTASKに含まれないため、現時点で`live-e2e`は割り当てない。temporary fixtureでは受け入れ真実を再現できないと判明した場合は、期待値を弱めずMain承認の上でmodeを再検討する。

## 前提・環境・必要証跡

- Go 1.23互換のcandidate worktreeで実行し、SQLite DB、WAL/SHM、副生成物はQA所有の一時directoryへ限定して終了時にcleanupする。外部network、実利用DB、共有DBを使用しない。
- SQLite driverはpure-Go/CGO不要で、ライセンスとGo 1.23互換性をDEV開始前証跡及びcandidate差分で確認する。依存取得不能又は契約不明は`dependency/environment_issue`候補としてPLANへ戻し、別driverへ無承認変更しない。
- 各focused caseは`go test -count=1`又は同等のcache無効化で実行し、test名、fixture path種別、開始時のDB不存在、終了時cleanup、exit、要約、出力artifact digestを残す。raw DB digestはSQLite内部表現に依存するため単独のPASS根拠にせず、query結果、pragma、schema snapshot、table countを正本とする。
- DEV証跡はcase ID、`candidate_commit`、`candidate_tree`、command/test、環境/fixture、cache条件、exit、artifact digest、未実施理由を持つ。QAは同一candidateへ束縛し、negative caseが実際に失敗を検出できることとtest弱体化の有無を独立監査する。
- typed errorは呼出側が分類可能で、原因を保持することを確認する。error文字列の完全一致や未指定の公開API名は固定しない。

## 受け入れ条件との対応

| ケースID | 受け入れ条件 | `qa_execution_mode` / 理由 | 操作 | 期待結果 | 必要証跡 / fail-closed |
|---|---|---|---|---|---|
| QA-001 | 新規DB migrationと同一version再open、pragma設定、起動fail-closed | `focused-rerun` / migrationとconnection初期化はpersistence境界だがtemporary SQLiteで完全再現できる | 存在しないpathを開き、schema version、`sqlite_master`又は同等のschema snapshot、foreign key、WAL、busy timeoutをqueryする。一のroot Taskを作成してcreation read modelを記録し、close後に同じDBを同一versionで再openし、migrationが重複適用されず同じversion/schemaで、同一creation read modelを取得できることを確認する。未知version、故意のmigration SQL失敗、必要pragma設定失敗を各々注入する | 新規openでversion 1相当の単一migrationが原子的に完了する。同一version再openは成功しschemaを増殖・変更せず、初回作成直後と同一のcreation read modelを返す。foreign key有効、WAL有効、busy timeout設定済み。未知version/migration/pragma failureはtyped errorとなりworkload/createへ進まない | 各subcaseのDB初期状態、pragma値、version、正規化schema snapshot、close前/再open後のread model比較、typed error分類、failure subcaseのcreate未呼出/row count 0、cleanup、command/exit/digest。再openがmigrationを再適用、read modelが欠落/変化、未知versionを黙認、又はpragma failure後に利用可能ならFAIL |
| QA-002 | root Task作成の単一transaction正常系 | `focused-rerun` / 複数tableの原子性・順序・参照整合は高リスクだがtemporary DBで完全観測できる | migration済み空DBへ一つの入力を作成し、Task、owner Agent割当、workspace reference、contract snapshot、progress、eventをtransaction経由で作成する。全tableを直接queryし、公開read modelとも突合する | 一回の成功でTask 1、owner割当 1、workspace参照 1、contract v1 snapshot 1、progress v0 1、event 2がcommitされる。全ID/FKが同じTaskへjoinし、eventはsequence 1=`TaskCreated`、2=`OwnerAssigned`、重複又は欠落なし。read modelは入力と永続行に一致する | 入力fixture、table別件数・主なID/FK、event sequence/type、read model projection、transaction成功、command/exit/digest。tableごとの個別commit、event順序違反、孤立参照、追加rowがあればFAIL |
| QA-003 | contract v1 JSON snapshotとprogress v0の不変保存 | `focused-rerun` / snapshot metadataとJSON payloadの永続化はpersistence境界であり実DBqueryが必要 | 識別可能なcontract v1 JSONとschema ID/revision/digestを含む入力を作成し、DB rowとread modelから再取得する。別Taskのpayloadと混線しないnegative fixtureも用いる | schema ID、revision、digest、contract v1 payloadが同じsnapshotとして保存され、入力bytes又は明示された受理時canonical representationと一致する。progressは初期version/state `v0`で一件だけ作成され、contract/progress更新履歴は生成しない | 入力payload digest、保存payload digest、schema metadata、progress row、Task join、read model比較、command/exit/digest。metadata/payload不一致、暗黙のSchema意味変更、複数progress、TASK-0029の履歴先取りならFAIL |
| QA-004 | transaction途中の故意SQL failureで全table rollback | `focused-rerun` / error injection、rollback、再試行はfail-closedの高リスク境界だがtemporary DBでboundedに再現できる | 作成transactionの中間（少なくともTask作成後かつcommit前）へ故意のSQL failureを注入する。failure前後に全対象tableの件数と対象IDをqueryし、error後に同じ入力をfailureなしで再試行する | 初回はtyped errorで終了し、Task、owner、workspace、contract、progress、eventの全tableに対象rowを一件も残さない。transaction/connectionは安全に終了し、同じ入力の再試行が一回だけ正常commitしてQA-002の完全な状態になる | injection位置と実SQL error、rollback後の全table count/対象ID不存在、WALを含む再open後も不存在、再試行結果、command/exit/digest。部分row、sequence消費の残骸、lock残留、再試行失敗/重複ならFAIL |
| QA-005 | Core単独writer、依存・scope・後続Task境界 | `evidence-review` / ownershipとdependencyはcandidate差分で静的に完全監査でき、挙動caseはQA-001〜004で再実行する | candidate差分、`core/go.mod`、`core/go.sum`、store API、package import、既存draft-v0 Schemaを監査し、pure-Go/CGO不要、Go 1.23互換、ライセンス証跡、Control package内のwriter限定を確認する。TASK-0028/0029と対象外機能の追加を検索する | 変更はTASK/PLANが固定した`core/go.mod`、生成checksum lockの`core/go.sum`、`core/internal/control/store.go`、`core/internal/control/store_test.go`だけに限定される。外部Plane用write API、transport/CLI/Inbox/Outbox/Agent Run、owner lifecycle、contract/progress更新履歴、Schema変更を追加しない | changed-path一覧、dependency metadata/license出典digest、import/API検索、Schema差分0、candidate/tree、command/exit/digest。CGO必須、Go非互換、不明license、外部writer、未承認path、scope外変更又は影響不明ならPASS不可 |
| QA-006 | candidate-bound統合証跡と回帰 | `evidence-review` / focused rerun結果、差分、広域checkを同一candidateへ束縛して監査できる | QA-001〜004のfresh結果をcandidate-bound DEV証跡と突合し、`go test -count=1 ./internal/control/...`、`make check`、`git diff --check`、変更scopeを監査する。test差分から各negative mutationの失敗検出能力を確認する | 全caseが同一candidate commit/treeを対象にし、cache無効化、fixture、exit、artifact digestが揃う。focused testsと広域checkがPASSし、test削除/弱体化や既存wire/schema契約の変更がない | candidate commit/tree、全command/exit/time/digest、差分digest、未実施理由、negative assertion対応表。証跡欠落、不一致、testがmockだけ、scope逸脱、期待値不明ならPASS不可 |

## forward / negative scenarios

| ID | 種別 | 条件 | 期待結果 |
|---|---|---|---|
| FT-001 | 正常 | 存在しないtemporary DBをopen | migration、schema version、必要pragmaが一度だけ成立する |
| FT-002 | 回帰 | 作成済みTaskを持つ同じDBをcloseして同一versionで再open | migration重複なしで成功し、初回作成直後と同一のcreation read modelを返す。更新履歴とversioned recoveryは先取りしない |
| FT-003 | negative | DBへ未知schema versionを設定 | typed errorでopenを拒否し、workloadを進めない |
| FT-004 | negative | migration SQL又は必要pragma設定を故意に失敗 | 部分schemaを利用可能にせずtyped errorを返す |
| FT-005 | 正常 | root Taskを一回作成 | 6種類の状態rowとsequence 1/2の2 eventが単一transactionでcommitされread modelと一致する |
| FT-006 | 境界 | schema metadata付きcontract v1 JSONとprogress v0 | payload/ID/revision/digestと初期progressが正しいTaskへ一件ずつ保存される |
| FT-007 | negative | create transaction中間のSQL failure | 全対象tableが作成前状態へrollbackし、同一入力の安全な再試行が成功する |
| FT-008 | negative | failure後にDBを再openして全tableをquery | WAL/connection内だけに隠れた部分状態がなく、対象IDが存在しない |
| FT-009 | scope | 外部Plane writer又はTASK-0028/0029機能を候補へ追加 | scope逸脱としてPASS不可 |

## exact checks

実装後、実装されたtest名へ操作を具体化し、少なくとも同一candidateで次を実行する。`-count=1`を外さず、対象testが一つも選択されない場合はFAILとする。

```sh
cd core && go test -count=1 ./internal/control/...
make check
git diff --check <approved-base>...<candidate>
```

QA-001〜004を一括testで実行する場合も、migration/reopen/unknown version/migration failure/pragma failure、正常atomic create、snapshot/progress、SQL failure rollback、reopen確認、安全な再試行の各subtest名とresultを個別に証跡化する。SQLite DB/WAL/SHMが一時directory外へ残らないことを実行後に確認する。

## 実施不能・merge後

- pure-Go driver取得又はlicense/Go互換性を確認できない場合はdependency/`environment_issue`候補としてblockedにし、別driverやCGOを無承認採用しない。
- temporary SQLite実DBを使うfailure injectionを用意できない場合、mock結果やDEV自己申告で代替PASSせず、`qa_plan_defect`又は`requirement_gap`候補をMainへ返す。
- command未実施、fixture作成失敗、cleanup不能、digest不一致、candidate/tree不一致は影響caseと再現条件を記録し、PASSへ置換しない。
- Mainはapproved candidate treeと`merge_tree`を比較する。同一かつ環境依存caseなしなら全面再QAを省略できる。本Taskはtemporary DBで閉じる前提だが、実配置・実権限・外部作用が追加された場合は該当caseを`live-e2e`へ昇格してマージ後にも確認する。

## 実装後の再確認

- [ ] 同一candidate commit/treeへDEV証跡、REVIEW、QAを束縛した。
- [ ] QA-001〜004をtemporary SQLite実DB、cache無効化、bounded fixtureで独立再実行した。
- [ ] negative injectionが対象migration/transactionを実際に失敗させ、false-passしないことを確認した。
- [ ] PLANとの突合後もTASK-first期待値と範囲を変更していない。
- [ ] 操作具体化以外の期待値又は範囲変更が必要なら、改訂理由とMain承認を得るまでPASSにしない。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-20 | QA Agent | TASK-first初版。temporary SQLiteのmigration/reopen、atomic create、snapshot、SQL failure rollback/retry、scope/evidence caseを固定。 | `approved by main-agent-sol-high` |
| 2 | 2026-07-20 | Main Agent | `go.sum`を未承認追加pathと誤記したqa_plan_defectを修正。承認済みPLANどおり生成checksum lockとしてQA-005の監査対象へ明記。製品期待値・candidateは不変。 | `approved by main-agent-sol-high` |
