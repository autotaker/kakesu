---
task_id: "TASK-0058"
status: complete
completed_at: "2026-08-01T18:45:08Z"
candidate_commit: "e539f70e52ddf05f1e1d056859633b2c6563d6e0"
---

# TASK-0058 HANDOVER

## 成果

- strictなegress allowlistとbroker-owned credential directoryをconfig/provision境界へ追加した。
- 既存のpolicy、空Registry、credential/transport、HTTP/TLS、peer binding、socket activationを一つの`dev-agent-egress`起動graphへ合成した。
- `serve --config PATH`、signal cancellation、固定systemd socket/config wiringを追加し、補助binaryのfail-closed契約を維持した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -count=1 ./internal/config ./internal/command ./internal/provision ./internal/egressservice ./cmd/dev-agent-egress` | PASS |
| `cd tools/dev-agent-harness && GOOS=linux GOARCH=amd64 GOCACHE=$PWD/.build/go-cache go test -run '^$' -exec /usr/bin/true ./internal/egressservice ./cmd/dev-agent-egress` | PASS（compile-only） |
| configured harness `make check` | PASS |
| configured harness `make distcheck` | PASS |
| `make lint-docs` | PASS |
| `make candidate-commit TASK=TASK-0058`内のroot `make check` | PASS |
| `git diff --check` | PASS |

## 主要な変更

- configは必須のGitHub repository/OpenAI model allowlistを1〜32件で検証・copyし、既存`egresspolicy`の受理規則を正本として利用する。
- serviceはconfig→identity→credentials/CA authority→constructor graph→socket Take→Serveを一回ずつ進め、欠落・失敗を固定errorへ畳む。
- 同じidentity snapshotをPeerBinderとsocket receiverへ、同じCA authority snapshotをCONNECT sessionへ渡す。空Registryの未知handle拒否もcomposition testで確認する。
- provision action 8へ`config_dir/credentials`（broker:broker `0700`）を追加し、service actionを9〜11へ移した。
- CLI/main/systemd/README/exampleと、順序・配線・nil依存・固定診断を検出するhermetic testを同期した。

## 検証結果

- candidate `e539f70e52ddf05f1e1d056859633b2c6563d6e0`（tree `d804a478b7a405a5760a784c8021fd3fdbec22d1`）に対するroot `make check`: PASS
- focused Go tests、Linux compile-only、configured harness `make check`/`make distcheck`、root lint、diff check: PASS
- candidate差分: 許可path内12ファイル、追加941/削除38（合計979行、上限1,100行以内）。生成物・外部dependency・既存core内部変更なし。

## 判断・既知の制約

- capability issuer/deliveryは対象外でRegistryは意図的に空である。未知handleは拒否され、後続Taskがtrusted発行経路を追加する。
- 実Linux NSS/systemd FD 3/socket permission、実secret、実GitHub/OpenAI、VPS live E2Eは承認済み環境と安全なcleanupが未指定のためblocked。hermetic test/cross-compileを代替PASSにしない。
