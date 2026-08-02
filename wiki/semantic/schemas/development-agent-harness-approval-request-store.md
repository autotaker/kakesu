---
kind: schema
title: Development Agent Harness Approval Request Store
---

# Development Agent Harness Approval Request Store

## 問い

canonicalなpush approval manifestを、再起動後も一意に参照できるrequest状態として保持し、期限・永続化失敗・複数writerによる認可判断の取り違えをどうfail closedにするか。

## 所有と入力境界

`internal/approvalstate`は、broker ownerがあらかじめ用意したowner-only directoryを一processだけで開くdurable storeである。`Open`は既存のcurrent-owner `0700` root、固定state/lock/temp名、regular node、non-blocking exclusive process lockを受理条件とし、root、node又はsymlinkの検査後も文字列pathへ戻らない。検証済みroot descriptor（`os.Root`）にstate、temp、lockの操作を束縛するため、rootのrename又はsymlink置換で別directoryを更新しない。

`Create`は[Push Approval Manifest](development-agent-harness-push-approval-manifest.md)を再parseし、canonical bytes、policy version、revocation epoch、trusted clock、正のbounded TTL、capacity、未使用request IDを検証してから`pending`を作る。同じIDは同じbytesでもconflictである。recordはcanonical manifestとderived digestをcopyして保持し、公開getterはcopyを返す。callerはstate、時刻、path又は任意digestを選べず、公開errorもroot、ID、actor、manifest/snapshot bytes、digest、下位OS errorを含めない。

## 状態と時間

状態は`pending`、`approved`、`denied`、`cancelled`、`expired`、`stale`である。request IDとconstant-time digest一致を前提に、`pending`から`approved`、`denied`、`cancelled`、`expired`へ、`approved`から`stale`又は`expired`へだけ遷移できる。approval/denialは上位が検証済みとして渡したactor IDを記録するが、storeはPasskey又はactorの本人性を検証しない。

mutation、`Get`、expiry処理はtrusted clockをmutex下で比較し、期限到達をdecisionより先に`expired`として永続化する。clock rollback、policy/epoch又はdigest mismatch、terminal state再遷移、approved以外からのstaleは拒否する。この順序により、期限切れrecordをapproval可能として返さない。

`approved`はverified decisionが保存された状態であり、grant、push authorization、one-shot consumption、実push成功を意味しない。Passkey/WebAuthn verifierの前段となるchallengeのreservationと消費は[Passkey Challenge Lifecycle](development-agent-harness-passkey-challenge-lifecycle.md)が所有し、両者を`Consume`後のexact `Approve`/`Deny`へ接続する順序は[Verified Decision Coordinator](development-agent-harness-verified-decision-coordinator.md)が所有する。それらの上位認可は後続の別境界である。

## Snapshotの確定境界と復旧

V1 snapshotはgeneration、最後に観測したtrusted time、request-ID順のbounded recordsをcanonical JSON一文書として保存する。open時はmanifest/digest/state/time/actor shape、record順序と一意性、snapshot shapeと上限を全て再検証し、partial、trailing、unknown、duplicate、noncanonical、oversize又は残存tempを推測して回復しない。

mutationはcopy-on-write snapshotをtemp regular fileへ全write、file sync、close、same-directory atomic replace、directory syncの順で進める。replace前の失敗ではold memoryとdiskを維持する。replace失敗又はreplace後のdirectory sync失敗はnamespace結果を推測できないためstoreをpoisonし、Close以外の操作を拒否する。利用者はClose、Open、上位reconciliationを明示的に行い、successの再試行やpartial memory commitで曖昧さを隠さない。

## 適用限界

このStoreは単一host・単一process writer、既存root directory、hermetic filesystem seamの範囲だけを扱う。rootの作成/chown、実`/var/lib`のUID/permission、network filesystem、backup/rotation、power-loss durability、systemd配置/restart/rollback、Passkey verification、verified-decision APIへ到達できるOS/process境界、grant発行・消費・reconciliation、Git receive-pack又は実pushは保証しない。これらの実環境性は別のlive E2Eで確認する。

## 関連

- [TASK-0071 HANDOVER](../../../tasks/TASK-0071-approval-request-store/HANDOVER.md)
- [Development Agent Harness Push Approval Manifest](development-agent-harness-push-approval-manifest.md)
- [Development Agent Harness Passkey Challenge Lifecycle](development-agent-harness-passkey-challenge-lifecycle.md)
- [Verified Decision Coordinator](development-agent-harness-verified-decision-coordinator.md)
