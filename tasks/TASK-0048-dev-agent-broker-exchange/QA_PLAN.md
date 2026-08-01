---
task_id: "TASK-0048"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T10:55:32Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# QA_PLAN: TASK-0048

## QA scope

TASK.md の `Planning input packet` を唯一の要件正本として、candidate の
`tools/dev-agent-harness/internal/brokerexchange/` と許可される README 差分を独立に確認する。
既存 Policy、Registry、CredentialResolver、RoundTripper、Transaction、Forwarder を wrapper 内で
再実装又は変更していないことも確認対象にする。

実 provider、実 credential、Internet DNS/TLS/system trust、Agent proxy、production resolver/
transport wiring は `live-e2e` の対象だが、この Task では安全な実環境と cleanup が定義されていない。
よって blocked とし、fake dependency の結果で実 provider 受理を主張しない。

## Cases

| Case ID | 対象AC | 確認内容 | Mode | Evidence |
|---|---|---|---|---|
| QA-001 | AC-1 | `New` が Policy/Registry/CredentialResolver/RoundTripper の nil、credential 上限の 1〜4,096 byte 外、timeout の 1ms〜30秒外、response 上限の 1 byte〜1 MiB 外を固定 non-leak error で拒否し、有効 Rules から immutable な `Exchange` を返すことを確認する。nil/zero receiver の `Do` も dependency detail を出さず zero response と固定 `exchange-denied` になることを確認する。 | evidence-review | candidate source/test、HANDOVER、DEV `make check` 証跡 |
| QA-002 | AC-2 | 各 `Do(subject, request)` が call-local capture sink、Forwarder、Transaction を構成して Transaction へ subject/request を同期一回だけ渡すことを確認する。caller 所有の Body と Authorization slice を変更・保持しないこと、既定 transport/network client、redirect、retry を導入しないことを確認する。 | evidence-review | candidate source/test、HANDOVER、DEV `make check` 証跡 |
| QA-003 | AC-3 | real Policy/Registry と fake resolver/transport の integration test が GitHub REST read と OpenAI Responses text の双方で成功し、実 credential による Bearer 置換、status、空又は正規 `application/json` content type、独立本文だけを成功 response として返すことを検出するかを確認する。sink の未通知、二重通知、不整合は成功に昇格せず固定 error と zero response になることも確認する。 | evidence-review | candidate source/test、HANDOVER、DEV `make check` 証跡 |
| QA-004 | AC-4 | policy、Authorization、capability の各拒否が resolver/transport より先であることを確認する。subject 又は scope mismatch が capability を消費せず、正しい後続交換で使えること、resolver 又は transport 到達後の失敗では capability が消費されたままで、同一 handle の再試行が resolver/transport に再到達しないことを確認する。 | focused-rerun | `tools/dev-agent-harness` で一回だけ実行する `GOCACHE=$PWD/.build/go-cache go test -race ./internal/brokerexchange` |
| QA-005 | AC-4, AC-6 | fake resolver と RoundTripper の call counter により、成功、resolver failure、transport failure、消費済み handle 再試行のそれぞれで resolver/transport が一回を超えず、認可前拒否では零回であることを確認する。resolver/transport failure 後にも rollback、再発行、同一交換内 retry がないことを確認する。 | focused-rerun | `tools/dev-agent-harness` で一回だけ実行する `GOCACHE=$PWD/.build/go-cache go test -race ./internal/brokerexchange` |
| QA-006 | AC-3, AC-5, AC-6 | failure path の response が常に zero で、constructor、Transaction、Forwarder、sink、resolver、transport の任意失敗が同じ `exchange-denied` に畳まれることを確認する。error/format/long-lived state に opaque handle、credential、URL、scope、provider、request/response body、下位 dependency error が漏れない固定 non-leak 境界を確認する。 | evidence-review | candidate source/test、HANDOVER、DEV `make check` 証跡 |
| QA-007 | AC-2, AC-3, AC-5, AC-6 | input Body/Authorization の mutate・alias・保持がなく、success output body も caller/dependency と alias しないことを確認する。連続及び並行 `Do` が response state を共有・混線せず、race test が concurrent response isolation を検出することを確認する。 | focused-rerun | `tools/dev-agent-harness` で一回だけ実行する `GOCACHE=$PWD/.build/go-cache go test -race ./internal/brokerexchange` |
| QA-008 | AC-6 | hermetic race test が real Policy/Registry + fake resolver/transport で、両 provider、Bearer 置換、policy/auth/subject/scope の拒否順序、mismatch 非消費、resolver/transport failure 後の消費維持、単回呼出、zero response、input/output copy、並行 response isolation、fixed non-leak error を実際に失敗検出できることを確認する。source/test/HANDOVER から DEV が harness `make check`、harness `make distcheck`、candidate launcher の root `make check` を実行済みであること、許可 path と base...candidate の追加＋削除が 1,000 行以下であることを監査する。QA はこれら包括 check を再実行しない。 | evidence-review | candidate source/test、HANDOVER、DEV command/result |
| QA-009 | 対象外 / AC-6 の制限 | real GitHub/OpenAI、credential、DNS/TLS/system trust、Agent proxy、production resolver/transport の実配置を確認する。 | live-e2e — blocked | 実環境、権限、credential の安全な取得・cleanup、Agent proxy 経路が未定義。この blocked は他 case の PASS で代替しない。 |

## Execution rule

`focused-rerun` の QA-004、QA-005、QA-007 は同じ一回の race test の観測に束ねる。QA は
`tools/dev-agent-harness` を cwd として、次だけを一回実行する。

```sh
GOCACHE=$PWD/.build/go-cache go test -race ./internal/brokerexchange
```

それ以外は candidate の source/test、HANDOVER、DEV command/result を独立監査する。
root `make check`、harness `make check`、harness `make distcheck`、追加 process は QA では実行しない。

## Result criteria

各 case について candidate source/test と HANDOVER の事実を要件に照らして記録し、focused-rerun は
command と結果を記録する。失敗は実装不具合と決めつけず、QA ガイドラインに従って candidate、環境、
依存、要件又は証跡のいずれかへ分類する。QA-009 は実施可能になるまで blocked のままとする。

## 実装後の再確認

- [ ] candidate の source/test、HANDOVER、DEV check 証跡を独立に確認した。
- [ ] 指定 race test を candidate で一回だけ実行した。
- [ ] live-e2e blocked を PASS に置換せず、期待結果又は範囲を変更していないことを確認した。
