# Control Plane Schema catalog — draft-v0

Task責任、Contract、親子関係、Mailbox routing、Completion、および人間・外部Authorityとの唯一の通信境界であるAuthority Gatewayを所有する。Gatewayは認証、配送、期限、重複排除、Decision永続化を担うが、各Planeが所有する判断の意味やscopeは変更しない。Agent RunやCASB Rule本文は所有しない。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `task-contract.schema.json` | Objective、Acceptance、Instructions、Contract version |
| `task-command.schema.json` | delegate、Ask、Escalate、reply、cancel child、Completion Candidate |
| `task-event.schema.json` | Lifecycle、Contract change、Completion、Cancellation payload |
| `mailbox-event.schema.json` | 共通EnvelopeとAsync/Child/Ask/Escalation/Governance payload union |
| `mailbox-consumption.schema.json` | consumer、event sequence、watermark、冪等consume結果 |
| `completion-review-input.schema.json` | Reviewerへ渡す固定Candidate snapshot |
| `completion-review-output.schema.json` | accept / reject / insufficient evidence |
| `task-authority-request.schema.json` | Root Task EscalationをAuthorityへ提示するpayload |
| `task-authority-decision.schema.json` | Contract patch、terminate、responder provenance |
| `task-containment-command.schema.json` | Containment集合のTask別suspendと保存済み状態へのresume要求 |

## P1

| Schema | 固定する内容 |
|---|---|
| `task-progress.schema.json` | Todo ledger、watermark、version |
| `resume-context.schema.json` | Contract、Progress、Mailbox、過去Run Eventの再開snapshot |
| `suspension.schema.json` | source、error、retry policy、next retry |

## 現在のAPI adapter

Control Plane由来のWork Agent Tool、Acceptance Review出力、Task Escalation Decisionは`../api/`の合成bundleに含まれる。実装時は本directoryのcanonical Schemaから生成する。
