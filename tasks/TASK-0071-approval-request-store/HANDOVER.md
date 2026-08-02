---
task_id: "TASK-0071"
status: complete
completed_at: "2026-08-02T07:39:03Z"
candidate_commit: "dc38a17a01223c49af025375366b0d781b1302fa"
---

# TASK-0071 HANDOVER

## 成果

- canonical push承認manifestをowner-only directoryへ永続化するsingle-writer request storeを追加した。
- `pending/approved/denied/cancelled/expired/stale`の期限優先状態機械、restart validation、clock rollback拒否を実装した。
- temp write/sync/atomic replace/directory syncとphase別failure処理を持ち、置換結果が不確実な場合はstoreをpoisonして成功を推測しない。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate gateのroot `make check` | `PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/approvalstate` | `PASS` |
| Linux/Windows cross-compile | `PASS` |
| isolated configured harness `make check` / `make distcheck` | `PASS` |
| candidate worktree docs lint | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- 承認済み5パス、1,414 additions / 0 deletions。
- `Open`は既存0700 rootをdescriptorへ固定し、`os.Root`配下のfixed state/lock/temp name、current owner、Linux/Darwinのnon-blocking exclusive lockを検査する。二重writer、root又はnodeの差し替え/symlink、wrong mode/type、残存tempを拒否する。
- `Create`はTASK-0070 parserを再利用し、policy/epoch、trusted clock、TTL、capacity、request ID一意性を検査してpending snapshotを作る。
- mutationはrequest IDとconstant-time digest一致を要求し、期限到達をdecisionより先に永続化する。record/snapshot getterはdeep copyを返し、fixed errorは入力値又はOS errorを含めない。
- snapshotはgeneration、observed time、request ID順recordsをcanonical JSONで持ち、Open時にmanifest/digest/state/time/sort/shapeを全再検証する。

## 検証結果

- candidate gate root `make check`: `PASS`
- 最終candidate codeへのfocused race、cross-compile、restart/corruption/failure/expiry/concurrency fixture: `PASS`
- 最終READMEへのdocs lint、harness check/distcheck、diff check: `PASS`

## 判断・既知の制約

- 初回DEV focused suite後のMain事前監査でCreate負例、symlink/strict snapshot、approved expiry、non-leak、Close競合の証跡不足を検出した。candidate固定前に既存fixtureを表駆動で補強し、focused raceを再実行した。
- 初回candidate gateはREADME新規節の既存用語lintだけで停止した。個別lintと集計lintを分けて新規20行だけを日本語化し、final candidate gateをPASSした。製品code/testのFAILではない。
- 初回candidate `ceb57a6`の独立Reviewで、root descriptor検証後に文字列pathへ戻るためroot rename/symlink差し替え競合を防げないblocking defectを検出した。state/temp/lock操作を検証済み`os.Root`へ束縛し、元directoryだけが更新されるroot replacement fixtureを追加してcandidateを再固定した。
- 初回QAでpartial/oversize snapshot拒否とrename前failure時のmemory/disk不変性に直接検出証跡がないことを検出した。各fixtureを追加し、新candidateでfocused raceを再実行した。
- 実UID/`/var/lib`、実power-loss durability、systemd配置/restart/rollback、Tailscale/Passkey、verified actorを限定するprocess境界、grant、実pushは未確認であり、hermetic PASSで代替しない。
- rename前failureは旧memory/diskを維持する。rename失敗又はdirectory sync失敗はstoreをpoisonし、Close→Openと上位reconciliationを要求する。
