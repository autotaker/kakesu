---
task_id: "TASK-0043"
status: complete
completed_at: "2026-08-01T08:05:20Z"
candidate_commit: "1914d7ebcd2e201b78a44e456d51e835eb6dab53"
---

# TASK-0043 HANDOVER

## 成果

- TASK-0042のbroker bundleをTASK-0041のtrusted resolver interfaceへ接続し、OpenAI keyのnetworkなし解決と、GitHub App JWTから一repository限定installation tokenへの単発交換を実装した。
- GitHub exchangeを固定endpoint、API version、timeout、1回の`RoundTrip`、上限付きstrict JSON responseへ閉じ、実Credentialをcache、log、既定format、Agentへ残さない境界にした。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `GOCACHE=/tmp/task0043-gocache go test -race ./internal/providercredentials` | PASS |
| `./configure && make check`（harness、最終byteのtemp copy） | PASS |
| `make distcheck`（harness、最終byteのtemp copy） | PASS（live tests SKIP） |
| `make lint-docs`（README表記修正後） | PASS |
| `make check`（candidate launcher、成功candidateで一回） | PASS |
| `git diff --check` | PASS |

## 主要な変更

- `openai`かつ空repositoryだけは検証済みOpenAI API keyを直接返し、JWT生成とtransportへ到達しない。
- `github`かつ正規`owner/name`だけはrepository名一件をJSON bodyへ束縛し、固定GitHub endpointへBearer JWT付きPOSTを一度だけ渡す。default transport、redirect、retry、cacheを実装しない。
- `201`、JSON media type、128 KiB、重複top-level fieldなし、visible ASCII token、現在より後かつ65分以内のRFC3339 expiryを検証し、response bodyを成功/失敗でcloseする。
- package-private固定clockとfake RoundTripperでexpiry境界、timeout、call count、body close、token長、固定error/non-leak、GitHub/OpenAI transaction接続をhermeticに検証する。
- `Resolver.Format`はpointer/valueの通常format verbを固定ラベルへredactする。

## 検証結果

- `make check`: PASS（candidate `1914d7ebcd2e201b78a44e456d51e835eb6dab53`）
- candidate差分は許可3 files、追加678・削除0、合計678行（上限1,200以下）。

## 判断・既知の制約

- 実GitHub App/installationでのJWT受理・repository scope・新token形式と、実TLS/CA/DNS/proxy/IP transportは未実施であり、fake transportのPASSで代替しない。
- installation tokenのcache/refresh/retry、default transport、request Forwarder、HTTP server、Git Smart HTTP、pushは後続Taskの責務である。
- 最初のcandidate launcherは追加READMEの既存用語lintだけでFAILし、candidateを作成しなかった。glossary/例外を増やさず、追加段落を既存の正規日本語表記へ直して`make lint-docs`を先にPASSさせた。本candidateは修正後byteに対するroot `make check`で固定した。
