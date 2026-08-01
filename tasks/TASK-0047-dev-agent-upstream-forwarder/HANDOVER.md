---
task_id: "TASK-0047"
status: complete
completed_at: "2026-08-01T10:41:49Z"
candidate_commit: "57e7411868094db977e304593846c2f8ad2cdec0"
---

# TASK-0047 HANDOVER

## 成果

- `egresstransaction.PreparedRequest`を同じPolicyで再評価し、scopeとBearerを再検証してから注入RoundTripperへ一度だけ送る`internal/upstreamforwarder`を追加した。
- 成功responseを上限付きで完全読取・検証・closeし、status、正規JSON media type、独立本文だけをrequest単位sinkへ一度渡す。
- GitHub GET/HEADは空content type・空bodyだけに狭め、Agent制御bodyを実認証情報付きで上流へ送らない。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate launcherのroot `make check` | PASS。最終working bytesを固定した案作成runで完走 |
| `make check && make distcheck`（`tools/dev-agent-harness`） | PASS。最終candidateでpackage全体、vet、build、配布tarball再展開を確認 |
| `go test -race ./internal/upstreamforwarder` | PASS。GitHub body拒否を含む最終codeのfocused race test |

## 主要な変更

- `Rules`、固定error、Response/ResponseSink、`egresstransaction.Forwarder`実装を新packageへ閉じた。
- policy/scope/Bearer/GitHub empty-body境界をtransport前に検査し、上流headerをAuthorization、Accept、User-Agent、OpenAI Content-Typeだけに限定した。
- 2xx、HEAD/204 empty、JSON media type、size、UTF-8/JSON、read/close/timeout、response+error、sink error、copy/race/non-leakをfake依存だけで検出するtestを追加した。
- READMEへ既存transaction/transportとの合成責務とlive境界を追記した。

## 検証結果

- candidate launcherのroot `make check`: PASS。README用語lintで停止した2回のpre-candidate runはcommitを作らず、表記修正後の最終bytesで全検査を再実行してcandidateを固定した。
- harness `make check` / `make distcheck`: 最終candidateでPASS。live testはconfigure既定どおりSKIPであり、実provider成功の証拠にはしていない。
- 差分は許可2 path内の3 files、697追加・0削除で1,000行以下。dependency、config、generated artifact、既存package変更はない。

## 判断・既知の制約

- Forwarderはprovider error本文と上流headerをAgent側へ返さず、2xxのJSON/empty successだけを縮退する。pagination/rate-limit等の必要headerは実クライアント統合で観測後に別Taskで最小追加する。
- 実GitHub/OpenAI、実認証情報、Internet DNS/TLS/system trust、Agent proxy/response writerは未実施のlive E2E境界である。
- Agent向けlistener、Transactionとのproduction wiring、Git Smart HTTP、streaming、redirect/retryは対象外のままである。
