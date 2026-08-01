---
task_id: "TASK-0045"
status: complete
completed_at: "2026-08-01T09:22:24Z"
candidate_commit: "929ad89e8b8238af4660da088ed85f14b0d8c71e"
---

# TASK-0045 HANDOVER

## 成果

- brokerからGitHub/OpenAIへ出る固定allowlistの`http.RoundTripper`を`internal/upstreamtransport`へ追加した。
- DNS answer全体を検査して安全なIP literalへだけdialし、元hostnameでTLS証明書とSNIを検証する一request一接続境界を実装した。
- runtime生成certificateと`net.Pipe`を使うhermetic testで、外部networkなしにDNS/TLS/retry/error/所有権境界を検出する。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|
| `go test -race ./internal/upstreamtransport` | PASS。最終codeで実施 |
| harness `make check` / `make distcheck` | PASS。P1修正前の全package codeで各一回。後続P1差分はHTTP version判定とfocused testだけ |
| candidate launcherのroot `make check` | PASS。最終candidateで一回完走 |
| `git diff --check` | PASS |

## 主要な変更

- 許可originと`Request.Host`をDNS前に検査し、system resolverの全answerを正規化してunsafe/mixed集合を拒否する。
- 検査済みIPの443番へだけdialし、TCP接続失敗時だけ次の検査済みIPへ進む。TLS/HTTP開始後は再dialしない。
- TLS 1.2以上、system root、元hostname verification、HTTP/1.1だけを使い、環境proxy、keep-alive、自動compression、HTTP/2、redirect/retryを持たない。
- 失敗detailを固定errorへ畳み、error併存response bodyをcloseし、成功bodyだけをcallerへ渡す。公開Formatも固定labelである。
- 差分は許可3 path、769追加行で、外部dependency、config、CLI、既存package変更はない。

## 検証結果

- focused race testはnil/zero、両origin、Host mismatch、DNS一回、unsafe/mixed/dedupe、IP:443、SNI/certificate、TLS 1.1拒否、HTTP/1.1、dial-only fallback、TLS/HTTP後no-redial、context、proxy env無視、固定設定、non-leak、body closeをPASSした。
- 最初の独立REVIEWはGoのHTTP1設定がHTTP/1.0 responseも含む違反を検出した。成功responseをHTTP/1.1完全一致へ限定し、HTTP/1.0 body close testを追加した後、focused race testと新candidateのroot checkをPASSした。
- 最初のcandidate launcherはREADMEの既存用語12件だけでdocs lintをfail-fastし、product build/testへ進まなかった。glossaryやruleを増やさず既存正規表記へ直し、再実行でPASSした。
- DEVの包括harness検査をMainの最終test拡張前に一度早く実行したため、最終codeで各一回再実行した。次Taskではfocused testとMain差分監査の後に包括検査を置く。

## 判断・既知の制約

- 一requestごとに新しいinner `http.Transport`を作りkeep-aliveを無効化する。pool/HTTP2/retry最適化は実測された必要性が出るまで追加しない。
- 実GitHub/OpenAI、実Internet DNS、実system trust store、実proxy/firewallは未実施であり、hermetic PASSで代替しない。
- Agent向けproxy/CONNECT/TLS interception、Forwarder、Authorization置換、response capture、Git push/pullは後続Taskの責務である。
