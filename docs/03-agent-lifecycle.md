# Agentライフサイクル設計

## 1. 対象

本書は、持続的な論理主体であるAgentが生成され、TaskのOwnerへ割り当てられ、Agent Runを通じて実行し、Task終了後に解放されるまでを定義する。

Taskの状態遷移は[02-task-lifecycle.md](02-task-lifecycle.md)、1回ごとの推論実行とResponses APIの対応は[05-runtime-and-responses-api.md](05-runtime-and-responses-api.md)を正本とする。

```text
Agent lifecycle     = 論理的な作業主体の寿命
Agent Run lifecycle = 一つの実行セッションの寿命
Task lifecycle      = Ownerが負う完了責任の寿命
```

三者のIDと状態を同一視しない。

## 2. Agentの状態

```typescript
type AgentStatus = "idle" | "assigned" | "retired";
```

| 状態 | 意味 |
|---|---|
| `idle` | 非終端Taskを所有せず、割当可能 |
| `assigned` | 一つの非終端TaskのOwnerである |
| `retired` | 新しいTaskへ割り当てない終端状態 |

AgentはLLM Response、OSプロセス、Agent Runとは異なる。Runが停止・失敗しても、AgentとOwnerの責任が直ちに消滅するわけではない。

## 3. 生成と登録

Root AgentはRoot Task生成時に、SubAgentは`delegate`によるChild Task生成時に、HarnessのAgent Registryへ登録される。

登録時に最低限、次を固定する。

- `agent_id`
- `profile_id`
- `level`
- 使用可能なToolとPolicy View
- 親子Taskから導出される権限境界

Agentを生成しただけでは作業を開始しない。Task、Owner割当、Workspaceの準備が成立してからAgent Runを開始する。

## 4. Owner割当

HarnessはAgentをTaskの単一Ownerとして割り当てる。

```text
idle Agent
  → Owner排他を検査
  → Task.owner_agent_idを設定
  → Agent.current_task_idを設定
  → Agentをassignedへ遷移
```

1つのAgentは同時に1つの非終端Taskだけを所有する。Taskが`waiting`でもOwner責任は継続するため、Agentは`assigned`のままである。待機対象は`WaitCondition.kind`で区別する。

## 5. Agent Run開始

HarnessはTask実行のためにAgent Runを生成する。

```typescript
type AgentRunStatus = "running" | "stopped" | "failed" | "completed";
```

Run開始時には次を行う。

1. AgentとTaskのOwner対応を検証する
2. Task Contractと現在状態を読む
3. Workspaceを生成または復元する
4. Memory Contextを取得する
5. MailboxとContinuationからContext Viewを構築する
6. 新しい`run_id`で推論を開始する

1つのTaskに複数のAgent Runを許すが、同じTaskへ同時に複数のResponse Stepを走らせない。

## 6. 実行・中断・再開

Agent RunはActionをyieldし、結果を同期出力またはMailboxで受けて再開する。

```text
running
  → Actionをyield
  → 続行可能なら同じRunで次Step
  → 外部イベントが必須ならContinuationを保存してstopped
  → イベント到着後に同じAgentの新規または継続Runで再開
```

Taskが`waiting`へ入っても、Agentが`idle`へ戻るわけではない。待機はTask実行の一時停止であり、Owner解放ではない。

次の場合は同じAgent・同じTaskに新しいRunを作れる。

- プロセス再起動
- 長時間停止後の復帰
- Context圧縮または再構築
- モデル切替
- `previous_response_id`喪失
- 先行Runの回復可能な失敗

Responses APIのserver-side compactionは同じRun内で利用できる。HarnessがResume Cursorによる再構築を選んだ場合だけ、TaskやOwner Assignmentを変更せず、現在Runを`stopped`として閉じ、同じAgentの新しいRunを開始する。意味的な再開情報はTask Progress Ledgerから取得する。二層の具体的な境界は[05-runtime-and-responses-api.md](05-runtime-and-responses-api.md)を正本とする。

## 7. AskとEscalation中のAgent

AskはOwner Agent間の助言通信である。Child Ownerは判断責任を保持し、回答後も同じTaskのOwnerとして判断を続ける。

Escalationは親子Task間の判断責任移転である。Agentは各TaskのOwnerとしてメッセージを送受信する。親の決定でChild Task Contractが更新されても、原則としてChild AgentのOwner割当は維持され、新しいContract バージョンでRunを再開する。

`terminate: true`またはTaskの終端遷移が確定した場合だけ、終了処理へ進む。

## 8. Run終了と失敗

Agent Runの終了はTaskの終了を意味しない。

Agent Runから発生するリクエスト、レスポンス item、Tool Call、usage、エラーの保存範囲は[05-runtime-and-responses-api.md](05-runtime-and-responses-api.md)の「Agent Run Record Policy」を正本とする。

| Run結果 | Taskへの作用 |
|---|---|
| `completed` | そのRunを閉じる。TaskはAction結果に応じて継続または終端する |
| `stopped` | Continuationを保存し、再開可能にする |
| `failed` | Runのエラーを記録し、Taskを`suspended`へ遷移してHarnessが再試行または新Runを判断する |

LLM API障害やプロセス障害だけでOwnerを解放してはならない。Taskの責任が残る限り、Agentは`assigned`を維持する。

## 9. Owner解放

Taskが`completed`または`cancelled`へ確定した後、HarnessがOwnerを解放する。`suspended`では責任が残るため解放しない。

```text
Task terminal確定
  → 実行中Runを閉じる
  → 未処理OperationとMailboxの扱いを確定
  → 論理WorkspaceのOutcome参照を確定
  → Task Episode生成を要求
  → Agent.current_task_idを解除
  → Agentをidleまたはretiredへ遷移
  → assignment scopeのAgent Resource Cleanupを起動
```

Owner解放とTask終端は同一Transaction、または再実行可能な終端処理として整合させる。Resource Cleanup完了はOwner解放の条件にしない。終端Taskを所有したままのAgentや、非終端TaskのOwnerが`idle`になる状態を残さない。

## 10. Agent Resource

プロセス、サーバー、worktree、一時ディレクトリなどの物理実行資源はTaskではなくAgentへ紐づける。Taskの論理Workspace、Artifact、Evidenceとは分離する。

```typescript
type AgentResource = {
  resource_id: string;
  agent_id: string;
  assignment_id?: string;
  run_id?: string;
  kind: "process" | "server" | "worktree" | "temporary_directory";
  resource_ref: string;
  lifetime: "run" | "assignment" | "agent";
  cleanup_policy: "stop" | "delete" | "retain";
  status:
    | "active"
    | "cleanup_pending"
    | "cleaning"
    | "released"
    | "needs_operator";
};
```

AgentまたはTool実行基盤は生成したResourceをHarnessへ登録する。`assignment` スコープでは再利用されたAgentの新しいResourceと混同しないよう`assignment_id`を必須にする。Harnessは`run`、`assignment`、`agent`の各境界でCleanupを開始し、専用Managerによる停止・削除、冪等な再試行、Operator通知を追跡する。Cleanup失敗はTaskを`suspended`へ戻さず、Resourceを`needs_operator`にする。

論理Workspaceのスナップショット、Artifact、Evidence固定はTask Outcomeの責務であり、固定内容はEvidence DBへ取り込む。一方、Git worktreeやcontainer volumeなどの物理実体はAgent Resourceであり、DB取込とOutcome参照の固定後に削除できる。

## 11. Retire

Agentを今後再利用しない場合は`retired`へ遷移する。

- 非終端Taskを所有するAgentは直接retireできない
- 先にTaskを完了・キャンセルするか、明示的なOwner移管手続きを完了する
- 過去のTask、Run、EpisodeからAgentへの参照は保持する

初期実装でOwner移管を提供しない場合、非終端TaskのOwner Agentは交換せず、同じ論理Agentに新しいRunを生成して障害復旧する。

## 12. 不変条件

1. 1つの非終端Taskには一人のOwner Agentがいる。
2. 1つのAgentは同時に1つの非終端Taskだけを所有する。
3. `waiting`中もAgentのOwner割当は継続する。
4. Agent Runの終了・失敗だけではOwnerを解放しない。
5. 同じTaskの複数Runは許すが、同時に複数のResponse Stepを実行しない。
6. Task終端後はAgentを`idle`または`retired`へ収束させる。
7. AskはAgent間通信であり、EscalationはTask間の責任移転である。
8. 物理実行ResourceはAgentへ登録し、CleanupはHarnessが所有する。
9. Resource Cleanupの成否は終端Taskの状態を変更しない。
