# OpenAI Responses API仕様確認メモ

確認日: 2026-07-11

本設計はResponses APIの次の機能だけに依存する。API固有状態をTaskの正本にはしない。

## Responses API

公式API Reference:

- https://platform.openai.com/docs/api-reference/responses

確認事項:

- Custom function toolsを`tools`へ渡せる。
- `parallel_tool_calls`を設定できる。
- `previous_response_id`でマルチターン継続できる。
- `previous_response_id`と`conversation`は同時使用できない。
- `previous_response_id`を使っても以前の`instructions`は次Responseへ自動継承されない。
- `background`でResponseを非同期実行できる。

## Function Calling

公式Guide:

- https://developers.openai.com/api/docs/guides/function-calling

確認事項:

- Response outputの`function_call` itemは`call_id`、`name`、JSON encoded `arguments`を持つ。
- Tool実行結果は`function_call_output` itemとして同じ`call_id`へ対応付けられる。
- Responses APIのFunction Toolは`type: function`、`name`、`description`、JSON Schema `parameters`で定義できる。
- `strict: true`を利用できる。

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
