---
task_id: "TASK-0050"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T11:58:06Z"
revision: 1
implementation_reviewed_at: "2026-08-01T12:21:32Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# QA_PLAN: TASK-0050

## QA scope

TASK.mdの`Planning input packet`だけを期待値正本として、candidateの
`tools/dev-agent-harness/internal/proxyca/`と許可されるREADME差分を独立に確認する。
Authorityが既存brokerhttp、Exchange、Policy、credential/transportの意味を変更又は再実装して
いないことも確認対象にする。

実CA file/rotation、trust store/client、listener、CONNECT、VPSはlive E2E対象だが、安全な実環境と
cleanupが定義されていない。このためblockedとし、in-memory PEM又は`net.Pipe`の結果で実配布又は実trustを
主張しない。

## Cases

| Case ID | 対象AC | 確認内容 | Mode | Evidence |
|---|---|---|---|---|
| QA-001 | AC-1 | `New`が単一CA certificate/private-key PEM、非nil Clockだけを受理し、複数/末尾block、parse、自己署名、BasicConstraints/IsCA、CertSign、ECDSA P-256、不一致key、現在有効性、leaf lifetime以上残存期間、typed nilを固定non-leak errorで拒否することを確認する。nil/zero Authorityも固定errorであることを確認する。 | evidence-review | candidate source/test、HANDOVER、DEV check証跡 |
| QA-002 | AC-2 | Authorityがparse済みCA certificate/signer、copyした公開certificate PEM、Clockだけを保持し、caller PEMを変更・保持しないことを確認する。`PublicCertificatePEM`がcertificate blockだけの新しいcopyを毎回返し、private key/追加blockを出さず、error/Formatに秘密、subject/serial/host/parser detailを出さないことを確認する。 | focused-rerun | `tools/dev-agent-harness`で一回だけ実行するrace test |
| QA-003 | AC-3 | `Issue`がexact `api.github.com`/`api.openai.com`だけを署名前に受理し、空、port、case差、末尾dot、wildcard、IP、unknown、control/non-ASCIIを固定errorで拒否することを確認する。拒否時にserial/key/certificateを返さず、retry/cache/host補正しないことを確認する。 | focused-rerun | `tools/dev-agent-harness`で一回だけ実行するrace test |
| QA-004 | AC-4 | 各許可hostで新しいECDSA P-256 leaf keyとnonzero random 128-bit serialを作り、単一exact DNS SAN、空CN、ServerAuth、DigitalSignature、IsCA false/BasicConstraintsValid、5分以内backdate、15分以内かつCA期限以前のvalidityを発行することを確認する。IP/email/URI SAN、ClientAuth、CertSignがないことを確認する。 | focused-rerun | `tools/dev-agent-harness`で一回だけ実行するrace test |
| QA-005 | AC-5 | `tls.Certificate`がleaf→CA chain、parse済みLeaf、対応leaf keyを持ち、call間でcertificate/key/serial/bufferを共有しないことを確認する。公開CAによる両hostのTLS 1.2/HTTP1.1相当handshake/hostname verify成功と、wrong host/expired CA/未許可hostのfail-closed、並行Issueのrace/duplicate/cross-host SAN混線なしを確認する。 | focused-rerun | `tools/dev-agent-harness`で一回だけ実行するrace test |
| QA-006 | AC-6 | in-memory fixtureのrace testがPEM/block/key/CA validity拒否、input/public output copy、fixed non-leak、exact host拒否、leaf extension/validity/chain、TLS handshake/hostname verify、concurrent uniquenessを実際に失敗検出できることを確認する。source/test/HANDOVERからDEVがharness `make check`/`make distcheck`、README変更時Task worktree `make lint-docs`、candidate launcher root `make check`を実行済みであること、許可pathとbase...candidateの追加＋削除が1,000行以下であることを監査する。QAは包括checkを再実行しない。 | evidence-review | candidate source/test、HANDOVER、DEV command/result |
| QA-007 | 対象外 / AC-6の制限 | 実CA private key file/read/rotation、Agent trust store、実TLS client、listener、CONNECT、VPS環境での配布・trust・接続を確認する。 | live-e2e — blocked | 実環境、権限、実秘密とtrust storeの安全な取得・cleanup、listener/CONNECT/VPS経路が未定義。このblockedは他caseのPASSで代替しない。 |

## Execution rule

focused-rerunのQA-002、QA-003、QA-004、QA-005は同じ一回のrace test観測に束ねる。QAは
`tools/dev-agent-harness`をcwdとして、次だけを一回実行する。

```sh
GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/proxyca
```

それ以外はcandidateのsource/test、HANDOVER、DEV command/resultを独立監査する。root `make check`、harness
`make check`、harness `make distcheck`、lint、追加processはQAでは実行しない。

## Result criteria

各caseについてcandidate source/testとHANDOVERの事実をPlanning input packetに照らして記録し、focused-rerunはcommandと結果を記録する。失敗は実装不具合と決めつけず、QAガイドラインに従ってcandidate、環境、依存、要件又は証跡のいずれかへ分類する。QA-007は実施可能になるまでblockedのままとする。

## 実装後の再確認

- [x] candidateのsource/test、HANDOVER、DEV check証跡を独立に確認した。
- [x] 指定race testをcandidateで一回だけ実行した。
- [x] live E2E blockedをPASSに置換せず、期待結果又は範囲を変更していないことを確認した。
