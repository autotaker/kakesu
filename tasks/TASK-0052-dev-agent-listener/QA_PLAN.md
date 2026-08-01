---
task_id: "TASK-0052"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T14:03:15Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0052 QA PLAN

## QA scope

期待値正本は TASK.md の `Planning input packet` だけとする。PLAN、実装案、DEV の自己申告から期待値を導かない。candidate の
`tools/dev-agent-harness/internal/brokerlistener/` と許可される README 差分を独立確認し、既存
`connectsession`、`brokerhttp`、Exchange/Transaction/Forwarder/Policy/capability/credential/transport、build/generated artifact を
変更していないことを確認する。

実 socket bind、SO_PEERCRED を含む OS peer credential、network namespace、systemd、real client/VPS は実環境、authority、
安全な cleanup が未定義であるため live-e2e を blocked とする。fake listener と `net.Pipe` の PASS は実到達制御又は OS identity を保証しない。

PeerBinder と Session は trusted かつ context に協調して return する注入依存である。任意の non-cooperative callback を Server が
強制停止することは対象外であり、QA は goroutine timeout による漏洩を許容する代替を要求又は PASS と主張しない。

## Cases

| Case ID | 対象AC | 確認内容と failure detection | Mode | Evidence |
|---|---|---|---|---|
| QA-001 | AC-1 | `New` が trusted cooperative PeerBinder と Session、および `1..64` の MaxConcurrent だけを受理して immutable Server を返すことを candidate source/test で監査する。nil/typed-nil dependency、範囲外値、nil/zero/corrupt Server、nil context/listener が panic せず fixed non-leak error になる test があり、dependency/subject/address/lower error を Format/error に露出しないことを確認する。long-lived state の追加、default binder/session、設定可能な公開 identity producer、又は test の削除・期待値緩和を failure として検出する。 | evidence-review | candidate source/test、HANDOVER、DEV command/result、candidate diff |
| QA-002 | AC-2 | slot を Accept 前に取得し、同時 Session 数が MaxConcurrent を超えないことを fake bounded listener と `net.Pipe` の race fixture で検出する。accepted conn ごとに binder が一回だけ Session 前に呼ばれ、UID 正数と既存 capability 同等の 1〜128 byte AgentInstanceID/WorkspaceID を検証・copy した root 由来 private context だけが Session へ一回渡ることを確認する。reject/invalid subject が Session 呼出、slot 不解放、conn 未 close、又は accept 停止になる実装を失敗検出する。 | focused-rerun | bundled package race test |
| QA-003 | AC-3 | Resolver が Server private key に束縛された subject の独立 copy だけを返し、missing/wrong-type/invalid/cancelled context を fixed error で拒否することを確認する。公開 setter、RemoteAddr/listener address/HTTP/CONNECT/TLS/header/caller 値からの生成又は補完、並行 connection 間の subject/context 共有を negative fixture で失敗検出する。 | focused-rerun | bundled package race test |
| QA-004 | AC-4 | binder/session error 又は panic が当該 conn の close と slot release だけとなり、次の accept を継続することを確認する。unexpected Accept error が listener close、全 connection context cancel、cooperative drain 後の fixed server error となり、retry/backoff、別 listener、default dependency、診断 log、dependency detail の返却を導入しないことを failure-detection fixture と source audit で確認する。 | focused-rerun | bundled package race test |
| QA-005 | AC-5 | caller cancel/deadline が listener close により Accept を解除し、全 connection context へ伝播し、cooperative binder/Session return 後に nil で drain することを確認する。normal cancel、accept failure、per-connection error/panic で Server watcher/goroutine、accepted conn、listener が残ること、又は callback を別 goroutine へ逃がす timeout leak を failure として検出する。 | focused-rerun | bundled package race test |
| QA-006 | AC-6 | hermetic race test が private Resolver、slot-before-Accept、binder-before-session、subject validation/copy、multiple connection identity isolation、binder/session panic/error の per-conn isolation、unexpected Accept error、caller cancel/deadline と drain、listener/conn close を実際に失敗検出できることを source/evidence audit する。candidate-bound root/harness check、distcheck、README 変更時 lint、candidate launcher の root `make check` は DEV 証跡を監査するだけで、QA は再実行しない。差分が許可 path、追加＋削除 1,000 行以下、目標 800〜900 行に収まり、test を弱体化していないことも確認する。 | evidence-review | candidate source/test、HANDOVER、DEV command/result、candidate diff |
| QA-007 | 対象外 / AC-6 の制限 | actual socket bind、SO_PEERCRED、network namespace、systemd socket activation、real client/VPS を実環境で検証する。 | live-e2e — blocked | 実環境の authority、OS identity path、real client/VPS と安全な cleanup が未定義。この blocked は hermetic PASS で代替しない。 |

## Execution rule

QA-002〜005 は同じ一回の focused-rerun 観測に束ねる。QA は `tools/dev-agent-harness` を cwd として、candidate に対し次だけを一回実行する。

```sh
GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/brokerlistener
```

QA は root/harness `make check`、`make distcheck`、lint、追加 test 又は rerun を実行しない。QA-001 と QA-006 は candidate source/test、HANDOVER、DEV command/result の独立 evidence audit だけを行う。package race test が指定した negative/boundary の失敗を検出できない、candidate/tree と証跡が一致しない、又は fixture が hermetic/deterministic/bounded でない場合、該当 focused-rerun は blocked 又は FAIL であり evidence-review PASS に置換しない。

## Result criteria

各 case を planning input packet と candidate-bound evidence に照らして記録する。focused-rerun では command、cwd/cache、exit status、実行 test、coverage と failure-detection evidence を残す。source/evidence audit は private context key、slot-before-Accept、binder-before-session、subject validation/copy、per-connection panic/error isolation、unexpected Accept error、caller cancel/drain、no timeout leak を明示的に確認する。

失敗を実装不具合と決めつけず、`implementation_defect`、`qa_plan_defect`、`requirement_gap`、`environment_issue`、`regression`、又は evidence 不足として根拠付きで分類する。QA-007 は安全に実行可能となるまで blocked のままとする。

## 実装後の再確認

- [ ] candidate source/test、HANDOVER、DEV check evidence を独立確認する。
- [ ] 指定 package race test を candidate で一回だけ実行する。
- [ ] private key/binding order/subject copy/isolation、panic/error isolation、accept failure、cancel/drain、no timeout leak の failure detection を確認する。
- [ ] 変更が許可 path と追加＋削除 1,000 行以内に収まり、既存 dependency/package の意味を変えていないことを確認する。
- [ ] real socket/OS credential/network namespace/systemd/real client/VPS live-e2e を PASS に置換せず、期待値又は scope を変更していないことを確認する。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-01 | qa-agent-terra-medium | Planning input packet に基づく独立 QA 計画 | `approved` |
