---
task_id: "TASK-0050"
status: pass
qa_agent: "qa-agent-terra-medium"
decision: pass
tested_at: "2026-08-01T12:20:43Z"
---

# TASK-0050 QA RESULT

## 結果

| ケース ID | モード | 結果 | 独立証跡 |
|---|---|---|---|
| QA-001 | evidence-review | pass | `New` は single-block PEM、self-signature、CA constraints、P-256 key match、clock validity と最低15分残存を固定 `invalid-rules` へ畳む。source/test は typed nil clock、PEM/block/key/validity/constraint の拒否と非漏えいを確認する。 |
| QA-002 | focused-rerun | pass | 指定 race test は exit 0（11.382s）。source/test は入力を変更後も Authority state が安定し、certificate-only public PEM の毎回独立copy、zero Authority と format/non-leak を確認する。 |
| QA-003 | focused-rerun | pass | 指定 race test は exit 0。literal allowlist により二hostだけを署名前に通し、test は空、case差、末尾dot、port、wildcard、IP、unknown、control を固定 `proxy-ca-denied` と空 certificate へ畳むことを確認する。 |
| QA-004 | focused-rerun | pass | 指定 race test は exit 0。test は host別の fresh P-256 key/nonzero 128-bit serial、single DNS SAN、ServerAuth/DigitalSignature、CA capabilityなし、backdate/lifetime/CA期限境界を検証する。 |
| QA-005 | focused-rerun | pass | 指定 race test は exit 0。`net.Pipe` TLS 1.2/HTTP/1.1 hostname verification は両hostで成功しwrong-hostは失敗する。concurrent issuance は raceなし、serial/key重複なし、cross-host SAN混線なしを確認する。 |
| QA-006 | evidence-review | pass | fixture test は PEM/block/key/CA validity 拒否、copy/non-leak、exact host gate、leaf extensions/chain/validity、TLS hostname verification、concurrent uniquenessを実行する。HANDOVERの candidate-bound evidence は root `make check`、harness check/distcheck、lint-docs、diff check がすべてpass、差分は許可3 path・590 additions/0 deletionsであることを示す。QAはこれら包括検査を再実行していない。 |
| QA-007 | live-e2e | blocked | 実CA file/rotation、OS trust store、実client、listener/CONNECT、VPS環境および安全なcleanupが未定義。hermetic結果で実配布/trustを代替していない。 |

## 発見事項

- なし。focused-rerunは approved QA_PLAN に指定された一回の `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/proxyca` のみを実行し、passした。
- QA-007 の blocked は hermetic scope の失敗ではなく、未定義の実環境依存を正しく未実施として分類する。

## 結論

`pass` — QA-001〜QA-006 は approved QA_PLAN と planning input packet に適合する。QA-007 は live-e2e のまま blocked であり、passへ置換していない。
