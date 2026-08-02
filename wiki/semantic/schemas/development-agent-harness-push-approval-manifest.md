---
kind: schema
title: Development Agent Harness Push Approval Manifest
---

# Development Agent Harness Push Approval Manifest

## 問い

後続の承認境界が、構造化された一件のpush proposalを曖昧なJSON表現や参照更新の取り違えなしに同じ内容として束縛するにはどうするか。

## 主体とV1の値境界

`internal/approvalmanifest`は、callerが提示した`Proposal`を検査してimmutableな`Manifest`へ変換するpure value boundaryである。Proposalは`request_id`、`agent_id`、`workspace_id`、`repository`、`remote`、順序付き`ref_updates`、`policy_version`、`revocation_epoch`、`created_at`、`expires_at`を持つ。packageはrequest ID、identity、policy、時刻を生成せず、それらの信頼性、現在時刻、TTL又はpolicyの妥当性も判断しない。

V1はlowercase canonicalな`owner/repo`と、それから導くexact `https://github.com/<owner>/<repo>.git`だけを受理する。時刻はUTCのwhole-secondで`created_at < expires_at`、参照更新は1--32件で入力順を保持し、重複を拒否する。各更新は`ref`、`expected_old_sha`、`new_sha`、`force`、`delete`から成り、branch headの安全な部分集合、40桁lowercase SHA-1表記、create・通常update・明示force・deleteの整合した組合せだけを表せる。順序をsetへ正規化しないため、同じ更新でも配列順が違えば別のproposalである。

## Canonical encoding とdigest

payloadのfield順は、`format_version`、`request_id`、`agent_id`、`workspace_id`、`repository`、`remote`、`ref_updates`、`policy_version`、`revocation_epoch`、`created_at`、`expires_at`で固定する。各`ref_updates`要素の順も`ref`、`expected_old_sha`、`new_sha`、`force`、`delete`で固定する。public manifestはこのpayloadの直後に、derivedな`request_digest`を最後のfieldとして置くcompact JSON一文書である。

`request_digest`自身を除くcanonical payload bytesへ、NUL終端のV1 domain prefix `dev-agent-harness/push-approval-manifest/v1\\x00`を先行させてSHA-256を計算する。公開値は実計算した`sha256:<64 lowercase hex>`だけであり、caller supplied digestを受け取らない。したがってidentity、repository/remote、policy/epoch、時刻、各参照更新のold/new SHA、force/delete、およびref配列順はすべてdigestの対象である。

## Strict parse と不変性

`Parse`は32 KiB以内のpackage自身のcanonical encodingだけを受理する。single-documentかつduplicate-awareな走査、unknown/欠落fieldの拒否、typed decode、Buildと共通のsemantic validation、constant-time digest照合、再encode bytesとの完全一致をすべて通過条件にする。空白、field順、escape、number、time、digest表記、trailing dataを「読めるJSON」として正規化して通す経路はない。

BuildとParseはcallerのslice/raw bytesを保持せず、`RefUpdates()`と`Encoding()`はfresh copyを返す。公開errorは固定class、field、任意のref update indexだけを示し、identity、repository、ref、object ID、digest、raw/canonical bytes又は下位parser errorを診断へ含めない。

## 認可との境界

validなManifestは承認候補となるcontentを一意に表すだけであり、**approved、granted、pushableを意味しない**。[Approval Request Store](development-agent-harness-approval-request-store.md)が現在時刻、TTL、request ID一意性、policyの信頼性を所有し、[Passkey Challenge Lifecycle](development-agent-harness-passkey-challenge-lifecycle.md)がrequest digestとdecisionへ一回限りのverifier入力を束縛する。one-shot push grantはさらに後続の別境界である。Manifest自身はGit receive-pack/pkt-lineを解析せず、remoteのold SHAを観測せず、forceをwireから推定せず、credential、実push、audit、network又は永続stateを持たない。

## 関連

- [TASK-0070 HANDOVER](../../../tasks/TASK-0070-push-approval-manifest/HANDOVER.md)
- [Approval Request Store](development-agent-harness-approval-request-store.md)
- [Development Agent Harness Passkey Challenge Lifecycle](development-agent-harness-passkey-challenge-lifecycle.md)
- [Development Agent Harness Egress Transaction](development-agent-harness-egress-transaction.md)
