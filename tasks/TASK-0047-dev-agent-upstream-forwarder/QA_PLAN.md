---
task_id: "TASK-0047"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T10:17:51Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0047 QA PLAN

## 方針

TASK.md の `Planning input packet` だけを要件正本として、DEV開始前に独立作成する。candidateの同一案をDEVおよびREVIEWから独立に評価する。fake `http.RoundTripper`、response body、ResponseSinkだけで完結する決定的なrace testを一回だけ再実行する。実provider、実credentials、実DNS/TLS、Agent proxyの主張はlive E2Eのblocked境界として残し、unitのPASSで代替しない。QAはroot/harness包括 `make check` を再実行しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1〜AC-6 | candidateで `go test -race ./internal/upstreamforwarder` を一回実行する。fake RoundTripper/body/sinkによるhermetic fixtureが、Rules（Policy、transport、sink、1ms〜30秒 timeout、1 byte〜1 MiB上限）とnil/zero receiverの固定error、Policyによるscope再評価・完全一致、認証形式、method/URL/content type/bodyのtransport前拒否を検出することを確認する。許可時には独立body、timeout context、RoundTripper一回だけ、redirect/retry/default transport/environment proxyなし、実Authorization・固定Accept/User-Agent・OpenAI時だけのContent-Typeというheader allowlistを確認する。さらに2xx、HEAD/204のempty受理とnonempty拒否、その他の空body、JSON media type、上限、UTF-8、JSON、status、read/close、timeout/cancel、response+error、sink異常、body close、transport/sink単回呼出、copy ownership/non-leakを検出することを確認する。 | `focused-rerun` / package内だけで受け入れ真実を再現し、外部network・実credentialなしで決定的かつ上限付きに実行できるrace testである。 |
| QA-002 | AC-1〜AC-5 | candidateのproduction sourceとtestを独立監査する。再評価又はscope不一致、invalid Authorization、入力のmethod/URL/content type/bodyの異常がcredential送信前かつtransport/sink 0回で拒否されることを確認する。caller入力の変更・保持、Agent由来header、proxy/default transport、redirect/retry、provider error本文・上流header・URL/scope/Authorization/underlying errorの漏洩がないこと、responseを完全検証・closeしてから一回だけsinkに独立copyを渡すことを確認する。 | `evidence-review` / QA-001の失敗検出能力、negative case、安全境界の弱体化の有無をcandidate source/testから独立監査する。 |
| QA-003 | AC-6 および完了経路 | HANDOVERをcandidate識別の唯一の正本として、candidateのsource/testとcandidate launcher証跡を監査する。candidate一回のroot `make check`、harness `make check`/`make distcheck`の結果は証跡どおりかを確認するが、QAとして再実行しない。candidateが許可パスの新packageと必要ならREADMEだけに収まり、dependency/config/generated artifactがなく、base...candidate差分の追加＋削除が1,000行以下であることを確認する。 | `evidence-review` / 包括checkはcandidate作成時の一回の証跡を独立監査する対象であり、QAによる重複実行は不要である。形式だけのdigest/cache/重複candidate転記は要求しない。 |
| QA-004 | 対象外およびAC-3〜AC-6の環境依存境界 | 実GitHub/OpenAI、実Internet DNS/TLS/system trust、実認証情報、実Agent proxyを通じたprovider受理とresponse伝達を確認する。 | `live-e2e` / `blocked`。承認済み実環境、credentials、および安全なcleanupがTaskに用意されず、恒久外部network testも対象外である。QA-001〜003のPASSはこのケースをPASSにしない。 |

## 境界・異常・回帰

- QA-001の失敗はfixture、candidate、または実行環境に分類し、実装不具合とは決めつけない。
- scope再評価、transport前拒否、header allowlist、body close、上限/JSON/media/status/read/close/timeout/sink異常、response+error、単回呼出、copy/non-leakの各検出が欠けるか弱められた場合は、focused-rerunのPASSを出さない。
- QA-002では非2xx又は不正responseをsinkへ渡さないこと、sink失敗をretryしないこと、公開error/formatに秘密・接続先・provider本文・下位error detailを出さないことを特に監査する。
- QA-003ではHANDOVER以外へcandidate hash、tree、digestを重複転記せず、成功stdoutの永続化も要求しない。差分外の製品変更又は安全契約の意味変更を発見した場合はPASSにせずMainへ再分類を報告する。
- QA-004は環境依存の未実施を明示したままにし、承認済み実環境がなければpost-mergeでも`blocked`を維持する。

## 実装後の再確認

- [ ] HANDOVERの`candidate_commit`を唯一のcandidate識別として、同一candidateをQA-001〜003で評価した。
- [ ] QA-001のfocused rerunを一回だけ実行し、結果、case coverage、失敗分類をQA_RESULTへ記録した。
- [ ] QA-002〜003のcandidate-bound source/test/HANDOVER証跡を独立監査した。
- [ ] QAとしてroot/harness包括 `make check` を再実行せず、candidate時の証跡だけを監査した。
- [ ] QA-004をunit PASSと分離して`blocked`のまま記録した。
- [ ] 期待結果又は範囲を変更した場合、Main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-01 | QA | DEV前の独立QA計画 | `approved` |
