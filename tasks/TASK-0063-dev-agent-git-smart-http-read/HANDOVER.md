---
task_id: "TASK-0063"
status: complete
completed_at: "2026-08-02T01:51:58Z"
candidate_commit: "c6ff1c0c10df12bb85ad4fe8090fdb196a943e6d"
---

# TASK-0063 HANDOVER

## 成果

- allowlist内repositoryに対するcanonical Git Smart HTTP upload-pack discovery/POSTだけを、新しいGit read scopeで許可した。
- peer-bound controlが明示Git read selectorから5分・一回のOpaque capabilityを発行し、AgentのHTTP Basic handleを同じRegistryで消費後に実GitHub tokenのHTTP Basicへ置換する。
- `github.com`をCONNECT、CA、inner mapping、forwarder、pinned transportへ閉じて追加し、receive-pack/push、redirect/retry、URL・HTTP・response逸脱を拒否する。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `make check`（candidate固定直前の最終実行） | `PASS` |
| affected 9 packageのfocused `go test -race` | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- 承認済み20パス、1,008 additions / 79 deletions。helper/launcher/Approval、listener/socket/unit、依存、Schema、Kakesu runtime、live stateの差分はない。
- GitHub REST/OpenAIの既存authorization、JSON response、host/transport契約は既存testsと追加回帰testsで維持した。

## 検証結果

- `make check`: `PASS`
- `cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/egresspolicy ./internal/capability ./internal/capabilitycontrol ./internal/connectsession ./internal/proxyca ./internal/brokerhttp ./internal/egresstransaction ./internal/upstreamforwarder ./internal/upstreamtransport`: `PASS`
- terminology検査、docs lint、`git diff --check`: `PASS`

## 判断・既知の制約

- 初回candidate gateはREADMEの未登録英語`binary`頻度で停止し、日本語表記へ整理した。次の実行はREADMEの既登録用語7件で停止し、用語集やruleを増やさず日本語へ修正した。いずれもcandidate commit作成前のdocs lint failureであり、最終gateはPASSした。
- 最終gateの呼出しは30秒yield後も内部で継続してcandidateを正常固定した。重複呼出しは`candidate branch must contain only the planning commit`で変更前に停止し、追加commitや差分変更はない。
- 初回candidateの独立Reviewで、policyが許可する明示`:443`をpinned transportが拒否するF-1を検出した。Task branchをplanning commitへmixed resetして製品差分を保持し、許可済みtransport 2パスだけでdefault portを正規hostへ変換し、3許可hostの`:443` positive testと`:444`/near-match拒否を追加した。影響5 package raceと再固定時のroot `make check`はPASSした。
- 実GitHub credential/DNS/TLS、実Git client、別UID/NSS、systemd/VPSは未確認であり、hermetic PASSからlive事実を主張しない。
