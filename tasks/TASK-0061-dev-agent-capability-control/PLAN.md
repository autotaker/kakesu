---
task_id: "TASK-0061"
change_class: "product"
status: "approved"
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "credential-bearing egress、peer-bound authorization、strict protocol parsing、single-use capability lifecycleを同一Registryと既存CONNECT sessionに安全に統合する高リスク実装であるため。"
approved_dev_profile_risk_signals:
  - "credential-bearing egress boundary"
  - "peer-bound authorization"
  - "protocol parsing"
  - "single-use capability lifecycle"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T00:40:39Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T00:40:39Z"
classification_approval_reason: "既存egress socketの外部観測可能なcontrol protocol、peer-bound認可、Capability発行・失効・消費経路を変更するため。"
---

# PLAN: Agent向けCapability発行・失効control sessionを実装する

## 根拠と分類

本計画の唯一の要求根拠は`TASK.md`の`Planning input packet`である。既存egress socketで外部観測可能なcontrol protocolとGitHub/OpenAI transactionへのCapability供給を追加するため、`change_class`は`product`とする。Kakesu本体runtime、Go workspace、Schema、依存、生成物、配布境界およびlive VPS状態は変更しない。

MainはDEV開始前に本PLANとTASK-firstの独立QA_PLANについて、意図、scope、受け入れ経路の一致を確認する。本PLANに独立計画レビューは設けない。DEV後の同一candidateに対する独立REVIEW/QAは必須である。

## 変更境界と制約

変更候補は次の8パスだけである。

- `tools/dev-agent-harness/internal/capabilitycontrol/control.go`（新規）
- `tools/dev-agent-harness/internal/capabilitycontrol/control_test.go`（新規）
- `tools/dev-agent-harness/internal/connectsession/session.go`
- `tools/dev-agent-harness/internal/connectsession/session_test.go`
- `tools/dev-agent-harness/internal/egressservice/service.go`
- `tools/dev-agent-harness/internal/egressservice/service_test.go`
- `tools/dev-agent-harness/internal/capability/capability.go`
- `tools/dev-agent-harness/README.md`

candidateはおよそ1,000行以内を目標とする。新socket、unit、listener、TCP/localhost入口、別registry、永続化、cache、retry、監査永続化、credential copy、Unix socket client/helper、launcher、環境変数注入、依存追加は導入しない。Git Smart HTTP、`github.com`、push、Approval、Tailscale、Passkey、Codex auth例外も対象外とする。

## 実施設計

1. 既存egress socket内にbounded control protocolを追加する。
   - `connectsession`は通常のHTTPS `CONNECT`を従前どおりの経路に通し、controlをCONNECTと明確に区別する非CONNECTの一接続一操作として分岐する。
   - control wire contractは次だけとする。通常proxyはrequest lineが厳密に`CONNECT api.github.com:443 HTTP/1.1`又は`CONNECT api.openai.com:443 HTTP/1.1`である既存経路だけであり、それ以外の非CONNECTは下記control候補として一度だけparseする。controlにはTLS CONNECT response、CA発行、inner HTTP handlerを通さない。

     | 操作 | request line | 許可header | body | 成功response |
     |---|---|---|---|---|
     | issue | `POST /v1/capabilities HTTP/1.1` | 一個の`Content-Length: <n>`と一個の`Content-Type: application/json`だけ | `n`は1〜512、UTF-8 JSON object。`{"provider":"github","repository":"owner/repo"}`又は`{"provider":"openai"}`だけを受理する。key重複、unknown key、欠落、null、余分な値、`model`、`subject`、TTL/use/scope指定は拒否する。 | `HTTP/1.1 200 OK`、`Content-Type: application/json`、`Content-Length`、`Connection: close`。bodyは`{"handle":"cap_..."}`のみ。 |
     | revoke | `DELETE /v1/capabilities/cap_... HTTP/1.1` | 一個の`Content-Length: 0`だけ | なし。handleはpathの正規opaque handleだけであり、query/fragment/JSON field/subject/provider/repositoryを受けない。 | `HTTP/1.1 204 No Content`、`Content-Length: 0`、`Connection: close`。bodyなし。 |

     `Content-Length`は十進のcanonical non-negative integerで、宣言値と受信body bytesが完全一致しなければならない。request lineとheader全体は既存`maxConnectHeader`（16 KiB）以内とし、header name/valueのASCII grammar、重複なし、上表以外のheaderなしを要求する。`Host`、`Connection`、`Proxy-Connection`、`Transfer-Encoding`/chunked、`Trailer`、`Upgrade`、authorization、keep-aliveは全て拒否する。body読了後のextra/early byte、pipelining、複数request、EOF前の不足body、over-limit入力も拒否する。全てのcontrol拒否（unknown method/path/version、format/size/policy/issuer/revoke errorを含む）は、credential lookup前に固定`HTTP/1.1 403 Forbidden`、`Content-Length: 0`、`Connection: close`、bodyなしで返し、write後またはwrite不能時にもconnectionをcloseする。成功後も同様に必ずcloseする。レスポンス/diagnosticはrequest値、handle（成功issue body以外）、URL、allowlist、subject、credential、下位errorを含めない。
   - issue入力はproviderと最小scopeだけを受け、GitHubでは正規`owner/repo`一件を受理する。OpenAI issueはmodelを受けず、設定の`openai_models`が非空の場合だけ`provider=openai`のCapabilityを発行する。OpenAI modelは既存egress policyがrequest bodyで検査し、control protocolとCapability scopeへ追加しない。revoke入力は正規opaque handle一件だけに限定する。subject（Agent instance、UID、workspace）はrequestから受けず、既存kernel peer binderのcontextだけから取得する。
   - 成功issue応答は`cap_...` handleだけを返す。拒否・失敗応答とログ診断には認証情報、入力値、handle、URL、allowlist設定値、下位エラーを含めない。

2. control issuerを既存Registryの正規操作に接続する。
   - 新規`capabilitycontrol` packageはparser/sessionから切り離し、peer-bound contextと既存設定allowlistを検証して、短命・回数制限付きのopaque capabilityを既存`capability.Registry`へ発行する。
   - GitHub scopeは設定allowlist内の正規repositoryだけを照合する。OpenAIは`openai_models`が非空の場合だけ、既存Registryの`provider + operation + host` scopeで発行する。model照合は既存egress policyのrequest body検査のまま維持する。provider、repository（GitHub）、operation、host、subjectの不一致、unknown/malformed/期限切れhandleは固定拒否する。
   - `capability.go`に、handleとpeer-derived Agent instance/UID/workspaceの完全一致を要求する最小のsubject-bound revoke APIを追加する。revokeは正規handle一件だけを同じRegistryで失効し、unknown、malformed、不正主体のrevokeはgrant情報を露出せず拒否する。
   - 発行/失効はin-memory Registryのみを使い、保存、rollback、retry、別registryを持たない。

3. 同一Registryを既存egress transactionへ配線する。
   - `egressservice`のcontrol sessionと既存`egresstransaction`へ、service構成時に同一のRegistry instanceを渡す。Registry lifecycle、peer identity、provider credential置換、GitHub REST/OpenAI policyの既存意味を変更しない。
   - 発行済みhandleは既存transactionが対応scopeで一回だけ消費する。不一致または再利用は既存拒否境界で失敗し、control側もtransaction側も失敗をrollback/retryしない。

4. Capability lifecycleの最小拡張を行う。
   - `capability.go`には、peer-bound subject、provider、operation、host、GitHub repository scope、TTL、使用回数、正規handleの発行・照合・消費・subject-bound revokeに必要な最小APIだけを追加する。OpenAI model scopeは追加しない。
   - issuer/transactionが同じRegistry instanceを使うこと、失効・期限・使用上限・scope不一致が消費前に固定拒否されることを守る。実credentialをCapabilityまたはAgentへ複製しない。

5. READMEを実装済み境界に同期する。
   - 既存socket上のcontrolがpeer-boundで、issue/revokeとPhase 1のGitHub/OpenAI scopeだけを扱うこと、handle以外の秘密を返さないこと、client/helperは次Taskであることを記載する。
   - live DNS/TLS、実GitHub/OpenAI、実NSS/別UID、systemd socket、VPS配置はhermetic testで証明しない対象外として区別する。

## 受け入れ条件への対応

| AC | 実施・検証の対応 |
|---|---|
| AC-1 | peer contextのみからsubjectを得るissuerと、GitHub repository allowlist、非空`openai_models`による`provider=openai`発行、TTL・回数上限のfocused tests。OpenAI model検査は既存egress policyへ維持する。 |
| AC-2 | `POST /v1/capabilities`と`DELETE /v1/capabilities/cap_...`のexact wire contract（16 KiB header、issue body 1〜512 bytes、唯一の許可header、canonical Content-Length、固定403空body/close、成功200 JSON handle又は204空body/close）のnegative testsと、通常CONNECTが従前どおり通るsession regression tests。 |
| AC-3 | service compositionで同一Registryを共有し、issue→既存transaction消費→再利用拒否までを確認するhermetic composition test。 |
| AC-4 | peer-derived Agent instance/UID/workspaceの完全一致を要求するsubject-bound revoke、正規handleのrevoke後拒否、unknown/malformed/不正主体拒否、秘密・handle・URL・設定・下位エラー非露出のnegative tests。 |
| AC-5 | CONNECT/TLS/HTTP、GitHub REST/OpenAI policy、credential置換、socket/peer identityの既存回帰をfocused Go testsとraceで確認する。 |
| AC-6 | 許可8パスと約1,000行規模、対象外runtime・Schema・依存・配布境界・live VPS差分なしをdiffで監査する。 |

## テスト設計

- `capabilitycontrol/control_test.go`: peer-derived subject、GitHub repository allowlist、非空`openai_models`によるprovider-only発行、TTL/使用上限、正規発行、scope/subject/provider/repository/operation/host不一致、subject-bound revoke、失効、診断非露出を単体で検証する。OpenAI modelのcontrol入力・Capability scopeは検証対象にしない。
- `connectsession/session_test.go`: controlとCONNECTの判別、issue/revokeのexact method/path/version、唯一の許可header、canonicalかつ実byte一致のContent-Length（header 16 KiB、issue 1〜512 bytes、revoke 0 bytes）、issue JSONのprovider/repository field table、成功200のhandle-only JSONと204空body、固定403空body/`Connection: close`をbyte単位で検証する。`Host`、duplicate/unknown header、malformed/unknown/overlong/early bytes、chunked/upgrade/keep-alive、JSON duplicate/unknown field、model/subject、複数操作を拒否し、CONNECT/TLS/HTTP非回帰を検証する。
- `egressservice/service_test.go`: 同一Registryでissueし、既存GitHub/OpenAI transactionが一回消費し、再利用と失効後使用を拒否するcompositionを検証する。
- Registryの発行/消費/失効の不変条件は、許可済み`capabilitycontrol/control_test.go`と`egressservice/service_test.go`から検証する。`capability` packageの新規test pathは追加しない。

DEVはfocused Go tests、race、root `make check`、`git diff --check`を実行する。QA_PLANは各ケースを`focused-rerun`、`evidence-review`、`live-e2e`へ理由付きで割り当てる。bounded protocolとRegistry lifecycleはhermetic・deterministicな`focused-rerun`とし、candidate-boundのテスト証跡と弱体化有無は`evidence-review`で独立監査する。実GitHub/OpenAI、DNS/TLS、NSS/別UID、systemd、VPSに依存するケースは安全な実環境とcleanupがなければ`live-e2e`をblockedのままとし、hermetic PASSで置換しない。

## 実施・完了経路

DEV開始前にMainがPLANと独立QA_PLANを承認する。DEV AgentはTask worktreeの許可8パスに製品差分だけを実装し、candidateを一回だけ固定する。Reviewer AgentとQA Agentは同じcandidateを相互のPASSを待たず独立に評価する。Mainだけがcandidate識別子、`--no-ff --no-commit`検査、completion transaction、main統合、必要なマージ後のlive-e2e判断を所有する。

## リスクと復旧

- control parserがCONNECTを誤分類するリスクは、明確な非CONNECT分岐とCONNECT/TLS/HTTP regression testsで抑える。
- Agent入力でsubject/scopeが拡大するリスクは、subjectをpeer contextのみから採用し、allowlistと正規scope検証をissuerに閉じ込めて抑える。
- 発行と消費が別Registryとなるリスクは、service composition testで同一instanceとissue→consume→reuse rejectionを検証して抑える。
- secretまたはgrant内部情報が失敗診断に漏れるリスクは、固定拒否とnegative assertionsで抑える。
- live環境の未確認をhermetic PASSで代替するリスクは、QA_PLANでlive-e2eを個別にblocked/not-applicableとして扱う。

復旧時は許可8パスのcandidate差分だけを戻し、既存egress session、Registry、transactionの経路へ復元する。外部credential、socket、registry永続状態、live VPS状態を作らないため、追加の環境cleanupは発生しない。戻した後はfocused Go tests、race、`make check`、`git diff --check`を再実行する。

## 引き継ぎ条件

DEVは承認済みPLANと独立QA_PLANの後に開始し、新socket/unit/dependency/persistence/client/helperを追加せず、許可8パス・約1,000行規模を守る。独立実装REVIEW/QAは同一candidateから実施し、MainだけがGit統合を行う。
