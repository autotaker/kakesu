---
task_id: "TASK-0070"
status: complete
completed_at: "2026-08-02T06:40:42Z"
candidate_commit: "3f74f26af52233386b4c04cdfa30920309825c97"
---

# TASK-0070 HANDOVER

## 成果

- 構造化されたpush proposalを厳格に検査し、順序付き全参照更新を含む正規JSONとdomain-separated SHA-256を生成するpure Go値境界を追加した。
- 自身の正規encodingだけをparseし、重複/未知/欠落キー、非正規表現、digest改変、過大入力を拒否する。
- 入出力sliceの所有権を分離し、manifestの妥当性が承認・grant・push許可を意味しない後続境界をREADMEへ固定した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate gateのroot `make check` | `PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/approvalmanifest` | `PASS` |
| harness `make check` | `PASS` |
| harness `make distcheck` | `PASS` |
| candidate worktree `make lint-docs` | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- 承認済み3パス、1,134 additions / 0 deletions。
- `Build`はID/identity、repository/remote、1〜32件のbranch参照更新、policy/epoch、UTC秒時刻をshared validatorで検査し、caller supplied digest、clock、randomness、I/Oを持たない。
- fixed-order payloadへv1 domain prefixを付けた実SHA-256を計算し、derived `request_digest`を最後に持つcompact JSONを生成する。参照の入力順もdigest対象である。
- `Parse`は32,768-byte上限、duplicate-aware走査、unknown拒否typed decode、shared validation、constant-time digest照合、byte-identical reencodeを全て通った入力だけを受理する。
- immutable getterはslice/bytesのcopyを返し、公開errorは固定class/field/ref indexだけを返す。

## 検証結果

- candidate gate root `make check`: `PASS`
- 最終candidate bytesへのfocused race、harness check/distcheck、docs lint、diff check: `PASS`

## 判断・既知の制約

- 初回candidate gateはREADMEの新規22行に対する既存用語lintで停止した。新規節だけを簡潔な日本語へ修正し、docs lintとfocused raceを再確認した最終bytesからcandidateを固定した。製品code/testのFAILではない。
- 初回QAはfocused race自体をPASSしたが、delete bit単独のdigest感度と全負例のnon-leak走査が不足しているとして証跡FAILにした。製品実装を変えずtestへ限定して両検出能力を追加し、focused raceとcandidate gateを再実行した新candidateへ固定した。
- manifest constructorは時刻、request ID、policy、identityを生成又は信頼判定しない。現在時刻、TTL、ID一意性、policy妥当性は後続state storeが所有する。
- raw Git receive-pack/pkt-line解析、remote旧SHA観測、force推定、Approval state、Tailscale、Passkey、push grant、実credential/実pushは未実装・未確認であり、このcandidateのPASSで代替しない。
