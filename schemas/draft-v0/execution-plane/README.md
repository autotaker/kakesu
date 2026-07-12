# Execution Plane Schema catalog — draft-v0

Agent Run、Function Call、Tool result、Async Operation、Continuation、Resource runtimeを所有する。Task ContractやSecurity Policyの意味判断は所有しない。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `tool-call.schema.json` | call ID、tool name、validated arguments、schema ref |
| `tool-result.schema.json` | completed / accepted / failed共通Envelope |
| `tool-result-values.schema.json` | Child Task、Ask、Escalation、Grant、Wait、Cancellation等の結果union |
| `async-operation.schema.json` | operation key、status、result/error ref、deadline |
| `agent-run-event.schema.json` | completed output item、tool call/output、compaction、maintenance event |
| `continuation.schema.json` | logical cursor、Wait Condition、resume watermark |
| `workspace-created.schema.json` | fork/empty、親Workspace、Policy binding、Grant非継承 |
| `resume-context.schema.json` | previous/new Run、Continuation、Mailbox、watermark |

## P1

| Schema | 固定する内容 |
|---|---|
| `agent-run-snapshot.schema.json` | Run再開に必要なExecution状態 |
| `agent-resource.schema.json` | process/server/worktree、lifetime、cleanup status |
| `workspace-runtime.schema.json` | runtime adapter、network identity、mount/resource refs |

## 現在のAPI adapter

Responses APIへ渡すWork Agent Tool bundleは`../api/work-agent-tools.json`にある。Function Call OutputはStructured Outputs対象外でも、本directoryの`tool-result*.schema.json`でHarness生成時と再配送時に検証する。
