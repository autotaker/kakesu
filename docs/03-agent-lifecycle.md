# Agentライフサイクル設計

## 1. 対象

本書は、持続的な論理主体であるAgentが生成され、Taskのオーナーへ割り当てられ、Agent実行を通じて実行し、Task終了後に解放されるまでを定義する。

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
| `assigned` | 一つの非終端Taskのオーナーである |
| `retired` | 新しいTaskへ割り当てない終端状態 |

AgentはLLM レスポンス、OSプロセス、Agent実行とは異なる。実行が停止・失敗しても、Agentとオーナーの責任が直ちに消滅するわけではない。

## 3. 生成と登録

ルート AgentはルートTask生成時に、SubAgentは`delegate`による子Task生成時に、ハーネスのAgent レジストリへ登録される。

登録時に最低限、次を固定する。

- `agent_id`
- `profile_id`
- `level`
- 使用可能なツールとポリシー ビュー
- 親子Taskから導出される権限境界

Agentを生成しただけでは作業を開始しない。Task、オーナー割当、Workspaceの準備が成立してからAgent実行を開始する。

## 4. オーナー割当

ハーネスはAgentをTaskの単一オーナーとして割り当てる。

```text
idle Agent
  → Owner排他を検査
  → Task.owner_agent_idを設定
  → Agent.current_task_idを設定
  → Agentをassignedへ遷移
```

1つのAgentは同時に1つの非終端Taskだけを所有する。Taskが`waiting`でもオーナー責任は継続するため、Agentは`assigned`のままである。待機対象は`WaitCondition.kind`で区別する。

## 5. Agent実行開始

ハーネスはTask実行のためにAgent実行を生成する。

```typescript
type AgentRunStatus = "running" | "stopped" | "failed" | "completed";
```

実行開始時には次を行う。

1. AgentとTaskのオーナー対応を検証する
2. Task契約と現在状態を読む
3. Workspaceを生成または復元する
4. 記憶コンテキストを取得する
5. メールボックスと継続情報からコンテキスト ビューを構築する
6. 新しい`run_id`で推論を開始する

1つのTaskに複数のAgent実行を許すが、同じTaskへ同時に複数のレスポンス ステップを走らせない。

## 6. 実行・中断・再開

Agent実行は操作をyieldし、結果を同期出力またはメールボックスで受けて再開する。

```text
running
  → Actionをyield
  → 続行可能なら同じRunで次Step
  → 外部イベントが必須ならContinuationを保存してstopped
  → イベント到着後に同じAgentの新規または継続Runで再開
```

Taskが`waiting`へ入っても、Agentが`idle`へ戻るわけではない。待機はTask実行の一時停止であり、オーナー解放ではない。

次の場合は同じAgent・同じTaskに新しい実行を作れる。

- プロセス再起動
- 長時間停止後の復帰
- コンテキスト圧縮または再構築
- モデル切替
- `previous_response_id`喪失
- 先行実行の回復可能な失敗

Responses APIのサーバー側 圧縮は同じ実行内で利用できる。ハーネスが再開 カーソルによる再構築を選んだ場合だけ、Taskやオーナー 割り当てを変更せず、現在実行を`stopped`として閉じ、同じAgentの新しい実行を開始する。意味的な再開情報はTask進捗 台帳から取得する。二層の具体的な境界は[05-runtime-and-responses-api.md](05-runtime-and-responses-api.md)を正本とする。

## 7. 質問と上位判断依頼中のAgent

質問はオーナーAgent間の助言通信である。子 オーナーは判断責任を保持し、回答後も同じTaskのオーナーとして判断を続ける。

上位判断依頼は親子Task間の判断責任移転である。Agentは各Taskのオーナーとしてメッセージを送受信する。親の決定で子Task契約が更新されても、原則として子 Agentのオーナー割当は維持され、新しい契約バージョンで実行を再開する。

`terminate: true`またはTaskの終端遷移が確定した場合だけ、終了処理へ進む。

## 8. 実行終了と失敗

Agent実行の終了はTaskの終了を意味しない。

Agent実行から発生するリクエスト、レスポンス 項目、ツール 呼び出し、usage、エラーの保存範囲は[05-runtime-and-responses-api.md](05-runtime-and-responses-api.md)の「Agent実行記録 ポリシー」を正本とする。

| 実行結果 | Taskへの作用 |
|---|---|
| `completed` | その実行を閉じる。Taskは操作結果に応じて継続または終端する |
| `stopped` | 継続情報を保存し、再開可能にする |
| `failed` | 実行のエラーを記録し、Taskを`suspended`へ遷移してハーネスが再試行または新実行を判断する |

LLM API障害やプロセス障害だけでオーナーを解放してはならない。Taskの責任が残る限り、Agentは`assigned`を維持する。

## 9. オーナー解放

Taskが`completed`または`cancelled`へ確定した後、ハーネスがオーナーを解放する。`suspended`では責任が残るため解放しない。

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

オーナー解放とTask終端は同一トランザクション、または再実行可能な終端処理として整合させる。リソース クリーンアップ完了はオーナー解放の条件にしない。終端Taskを所有したままのAgentや、非終端Taskのオーナーが`idle`になる状態を残さない。

## 10. Agentリソース

プロセス、サーバー、ワークツリー、一時ディレクトリなどの物理実行資源はTaskではなくAgentへ紐づける。Taskの論理Workspace、成果物、証跡とは分離する。

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

Agentまたはツール実行基盤は生成したリソースをハーネスへ登録する。`assignment` スコープでは再利用されたAgentの新しいリソースと混同しないよう`assignment_id`を必須にする。ハーネスは`run`、`assignment`、`agent`の各境界でクリーンアップを開始し、専用マネージャーによる停止・削除、冪等な再試行、運用者通知を追跡する。クリーンアップ失敗はTaskを`suspended`へ戻さず、リソースを`needs_operator`にする。

論理Workspaceのスナップショット、成果物、証跡固定はTask 結果の責務であり、固定内容は証跡DBへ取り込む。一方、Git ワークツリーやコンテナー ボリュームなどの物理実体はAgentリソースであり、DB取込と結果参照の固定後に削除できる。

## 11. Retire

Agentを今後再利用しない場合は`retired`へ遷移する。

- 非終端Taskを所有するAgentは直接retireできない
- 先にTaskを完了・キャンセルするか、明示的なオーナー移管手続きを完了する
- 過去のTask、実行、エピソードからAgentへの参照は保持する

初期実装でオーナー移管を提供しない場合、非終端TaskのオーナーAgentは交換せず、同じ論理Agentに新しい実行を生成して障害復旧する。

## 12. 不変条件

1. 1つの非終端Taskには一人のオーナーAgentがいる。
2. 1つのAgentは同時に1つの非終端Taskだけを所有する。
3. `waiting`中もAgentのオーナー割当は継続する。
4. Agent実行の終了・失敗だけではオーナーを解放しない。
5. 同じTaskの複数実行は許すが、同時に複数のレスポンス ステップを実行しない。
6. Task終端後はAgentを`idle`または`retired`へ収束させる。
7. 質問はAgent間通信であり、上位判断依頼はTask間の責任移転である。
8. 物理実行リソースはAgentへ登録し、クリーンアップはハーネスが所有する。
9. リソース クリーンアップの成否は終端Taskの状態を変更しない。
