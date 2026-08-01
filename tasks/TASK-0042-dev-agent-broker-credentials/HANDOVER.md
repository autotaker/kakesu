---
task_id: "TASK-0042"
status: complete
completed_at: "2026-08-01T07:19:20Z"
candidate_commit: "5fea9b870191a33aeb83d2726cb341f41564f14a"
---

# TASK-0042 HANDOVER

## 成果

- broker専用の固定4-file directoryからGitHub App client ID、installation ID、RSA private key、OpenAI API keyをfail-closedに読み込む境界を実装した。
- 標準libraryだけでGitHub App用RS256 JWTを生成し、秘密をAgent、error、既定format、READMEへ露出しないAPIへ限定した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `GOCACHE=/tmp/task-0042-fix-gocache go test -race ./internal/brokercredentials`（review修正後） | PASS |
| `./configure && make check`（harness） | PASS |
| `make distcheck`（harness） | PASS |
| Linux amd64/arm64 `go test -c`（review修正後） | PASS |
| `make check`（candidate launcher、成功candidateで一回） | PASS |
| `pnpm lint:docs` / terminology validation / `git diff --check` | PASS |

## 主要な変更

- Linuxでは一度開いたdirectory FDから固定basenameだけを`openat`し、owner/mode、regular file、size、読込み前後metadataを検証する。非Linux readerは開発test用途に限定した。
- client ID、installation ID、OpenAI key、単一PKCS#1/PKCS#8 RSA PEMをstrictに検証し、raw PEM/private keyを返さない。
- JWTは固定2-field headerと、整数Unix秒`iat=now-60`、`exp=now+540`、`iss=client ID`だけのpayloadをRS256署名する。同一秒の同一JWTを許す。
- `Bundle.Format`は全format verbを固定ラベルへredactし、Goの既定formatによるunexported secret漏洩を防ぐ。

## 検証結果

- `make check`: PASS（candidate `5fea9b870191a33aeb83d2726cb341f41564f14a`）
- candidate差分は許可9 files、追加818・削除0、合計818行（上限1,200以下）。

## 判断・既知の制約

- 実UbuntuのUID/権限隔離と実GitHubでのJWT受理は未実施であり、ローカルunit/cross-compileのPASSで代替しない。
- GitHub installation token交換、cache、resolver接続、HTTP/network、Credential rotateは後続Taskの責務である。
- candidate gateは実装外のREADME用語lintで二度FAILし、いずれもcandidateを作成しなかった。glossary/例外は増やさずREADME表記だけを修正した。
- 初回candidate reviewで、pointerだけに実装した`Format`を値copyが迂回する秘密漏洩を検出した。value receiverとpointer/value両方のnegative testへ修正し、旧candidateを破棄した。本candidateは修正後bytesに対する一回のroot `make check`で固定した。
