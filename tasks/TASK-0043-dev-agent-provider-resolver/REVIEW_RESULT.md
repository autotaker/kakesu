---
task_id: "TASK-0043"
status: complete
reviewer_agent: "reviewer-agent-terra-medium"
decision: pass
reviewed_at: "2026-08-01T07:59:37Z"
---

# TASK-0043 REVIEW RESULT

## 独立レビュー

- Reviewer（Terra/medium）として、HANDOVERで固定されたcandidateのdiff/sourceと候補worktreeのtest、TASK/PLAN/QA_PLANを独立に照合した。candidate識別子はHANDOVERだけで管理する。
- `git diff --check base...candidate`は無出力だった。
- 差分は許可された3ファイルだけで、追加678・削除0（上限1,200以下）。外部module、config、CLI、transport実装、既存packageの変更はない。

## 監査したDEV証跡

| コマンド/テスト | 結果 | 備考 |
|---|---|---|
| root `make check` | `PASS`（証跡監査） | HANDOVERのcandidate-bound結果はcandidate launcherで一回成功と記録している。レビューでは包括`make check`を再実行していない。 |
| harness `make check` / `make distcheck` / `go test -race ./internal/providercredentials` | `PASS`（証跡監査） | HANDOVER記載を候補source、scope、QA_PLANの実施分離と照合した。QAのfocused rerunは再実行していない。 |
| 実GitHub、TLS/CA/DNS/proxy | `blocked` | QA-004/QA-005のlive-e2e。fake RoundTripperのPASSで代替していない。 |

## 受け入れ条件の確認

| 条件 | 結果 | 根拠 |
|---|---|---|
| AC-1 | PASS | `New`はbundle/transport/1ms〜30秒以外を固定`invalid-rules`で拒否し、nil/zero resolverは固定resolve errorでfail-closedする。`*Resolver`のinterface適合もcompile-timeで固定されている。 |
| AC-2 | PASS | `Resolve("openai", "")`は検証済みkeyだけを返し、GitHub JWT/transport前に分岐する。repository付きOpenAI・未知provider・非canonical GitHub repositoryはtransport前に拒否される。 |
| AC-3 | PASS | canonical grammarはTASK-0039の`egresspolicy`と同値。固定HTTPS host/path、installation IDのdecimal path、repository名だけの単一JSON配列、Bearer JWT、4必須headers、request deadline、直接一回の`RoundTrip`をsource/testで確認した。client/default transport、redirect、retryはない。 |
| AC-4 | PASS | non-nil bodyを全経路defer closeし、201/media type/128 KiB上限、単一top-level object、全top-level field重複拒否、trailing値拒否、visible ASCII token、RFC3339 expiryを検証する。expiry評価直前にprivate clockを一回UTC化し、nowより後かつ65分以内に限定する。close/read/transport/parserの詳細は返さない。 |
| AC-5 | PASS | resolverの保持状態はbundle、注入transport、timeout、clockだけで、cache/refresh/singleflight/retry/default transport/I/O/log/persistenceは追加されていない。`Resolver.Format`と公開errorは固定値で、READMEもtrusted boundaryとlive-e2eの対象外を一致して記載する。 |
| AC-6 | PASS | hermetic runtime credential fixture/fake RoundTripper testがOpenAI no-network、GitHub request束縛、timeout/call count、response異常、body close、token/expiry境界、固定error/non-leakを検出する。transaction integrationはpolicy拒否時のtransport/Forwarder未到達とvalid GitHub/OpenAI時のtrusted Forwarderへのみcredentialが渡る順序を確認する。 |

## 指摘

- なし

## 結論

PASS。実GitHubのJWT受理・repository scope・token形式、および実TLS/CA/DNS/proxy transportは本レビューのPASS対象外で、QA_PLANどおりblocked live-e2eのままとする。
