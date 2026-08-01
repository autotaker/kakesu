---
task_id: "TASK-0045"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T08:37:57Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0045 QA PLAN

## 方針

TASK.mdだけを要件正本として、candidateの同一案をDEVおよびREVIEWから独立に評価する。hermeticかつ決定的なpackage testは一回だけ再実行する。実Internet、実DNS/プロキシ/firewall、実system trust storeを必要とする主張はlive E2Eのblocked境界として残し、unitのPASSで代替しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1〜AC-6 | candidateで `go test -race ./internal/upstreamtransport` を一回実行する。runtime生成CA/certificateと注入resolver/dialerによるhermetic fixtureが、許可/拒否origin、URL authorityと異なる非empty `Request.Host`のDNS/dial前拒否、DNS一回と検査済みIP:443だけへのdial、mixed/unsafe address全体拒否、SNI・hostname certificate verification、TLS 1.2以上、HTTP/1.1、connect/TLS/header timeout・context cancel、proxy無視、keep-alive/自動compression/HTTP2/redirect/retryなし、fallback境界、error非漏洩、body closeとidle closeを検出することを確認する。 | `focused-rerun` / package内で完結し、外部networkなしに受け入れ真実を再現する決定的かつ上限付きのrace testである。 |
| QA-002 | AC-1〜AC-5 | candidateのproduction sourceとtestを監査し、originおよびURL authorityと異なる非empty `Request.Host`の検査がDNS前であること、system DNS answerの正規化・全体拒否とDNS名を使わないIP literal dial、元hostnameだけをTLS SNI/certificate verificationへ渡すことを確認する。併せてHTTP/1.1・timeout・context、system rootのみ、proxy/keep-alive/compression/HTTP2/retry/redirect無効、dial失敗時だけの未使用安全IP fallback、固定non-leak error、失敗response body closeと成功body所有権を確認する。 | `evidence-review` / QA-001の失敗検出能力と実装の安全境界への対応を、candidate source/testから独立監査する。 |
| QA-003 | AC-6 | HANDOVERをcandidate識別の唯一の正本として参照し、candidate生成とcandidate launcherの証跡を監査する。candidate launcherは成功時の完全stdoutを永続化しない前提で、成功時だけcommitすること、失敗時にbyte不変の経路であること、candidate生成とcandidate一回のroot `make check`が記録どおりであることを確認する。harness/root包括checkはこのlauncher証跡だけを監査し、QAとして再実行しない。対象packageとREADMEのbase...candidate差分が追加＋削除1,000行以下で、許可範囲外・dependency/config/generated artifactがないことも確認する。 | `evidence-review` / root checkはcandidate作成時の一回の証跡を独立監査する対象であり、同じ包括checkを再実行しない。 |
| QA-004 | 対象外およびAC-3/AC-6の環境依存境界 | 実GitHub/OpenAI到達、実Internet DNS、実system trust store、実proxy/firewallの組合せを確認する。 | `live-e2e` / `blocked`。承認済み実環境と安全なcleanupがこのTaskに用意されず、恒久外部network testも対象外である。QA-001〜003のPASSはこのケースをPASSにしない。 |

## 境界・異常・回帰

- QA-001で失敗した場合は、fixture、candidate、または実行環境のどれが原因かを分離して記録し、実装不具合とは決めつけない。
- QA-002ではテストが安全境界を弱めずに検出できることを確認する。特にhostname再dial、mixed answerの部分受理、TLS又はHTTP送信開始後の別IP retry、errorへの接続情報又はrequest秘密の転送を見逃さない。
- QA-003ではHANDOVER以外へcandidate hash、tree、digestを転記しない。成功stdoutの保存を要求せず、既定の成功時commit/失敗時byte不変のlauncher経路で評価する。
- QA-004は環境依存の未実施を明示したままにし、post-mergeでも利用可能な承認済み環境がなければblockedを維持する。

## 実装後の再確認

- [ ] HANDOVERのcandidate_commitを基に、同一candidateをQA-001〜003で評価した。
- [ ] QA-001のfocused rerunを一回だけ実行し、結果と失敗分類をQA_RESULTへ記録した。
- [ ] QA-002〜003のcandidate-bound証跡を独立監査した。
- [ ] QA-004をunit PASSと分離して`blocked`のまま記録した。
- [ ] 期待結果または範囲を変更した場合、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-01 | QA | DEV前の独立QA計画 | `approved` |
