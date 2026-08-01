---
task_id: "TASK-0050"
status: complete
completed_at: "2026-08-01T12:21:32Z"
candidate_commit: "ffc220fcfb85e4ca234382297e20de1f78d74b94"
---

# TASK-0050 HANDOVER

## 成果

- broker memory内で単一の自己署名ECDSA P-256 CA certificate/keyを厳格に検証し、入力PEMを保持しない`proxyca.Authority`を追加した。
- exact `api.github.com` / `api.openai.com`だけに、呼出しごとに独立した短命P-256 leaf certificate/keyを発行する境界を追加した。
- 公開CA copy、fixed non-leak error、TLS hostname verify、並行発行のserial/key/SAN隔離をhermetic race testで固定した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate launcherのroot `make check`（固定前に一回） | `PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/proxyca` | `PASS` |
| `make -C tools/dev-agent-harness check` | `PASS` |
| `make -C tools/dev-agent-harness distcheck` | `PASS` |
| `make lint-docs` | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- `internal/proxyca`にRules/Clock/Authority、strict PEM・CA・key検証、certificate-only public export、2 host限定leaf発行を追加した。
- in-memory CA fixtureと`net.Pipe`により、拒否条件、extension/chain/validity、TLS HTTP/1.1 negotiation、wrong hostname、concurrent uniquenessを検査した。
- READMEにmemory-only CA境界と、file/listener/trust/live E2Eを実装していない制限を追記した。

## 検証結果

- candidate `ffc220fcfb85e4ca234382297e20de1f78d74b94`を固定した。base `1e0eabbf9bb5b3aaaf522cf1b61298ab4b2aeb4c`からの製品差分は3 files、追加590行・削除0行である。
- focused race test、harness check/distcheck、lint-docs、diff check、candidate launcherのroot checkがすべてPASSした。

## 判断・既知の制約

- private CA inputはmemoryから受け、Authorityはparse済みsigner/certificateと公開certificate copyだけを保持する。CA private materialをexportするAPIはない。
- CA file lifecycle、OS trust store、listener/CONNECT/SNI、実client/VPSは未実装・未確認であり、hermetic TLS PASSで代替しない。
- 初回lint-docsは作業ツリーの依存未配置とsandbox内DNSで停止した。依存を許可済み経路で配置後、README用語だけを修正して再実行しPASSした。glossary、check、processは変更していない。
