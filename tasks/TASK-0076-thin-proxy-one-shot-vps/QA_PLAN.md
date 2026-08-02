---
task_id: "TASK-0076"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T10:26:58Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0076 QA PLAN

## 独立性と判定方針

期待値の正本は `TASK.md` の `Planning input packet` だけである。PLAN、実装案、DEV自己申告、既存testは期待値の根拠にしない。QAは同一 `candidate_commit` を固定してから実施し、ケースID、mode、実行command又は監査対象、結果、未実施理由だけを `QA_RESULT.md` に記録する。実token、credential、opaque handle、request/response本文、Authorization/header値、VPS接続値は記録しない。

QA role契約は `.codex/agents/qa.toml` の `gpt-5.6-terra` / `medium` / `workspace-write` である。QAは実装、stage、commit、mergeを行わない。focused failureは candidate実装、test/fixture、実行環境、requirement、証跡不整合に分類してから報告し、DEV faultを前提にしない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法と期待結果 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1〜AC-5 | 一つのhermetic focused/race integration suiteで、repository単位request→decision→grantのAgent instance/UID、workspace、完全一致repository、短TTL、unused、revoke束縛と、barrier付き並行pushの成功/上流試行が一回だけであることを確認する。`git-receive-pack` のみを最小分類し、grantをdial/body送信前に消費すること、別repository/Agent/UID/workspace/reuse/expiry/revoke/REST転用が上流到達前に拒否されることを確認する。Git read/GitHub REST/OpenAIのopaque streaming、非2xx、非JSON、任意Content-Type、1 MiB超、slow reader/writerでのbackpressureをstatus/headers/sentinel bytes/counting fakeで確認する。peer/handle/host/credential/TLS CONNECT/local CA/timeout/concurrency/header上限、credential非露出、Tailscale identity/Passkey UVからの同一decision、主文言「このrepositoryへの次のpush一回」と参考情報が認可判定を変えないことも確認する。candidate diff/testの独立監査では、provider意味検査、全量buffer、重複責務wrapper/dead layerがなく、既存negative testの失敗検出能力が維持され、test弱体化がないことを確認する。 | `focused-rerun` / diff/test監査は実行前の成立条件であり、欠落時はPASSにしない。clock、registry、peer、transport、credential、WebAuthn/Tailscaleをfakeに閉じ、atomicity、stream、secret境界、UI結線を決定的・boundedに再現できる。 |
| QA-002 | AC-7 | clean temporary build rootで `./configure && make && make install DESTDIR=<temporary>/package-root` を一連で実行する。staged treeに実装と一致するbinary、設定例、systemd/sysusers/tmpfiles成果物があり、実hostのservice/user/secret/networkを変更しないことを確認する。runbookのinstall、rollback、cleanup記載と実装/configure出力をcandidate-boundで照合する。DEV candidate transactionの `make check` は再実行せず、そのcommand、candidateとの対応、exit、failure-detection証跡を独立に監査する。 | `focused-rerun` / temporary DESTDIRでbuild/install成果物を外部作用なしに再現できる。実systemd restart/VPS rollbackはQA-003のみ。 |
| QA-003 | AC-6 | 承認済み実VPSで、Agent userが実Credentialを読めないことを秘密値を表示せずアクセス結果だけで確認し、Git pull、限定GitHub REST、OpenAI API、スマートフォンPasskey承認後のtest branchへの一回pushを実行する。別repository、reuse、expiry、REST転用は上流外部作用なしで拒否されることを確認し、push後はtest branch/commit、service配置、必要なgrant/pending stateをrunbookどおりcleanup又はrollbackする。 | `live-e2e` / 実UID、TLS/外部provider、Tailscale、Passkey、systemd配置、実credentialとcleanupはhermetic化できない。**現在 `blocked/not-run`**：Mainのdependency-ready reconciliationで、秘密を含まないVPS識別子、operator/Tailscale identity識別子、Passkey enrollment識別子、repository識別子、使い捨てtest branch識別子、GitHub App installation識別子、OpenAI test credentialの参照識別子、配置対象識別子、rollback/cleanup所有者・手順が承認済みになるまで再開しない。QA-001〜002のPASSで代替しない。 |

## focused command群と証跡

candidate固定後、QAは次の最小command群を一回ずつ実行する。実装により新設packageがある場合も、QA-001を包含する単一のrace commandから除外しない。install fixtureはQA所有の `mktemp -d` 配下にのみ生成し、検査後に同じ一時directoryをcleanupする。workspace内に `package-root` を残さず、候補差分・ignore規則・Git状態を変更しない。

```sh
cd tools/dev-agent-harness && GOCACHE="$PWD/.build/go-cache" go test -count=1 -race ./...
cd tools/dev-agent-harness && ./configure && make && make install DESTDIR="<temporary>/package-root"
```

出力はcaseごとのexit、対象package/test名、成功/失敗分類、secret-free要約だけを残す。ログには `Authorization`、cookie、credential、handle、URL query、request/response body、VPS host/IP、Passkey assertionを転記せず、必要なら「sentinel non-observation」「upstream count=0/1」「status class」「staged relative path count」のような非秘密観測値に置換する。失敗出力に秘密が混在し得る場合は本文を証跡化せず、隔離してredaction可否をMainへ報告する。

## 境界・異常・回帰

- non-2xx/非JSON/任意Content-Type/1 MiB超を拒否、変換、bufferする、stream時に無制限に先読みする、又はprovider API/ Git wire本文を認可のため解釈するものはFAIL候補である。
- grantをupstream試行又はbody送信後に消費する、並行で二つ以上上流到達する、mismatch/rejectで有効grantを失う、cross-repository/instance/UID/workspace/reuse/expiry/revoke/REST転用が上流到達するものはFAIL候補である。
- real credential、opaque handle、Agent由来認証値、本文又は下位errorがAgent環境、proxy応答、audit、test outputへ露出するものはFAIL候補である。
- build/install成果物、設定例、systemd units、runbookの矛盾はAC-7のFAIL候補である。live前提値又は安全なcleanupが未確定ならQA-003はblockedのままとし、実行もPASS主張もしない。

## 実装後の再確認

- [ ] HANDOVERの `candidate_commit` とQA対象treeを固定し、candidateの製品差分だけを確認した。
- [ ] QA-001〜002を独立に実施し、各caseのcommand/監査、結果、分類を記録した。
- [ ] QA-003のdependency-ready reconciliationを確認した。未readyなら `blocked/not-run` を維持した。
- [ ] 期待値または範囲の変更が必要ならQAが補正せず、Main承認を待った。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-first初版。opaque streaming、one-shot atomicity、credential境界、UI、削除責務、configure/install、pending live VPSを独立ケースへ固定。 | `main-agent-sol-high / 2026-08-02T10:26:58Z` |
