---
task_id: "TASK-0042"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
qa_role: "independent-qa"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T06:43:37Z"
revision: 1
implementation_reviewed_at: "2026-08-01T07:09:09Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0042 QA PLAN

## 方針

TASK.mdのplanning input packetを唯一の要件正本とする。同一candidateに対し、秘密を出力せず、実装差分・source・DEV証跡を独立に監査する。hermeticかつdeterministicなrace検出だけを一回のfocused rerunに統合する。実環境の隔離とGitHub受理はunit結果で代替しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1, AC-2, AC-3, AC-5 | candidate diff、`internal/brokercredentials` source、DEVのhermetic unit証跡を監査し、固定basename/descriptor境界、file policy、parse/PEM拒否、最小API、固定errorと非漏洩、および禁止された入力源・I/O・logがないことを確認する。テストが各拒否境界を実際に失敗検出でき、弱体化されていないことも確認する。 | `evidence-review` / ファイル操作・秘密非露出・API境界はcandidate sourceとDEV証跡を独立に照合でき、実UID隔離は別live-e2eで扱う。 |
| QA-002 | AC-1, AC-2, AC-3, AC-4, AC-6 | candidateを固定後、`go test -race ./internal/brokercredentials` を一回だけ独立実行する。成功時はrace検出を含むpackageのvalid/拒否/JWT/非漏洩テストが通り、payloadが整数Unix秒のJSON数値`iat=now-60`、`exp=now+540`と`iss`だけを持ち、同一基準秒の決定的JWTも正しく扱うことを確認する。失敗時は出力とcandidateの関係を分類する。 | `focused-rerun` / hermetic・deterministic・上限付きのpackage testであり、race検出の再現をこの一回に集約する。 |
| QA-003 | AC-4, AC-6 | candidate diffとDEVの`make check`、harness `make check`/`make distcheck`、root `make check`の実行証跡を監査する。JWTのRS256、固定2-field headerと3-field payload、整数Unix秒、各呼出しでの署名処理、固定JWT error、および変更行数上限が候補差分と整合することを確認する。 | `evidence-review` / 同一candidateでのDEV実行証跡と差分を独立監査する。QAは同じ包括検査を重複再実行しない。 |
| QA-004 | AC-1, AC-5 | 実Ubuntu上でnon-root broker UID、実所有者・mode、directory FD/openat境界、FIFO非block、およびAgentからのCredential隔離を確認する。 | `live-e2e` / `blocked`（not-run）。承認済みUbuntu実環境、専用UID、隔離された実Credentialと安全なcleanupがこのcandidate QAには用意されていない。ローカルunit PASSで代替しない。 |
| QA-005 | AC-4 | 実GitHub AppのJWT受理をGitHubに対して確認する。 | `live-e2e` / `blocked`（not-run）。実App/installation、認可済みnetwork、実secret、およびtoken交換後の安全なcleanupが対象外かつ未提供である。JWT unit検証やローカル署名PASSで代替しない。 |

実施モードは `evidence-review`、`focused-rerun`、`live-e2e` のいずれかとする。実施不能なケースは理由を記録し、別モードのPASSで置き換えない。

## 境界・異常・回帰

- evidence-reviewでcandidate外のpackage、外部module、config/CLI/systemd/provision、network・process・永続書込み・secret fixtureが混入していないことを確認する。
- focused rerunの失敗は、candidate実装、test/fixture、実行環境、又は証跡不整合に分類し、DEV faultを前提にしない。live-e2eのblockedはFAILやPASSへ変換しない。

## 実装後の再確認

- [x] 実装差分とReviewerの開始条件を確認した。
- [x] 操作手順と期待結果が現行実装に適用可能であることを確認した。
- [x] 期待結果と範囲は変更していない。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-01 | QA (Terra/medium) | 独立QA計画。candidate-bound監査、race focused rerun、未提供実環境のlive-e2eを定義。 | `approved` |
