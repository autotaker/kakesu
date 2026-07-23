---
kind: schema
title: Control Durable Store
---

# Control Durable Store

## 責務と境界

Control-owned Storeは`control.db`の唯一のwriterであり、root Taskのcurrent record、active owner assignment、workspace参照、contract snapshot、progress、eventを永続化する。接続時にはWAL、foreign keys、busy timeoutを設定する。

この境界はroot Taskの作成、lifecycle遷移、ownerの単一割当・解放、および再open可能なread modelを扱う。contract/progressのversioned更新履歴、Schemaの意味検証、transport、Plane間原子性は別の責務である。

## Versioned contract、progress、およびrecovery integrity

contractとprogressの更新は、呼出元が提示したexpected versionをcurrent versionと同じtransactionで照合するcompare-and-setである。version不一致はcurrent、history、eventのいずれも変更せずに拒否する。成功時だけ新current、immutable history、schema参照を含むeventを原子的に記録するため、同じexpected versionを使う並行更新は一方だけが成功する。

永続read modelの復元では、current/history/event間のversion、schema参照、payload、およびTask/Agent watermarkとfuture Task sequenceを検証する。不整合を検出した場合は部分的なread modelを返さず、typed corruption errorでfail closedにする。close/reopen後も同じ完全性条件を適用し、migrationまたはSQL failureはtransactionをrollbackしてpartial stateやversion gapを残さない。

## Schema versionの受理条件

version tableは、認識済みのcurrent versionが**ちょうど一行**ある場合だけ既存Schemaとして受理する。未知version、重複行、異なるversionの混在は、値の一つが期待値でもfail closedにする。新規DBのmigrationは冪等に初期versionを作るが、既存DBの曖昧な状態を推測して修復しない。

この検査は「最新versionが含まれる」ことではなく、Storeが前提にできるSchema状態が一意であることを保証する。

## 原子的作成と返却値

作成は一つのtransactionに閉じる。成功時にはTask、owner、workspace、contract v1 snapshot、progress v0、`TaskCreated` sequence 1、`OwnerAssigned` sequence 2がそろい、失敗時にはそのいずれも残さない。storage conflictやmigration失敗はsilent overwriteや自動retryではなく、typed errorとして返す。

commit後に同じcancel可能contextでDB readをすると、commit済みなのに呼出元へerrorを返す場合がある。そのため、成功時のread modelはtransaction内で組み立て、commit成功後にそれを返す。commit後のreadを成功条件にしない。

## Active ownerとlifecycle

Agentごとに非終端Taskは一件だけとする。この排他はprocess-local mutexではなく、active assignmentだけを対象にするSQLite partial unique indexを正本として保証する。並行する二Store/二接続の割当では、DB制約によって一方だけ成功し、敗者はTask、owner、eventのpartial writeを残さない。

状態遷移はexpected current stateを条件にしたcompare-and-set transactionで受理する。許可辺は`ready → running`、`running → waiting | suspended | reviewing_completion`、`waiting | suspended → running`、`reviewing_completion → running | completed`、および任意の非終端状態からの`cancelled`だけである。許可されない辺、終端状態からの再遷移、expected state不一致はcurrent state、owner、event sequenceを変えずに拒否する。

`waiting`、`suspended`、`reviewing_completion`ではownerを保持する。`completed`または`cancelled`への確定は、current state更新、terminal event、`released_at`によるowner解放を同じtransactionでcommitする。そのため、terminal transactionの成功後だけ同じAgentを別の非終端Taskへ再割当できる。event payloadは検証して保存し、state、event、releaseを別transactionへ分けない。

## Migration failureの扱い

v1からv2へのmigrationで、既存の重複active ownerのためpartial unique indexを作れない場合は、DDLとschema version更新をまとめてrollbackする。不整合な既存データを推測して修復したり、unique制約を省略して続行したりしない。

## 検証上の注意

SQLiteは単一writerなので、内部insert後のbarrierで競合を作ると相互待ちになる。競合割当は二Storeを同時開始して検証し、unique indexを唯一のarbitratorとする。busy/lockedやconstraint conflictを自動retryで隠さずtyped conflictとして返し、再試行の判断は上位に残す。

## 適用限界

この設計は単一SQLite Store内の原子性だけを保証する。owner移管、複数owner、Agent Run同時実行制御、Task tree cancellation cascade、外部サービス、他Plane、配備手順を含む操作のatomicityを意味しない。`SQLITE_BUSY`の専用fixtureは未整備であり、busyのtyped分類はcode reviewと反復二writer raceで補完している。

## 関連

- [TASK-0027 handover](../../../tasks/TASK-0027-control-durable-foundation/HANDOVER.md)
- [TASK-0028 handover](../../../tasks/TASK-0028-control-ownership-lifecycle/HANDOVER.md)
- [TASK-0029 handover](../../../tasks/TASK-0029-control-versioned-recovery/HANDOVER.md)
