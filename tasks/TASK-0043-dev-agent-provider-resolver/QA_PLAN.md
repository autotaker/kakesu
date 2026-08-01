---
task_id: "TASK-0043"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
qa_role: "independent-qa"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T07:37:54Z"
revision: 2
implementation_reviewed_at: "2026-08-01T08:05:20Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0043 QA PLAN

## 方針

TASK.mdのPlanning input packetを唯一の要件正本とする。同一candidateに対し、秘密を出力せず、実装差分・source・DEV証跡を独立に監査する。hermeticかつdeterministicなrace検出を一回のfocused rerunに集約する。実GitHub受理および実TLS/DNS/proxy transportの安全性はunit結果で代替しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1, AC-2, AC-3, AC-4, AC-5 | candidate diff、`internal/providercredentials` source、DEVのhermetic unit証跡を監査し、constructor/zero receiver/provider・repository分岐、OpenAI no-network、GitHub requestのhost/path/method/header/body・JWT使用位置・timeout・単回`RoundTrip`、responseのstatus/media type/size/JSON一意性/token/expiry境界、expiry直前に一度だけUTC評価時刻を得る非公開clock dependency、body close、cache/retry/default transport/I/O/log不在、固定errorと秘密非漏洩を照合する。テストが各拒否境界を実際に失敗検出でき、弱体化されていないことも確認する。 | `evidence-review` / candidate sourceとcandidate-bound DEV証跡で実装境界・非漏洩・禁止事項を独立に照合できる。実ネットワーク安全性はlive-e2eに分離する。 |
| QA-002 | AC-1, AC-2, AC-3, AC-4, AC-5, AC-6 | candidateを固定後、`go test -race ./internal/providercredentials`を一回だけ独立実行する。valid OpenAI/GitHub、拒否provider/repository、完全なrequest束縛、timeout、call count、transport/status/content-type/body/JSON/token/expiryの異常、固定clockによるexpiryのnow拒否・now直後許可・65分許可・65分超拒否、body close、固定error/秘密非漏洩およびTASK-0041 transaction接続が検出対象であることを結果とtest sourceで確認する。 | `focused-rerun` / fake `RoundTripper`、runtime生成fixture、package-private固定clockに閉じたhermetic・deterministic・上限付きpackage testであり、race検出もこの一回に統合できる。 |
| QA-003 | AC-5, AC-6 | candidate diffとDEVのharness `make check`/`make distcheck`、root `make check`の実行証跡を監査する。対象packageとREADMEだけの差分、外部module/config/生成物の不在、READMEと実装の整合、base...candidateの追加＋削除1,200行以下、および包括検査のcandidate対応を確認する。 | `evidence-review` / 同一candidateでのDEV包括検査証跡と差分を独立監査する。QAは包括checkを重複再実行しない。 |
| QA-004 | AC-3, AC-4, AC-5 | 実GitHub App/installationでJWTの受理、repository一件scopeのinstallation access token交換、201 responseおよび新しいstateless token形式を確認する。 | `live-e2e` / `blocked`（not-run）。実App/installation、認可済みnetwork、実secret、およびtoken交換後の安全なcleanupがこのcandidate QAには提供されていない。fake transport又はJWT unit PASSで代替しない。 |
| QA-005 | AC-3, AC-5 | 承認済み実環境で注入transportのTLS/CA/DNS/proxy/IP policy、redirect非追従、および外部到達時のcleanupを確認する。 | `live-e2e` / `blocked`（not-run）。transport hardeningは対象外で、実network経路・安全なcleanup・承認済み環境が未提供である。local `RoundTripper` testを実transport安全性のPASSと扱わない。 |

実施モードは `evidence-review`、`focused-rerun`、`live-e2e` のいずれかとする。実施不能なケースは理由を記録し、別モードのPASSで置き換えない。

## 境界・異常・回帰

- evidence-reviewでcandidate外のpackage、外部module、config/CLI/systemd/provision、実network・process・environment/file入力・永続書込み・secret fixtureが混入していないことを確認する。
- focused rerunの失敗は、candidate実装、test/fixture、実行環境、又は証跡不整合に分類し、DEV faultを前提にしない。live-e2eのblockedはFAILやPASSへ変換しない。
- candidate固定後のsource/evidence監査とfocused rerunは同じcandidateを対象とし、candidateの変更時は本計画の適用可否を再確認する。digestの転記又は形式だけの証拠でcandidate同一性を主張しない。

## 実装後の再確認

- [x] DEV開始前にPlanning input packetだけから独立作成した。
- [x] 同一candidateのsource/evidence監査、focused rerun一回、blocked live-e2eを分離した。
- [x] 期待結果と範囲は変更していない。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-01 | QA (Terra/medium) | 独立QA計画。candidate-bound監査、race focused rerun、未提供の実GitHub/TLS環境をblocked live-e2eとして定義。 | `approved` |
| 2 | 2026-08-01 | QA (Terra/medium) | 非公開fixed clockによるexpiryのnow/直後/65分/65分超境界を明確化。既存の「現在より後かつ65分以内」をdeterministicに検証するもので、期待結果・範囲は変更しない。 | `approved` |
