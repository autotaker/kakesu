---
task_id: "TASK-0058"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T18:08:34Z"
revision: 1
implementation_reviewed_at: "2026-08-01T18:45:08Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0058 QA PLAN

## QA scope

期待値の正本は `TASK.md` の `Planning input packet` だけとする。PLAN、実装、DEV自己申告は期待値の根拠にしない。candidate の許可path内で、strict config と provision manifest、秘密読込より前のruntime identity 検証、既存trusted constructorによる一つのservice graph、socket取得を最後にするlifecycle、CLI/systemd wiring と固定非漏洩診断を独立に評価する。

既存のpolicy、credential、transport、HTTP/TLS、socket、peer の内部意味を変更すること、capability発行・既定handle、broker/approval IPC、永続化、audit、rotation/reload/restart復元は範囲外である。実Linux/systemd、実secret、provider、VPSの事実は下記 `live-e2e` としてblockedであり、hermetic test又はcross-compileをそのPASSにしない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 確認内容と failure detection | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | config V1 が必須 `egress.github_repositories` と `egress.openai_models` を独立copyし、各1〜32件、重複なし、既存egresspolicyで受理可能な値だけを受理することを確認する。missing、empty、33件、duplicate、unknown field/key、同一値の別表現、不正repository/modelを既存fixed classで拒否することを、candidate testの負のassertionで確認する。example、command/provision fixtureが必須値と一致し、manifestに固定 `config_dir/credentials` のbroker-owned `0700` directoryが一件だけ加わりaction count=11、既存user・directory・serviceの順序と意味が変わらないことを確認する。 | `focused-rerun` / parser、fixture、manifestはtemp fixtureでhermetic・deterministic・boundedに再現できる。 |
| QA-002 | AC-2 | startup が config load → runtime identity resolve → `config_dir/credentials` bundle load → trusted graph construction → socket Take → Serve を各一回だけ行うことを確認する。各前段の故意failureで全後段callが0、Take前にlistenerを持たず、全constructor成功後だけTakeし、Take後のlistener所有権がServerへ一回だけ移ることをcall log/counterでfailure-detectする。credentialをidentity前に読む、socketを早く取得する、partial service/listenerを残す、retry/追加lookupを検出する。 | `focused-rerun` / package-private fake factory、identity、credential、activation、Server seamにより外部OS/secretなしで順序、exact call、ownershipを直接観測できる。 |
| QA-003 | AC-3 | 承認済み既存constructorだけで、一つのpolicy、空Registry、upstream transport、provider resolver、Exchange、context-only handler、proxy CA Session、PeerBinder、Serverを構成することを確認する。policy `egress-v1`、TTL 10分、uses 16、epoch 1、body 64 KiB、output 4096、credential 4096 byte、provider 10秒、forward 30秒、response 1 MiB、connection 16をexact assertionする。同じresolved identity snapshotのbroker UID/agent GIDがsocketへ、agent UID/Subjectがpeerへ渡ること、空Registryの未知handleがdenyすることを確認する。nil/invalid dependency、constructor failure、identity snapshot混在、default client、fallback/retry/cache、network/identity再lookupを検出する。 | `focused-rerun` / fake dependencyとcounting constructorにより、graphと固定値・identity共有・deny境界を決定的に再現できる。 |
| QA-004 | AC-4 | `dev-agent-egress serve --config PATH` だけが起動面であることを確認する。`--version` とno-args fail-closed契約を維持し、unknown/subcommand、missing/empty/duplicate config argument、invalid config/identity/credential/dependency/socket、Serve error が固定exit/errorへ畳まれ、path、secret、provider、下位errorを漏らさないことを確認する。SIGINT/SIGTERMがcontext cancelへ一回だけ変換されること、systemd serviceが固定config pathとsocket unitを宣言しcredential/env/capabilityをargv/environmentへ渡さないことをcandidate source/testで確認する。 | `focused-rerun` / command signal/failure seamとstatic unit fixtureはhermeticかつboundedである。実systemd起動はQA-006へ分離する。 |
| QA-005 | AC-5 | QA-001〜004のtestが正常系だけでなく、順序違反、call重複、listener取得前後failure、ownership、cancel/Serve failure、fixed diagnostics、empty Registry denialを実際に失敗検出でき、skip・mockだけの自己充足・assertion弱体化がないことを監査する。Linux targetがcross-compile可能であること、READMEが実secret/provider/VPSの保証を主張せず境界を記すこと、許可path外、外部dependency、既存core semantics変更、生成物混入、追加＋削除が1,100行超でないことを確認する。harness `make check`/`make distcheck`、root `make lint-docs`、candidate root `make check`、`git diff --check`は同一candidateのDEV evidenceとしてcommand/cwd/resultを独立監査し、QAは重複実行しない。 | `evidence-review` / candidate diff、test本文、HANDOVERとDEVのcandidate-bound証跡を突合する。高リスクの挙動確認はQA-001〜004のfocused rerunに限定する。 |
| QA-006 | 対象外 / AC-2〜AC-4の実環境境界 | 実Linux NSS/UID/GID、systemd socket activation FD 3・socket permission・unit install/restart、broker-owned実secret配置、実GitHub/OpenAIとDNS/TLS、実Agent client、VPS上のserve/cancel/rollback/cleanupを確認する。 | `live-e2e` — `blocked`。承認済み隔離Linux/VPS、実主体・secret/provider権限、実行手順、安全なcleanup/rollbackがTaskに用意されていない。他ケースのPASSで置換しない。 |

## 一つの bounded focused rerun

candidate をHANDOVERの `candidate_commit` に固定後、QA-001〜004を次の一つのbounded rerunで一回だけ実行する。前半はcandidate上のhermetic Go testをcacheなしで実行し、後半はLinux binaryをhostで起動しないcompile-onlyである。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -count=1 ./internal/config ./internal/command ./internal/provision ./internal/egressservice ./cmd/dev-agent-egress && GOOS=linux GOARCH=amd64 GOCACHE=$PWD/.build/go-cache go test -run '^$' -exec /usr/bin/true ./internal/egressservice ./cmd/dev-agent-egress
```

双方のzero exit、対象testの存在、candidate一致、負のassertion、deterministicかつboundedなfake/seamが必要である。一つでも欠ける、またはtestが実OS/systemd/secret/providerの真実を偽装するなら該当ケースはPASSにしない。後半はLinux runtime、NSS、FD 3、permission、実配置又はVPSの証拠ではない。

## 境界・異常・回帰

- config allowlist規則は既存egresspolicyを再実装せず、config copy後のcaller mutationがservice policyを汚染しない。credentials pathは固定childで、config/environment/argvから選択しない。
- identityは一度だけ解決し、同じsnapshotだけをsocket/peerの両境界へ渡す。credential bundleはその後一度だけ読み、失敗時にsecret、input path、identity/provider/dependency detailを診断へ露出しない。
- 全trusted constructor成功まではsocketを取得せず、失敗はfail closedでpartial listener/serviceなし。Take後はServer以外がclose/serve/reuseせず、cancel及びServe failureは固定診断で終了する。
- Registryは空のままとし、issuer、静的default handle、approval連携を追加しない。default HTTP client、fallback、retry、cache、追加goroutine、diagnostic log、追加network/identity lookupは回帰として扱う。
- systemd unitのcredential/env/capability argv/environment渡し、固定config/socket wiring欠落、`serve --config PATH`以外のoperational面、生成`harness.json.example`混入、許可path外又は外部dependencyはscope/regression failureとして扱う。
- QAのfailureは実装不具合と決めつけず、candidate、environment、dependency、requirement、QA計画又は証跡のどれかに分類する。TASK期待値の矛盾は補正せず `requirement_gap` としてMainへ報告する。

## Result criteria

QA_RESULTには各ケースの判定、focused rerunのcommand/result、QA-005で監査したDEV evidence、QA-006のblocked理由と、findingの再現根拠を記録する。root/harness全checkのQA再実行、artifact digest、review version、carry-forward証明は要求しない。

## 実装後の再確認

- [x] 同一candidateに対しQA-001〜005を独立に評価し、指定focused rerunを一回だけ実行した。
- [x] strict config/manifest、startup exact order、identity共有、empty Registry deny、listener ownership、CLI/diagnostic/systemd wiringのnegative failure-detectionを確認した。
- [x] root/harness checksはDEV candidate evidenceとしてだけ監査し、QAが重複実行していない。
- [x] QA-006の実Linux/systemd/secret/provider/VPSをblockedのままとし、期待値又はscopeを変更していない。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-firstの独立QA計画。AC-1〜AC-5、single bounded rerun、実環境live-e2e blocked境界を定義。 | `approved` |
