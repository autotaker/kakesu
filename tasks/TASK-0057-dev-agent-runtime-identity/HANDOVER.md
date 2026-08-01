---
task_id: "TASK-0057"
status: complete
completed_at: "2026-08-01T17:53:59Z"
candidate_commit: "c8af879c9b93b385bfbcc81b83f0145411320e11"
---

# TASK-0057 HANDOVER

## 成果

- config V1へ必須の`identity.workspace_id`を追加し、既存のstrict JSON境界とconsumer fixtureを同期した。
- Linuxのagent/broker userとagent同名groupを一度ずつ解決し、現在のnon-root broker EUID、agent UID/GID、workspace、fresh AgentInstanceIDを一つのimmutable resultへ固定する`internal/runtimeidentity`を追加した。
- 非Linux、lookup/numeric/EUID/GID/entropy不整合をpartial resultなしの固定エラーで拒否し、診断へのidentity値流出を防いだ。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -count=1 ./internal/runtimeidentity ./internal/config ./internal/command ./internal/provision` | PASS |
| `cd tools/dev-agent-harness && GOOS=linux GOARCH=amd64 GOCACHE=$PWD/.build/go-cache go test -run '^$' -exec /usr/bin/true ./internal/runtimeidentity` | PASS（compile-only） |
| `cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go vet ./internal/runtimeidentity ./internal/config ./internal/command ./internal/provision` | PASS |
| `make check`（DEVで一回） | PASS |
| `cd tools/dev-agent-harness && ./configure && make distcheck` | PASS |
| `make candidate-commit TASK=TASK-0057`内のroot `make check` | PASS |
| `make lint-docs` | PASS |
| `git diff --check` | PASS |

## 主要な変更

- `config.Config.Identity.WorkspaceID`を追加し、1〜128 ASCII byte、先頭英数字、後続英数字/`._-`を検証する。
- `runtimeidentity.Resolver`がuser/group lookup、canonical uint32 ID、current broker EUID、agent primary group、16-byte entropyを一回の解決結果へまとめる。
- accessorはinvalid/corrupt receiverをzero valueへ閉じ、`Subject`を含む返却値をcopyする。`Format`はidentity値を出さない固定type名だけを返す。
- config example、command test fixture、provision direct-config validation、READMEを必須workspaceへ同期した。provision manifest/actionは変更していない。

## 検証結果

- `make check`: PASS
- focused Go tests、Linux compile-only、`go vet`、configured `make distcheck`、`make lint-docs`、`git diff --check`: PASS
- candidate差分: 許可path内11ファイル、追加＋削除約678行。外部dependency、config version、service compositionの変更なし。
- DEVは未生成Makefileで`distcheck` target不存在と誤判定した。Mainが同じ固定candidateで`./configure`後に既存`distcheck`を実行し、tarball内のconfigure/build/checkまでPASSした。製品差分は変更していない。

## 判断・既知の制約

- 実Linux NSS、別broker/agent UID/GID、sysusers、service restart、VPSは承認済み環境と安全なcleanupが未定のためblocked。fake seamとcross-compileをlive E2EのPASSに置換しない。
- socket activation、PeerBinder、brokerlistener Session/Exchangeとのservice compositionは後続Taskで一つのruntime identity resultをconsumeする。
