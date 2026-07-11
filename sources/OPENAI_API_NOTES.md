# OpenAI Responses API仕様確認メモ

確認日: 2026-07-12

本設計はResponses APIの次の機能だけに依存する。API固有状態をTaskの正本にはしない。

## Responses API

公式API Reference:

- https://platform.openai.com/docs/api-reference/responses

確認事項:

- Custom function toolsを`tools`へ渡せる。
- `parallel_tool_calls`を設定できる。
- `previous_response_id`でマルチターン継続できる。
- `previous_response_id`と`conversation`は同時使用できない。
- top-level `instructions` parameterは現在のResponseのContextへsystemまたはdeveloper messageとして挿入される。
- `previous_response_id`を使っても、前Responseで指定したtop-level `instructions` parameterは次Responseへ引き継がれない。公式Referenceは、これにより新Responseでsystem/developer messageを差し替えやすいとしている。
- `input` itemとして渡せる`role: "developer"` messageと、top-level `instructions` parameterはAPI上の別の入力面である。
- `background`でResponseを非同期実行できる。

## Function Calling

公式Guide:

- https://developers.openai.com/api/docs/guides/function-calling

確認事項:

- Response outputの`function_call` itemは`call_id`、`name`、JSON encoded `arguments`を持つ。
- Tool実行結果は`function_call_output` itemとして同じ`call_id`へ対応付けられる。
- Responses APIのFunction Toolは`type: function`、`name`、`description`、JSON Schema `parameters`で定義できる。
- `strict: true`を利用できる。

### Function CallとStructured Text Outputの境界

公式Guide:

- https://developers.openai.com/api/docs/guides/structured-outputs
- https://developers.openai.com/api/docs/guides/function-calling

確認事項:

- Responses APIのStructured Outputsには二つの別経路がある。Application Toolへ接続する場合はFunction Calling、ユーザーへ返すテキスト応答をJSON Schemaへ制約する場合は`text.format`を使う。
- `text.format`はmessageとして生成されるtext outputの形式を制約する。Response全体の`output` item列を独自Envelopeへ変換する機能ではない。
- ModelがFunction Toolを選ぶと、Responseの`output` arrayには`type: "function_call"`の独立itemが入り、`name`、`call_id`、JSON encoded `arguments`を持つ。
- Function Call引数を構造化したい場合は、各Function Toolの`parameters` JSON Schemaと`strict: true`を使う。
- `tool_choice: "auto"`はmessage生成またはTool CallをModelに選ばせる。`tool_choice: "required"`は一つ以上のTool Callを要求する。

設計上の判断:

- `text.format`を使って、すべてのAgent Response Stepを`{ action, progress_delta }`のような共通Envelopeにすることはできない。
- Tool Callが出るStepにも別の構造化text outputが必ず出ることを前提にしない。
- 本設計ではHarnessが一定の通常Response Stepごとに別のMaintenance Responseを開始し、利用可能Toolを`update_progress`だけに限定したうえで`tool_choice`に同Functionを指定する。これはResponses APIによる自動付加ではなく、Harnessが明示的に追加するFunction Calling Stepである。
- Episode AgentではSQLite read-only `query_evidence` Function Tool、`tool_choice: "auto"`、Task Episode用`text.format: json_schema`を最初から同時指定する。ModelがEvidence不足時にFunction Call、十分な時にSchema準拠messageを返す通常Tool Loopとして扱い、Harness側で調査Phaseと最終生成Phaseを切り替えない。

## Conversation state

公式Guide:

- https://developers.openai.com/api/docs/guides/conversation-state

設計上の判断:

- `previous_response_id`は同一Agent Runの短期継続にだけ利用する。
- Response保存期間やContext課金、API側状態に依存せず、Task / Continuation / Workspaceを独自永続化する。

## Background mode

公式Guide:

- https://developers.openai.com/api/docs/guides/background

確認事項:

- `background: true`で長時間Responseを非同期開始できる。
- GET Responsesで`queued` / `in_progress`をpollできる。
- in-flight Responseをcancelできる。
- Background modeは単一推論の実行方式であり、Task schedulerやHarnessのAsync Operationを代替しない。

## Compaction

公式Guide:

- https://developers.openai.com/api/docs/guides/compaction

公式API Reference:

- https://developers.openai.com/api/reference/resources/responses/methods/compact

### Server-side compaction

- `POST /responses`または`client.responses.create`で、`context_management`に`type: "compaction"`と`compact_threshold`を指定できる。
- rendered token countが閾値を超えると、同じResponse処理内でserver-side compactionが走る。別の`/responses/compact`呼び出しは不要。
- Response streamには暗号化された`compaction` output itemが含まれる。
- compaction itemは過去の重要なstateとreasoningを少ないtokenで次のwindowへ運ぶが、opaqueであり、人間が解釈する用途ではない。
- stateless input-array chainingでは通常どおりResponse outputを次のinputへ追加する。最新compaction itemより前のitemは削除できる。
- `previous_response_id` chainingでは新しいuser messageだけを渡し、手動で過去itemをpruneしない。
- `store=false`を指定したserver-side compactionはZDR-friendlyと公式Guideに記載されている。

例:

```python
response = client.responses.create(
    model="<supported-model>",
    input=conversation,
    store=False,
    context_management=[
        {"type": "compaction", "compact_threshold": 200000}
    ],
)
```

### Standalone compact endpoint

- `POST /responses/compact`または`client.responses.compact`はstatelessで、ZDR-friendlyな明示的Compaction手段である。
- 呼び出し側がmessages、tools、reasoningやtool interactionを含む完全なContext Windowを`input`として渡す。入力はCompaction前の時点で対象モデルのContext Window内に収まる必要がある。
- 戻り値は次の`/responses`へ渡せる新しいcompacted context windowである。
- 出力はcompaction itemだけとは限らず、以前のwindowから保持されたitemを含みうる。
- `/responses/compact`の出力を呼び出し側でpruneしてはならない。返された`output`全体をそのまま次のResponses inputへ渡す。

```python
compacted = client.responses.compact(
    model="<supported-model>",
    input=long_input_items_array,
)

next_response = client.responses.create(
    model="<supported-model>",
    input=[*compacted.output, next_user_message],
    store=False,
)
```

### 設計上の判断

- Responses API CompactionはLLM Context Windowの圧縮機構であり、Task、Owner、Mailbox、Async Operation、Artifact、Workspaceの正本ではない。
- opaqueなcompaction itemをTask ProgressやHarnessのResume Cursorの代替にしない。
- server-side compactionは同一Agent Run内で利用できる。API Compactionが発生しただけではRunを閉じない。
- Contract変更、モデル変更、Response chain喪失、長時間停止などでHarnessが新Runを作る場合は、API Compactionとは別に最小Resume Cursorを作り、Task Progressを含む各正本を再読込する。
- standalone endpointを利用する場合、その出力全体をAPI向けの次Contextとして保持し、Harness固有のContractやMailbox Viewは次のinput構築時に明示的に加える。
