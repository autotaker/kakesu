# OpenAI Responses API仕様確認メモ

確認日: 2026-07-12。

本設計はResponses APIの次の機能だけに依存する。API固有状態をTaskの正本にはしない。

## Responses API

公式API 参照を確認した。

- https://platform.openai.com/docs/api-reference/responses

確認事項を示す。

- Custom 関数 ツールを`tools`へ渡せる。
- `parallel_tool_calls`を設定できる。
- `previous_response_id`でマルチターン継続できる。
- `previous_response_id`と`conversation`は同時使用できない。
- 最上位 `instructions` パラメーターは現在のレスポンスのコンテキストへシステムまたは開発者 メッセージとして挿入される。
- `previous_response_id`を使っても、前レスポンスで指定した最上位 `instructions` パラメーターは次レスポンスへ引き継がれない。公式参照は、これにより新レスポンスでシステム/開発者 メッセージを差し替えやすいとしている。
- `input` 項目として渡せる`role: "developer"` メッセージと、最上位 `instructions` パラメーターはAPI上の別の入力面である。
- `background`でレスポンスを非同期実行できる。

## 関数 呼び出し

公式ガイドを確認した。

- https://developers.openai.com/api/docs/guides/function-calling

確認事項を示す。

- レスポンス 出力の`function_call` 項目は`call_id`、`name`、JSON encoded `arguments`を持つ。
- ツール実行結果は`function_call_output` 項目として同じ`call_id`へ対応付けられる。
- Responses APIの関数 ツールは`type: function`、`name`、`description`、JSON Schema `parameters`で定義できる。
- `strict: true`を利用できる。

### 関数 呼び出しと構造化 テキスト 出力の境界

公式ガイドを確認した。

- https://developers.openai.com/api/docs/guides/structured-outputs
- https://developers.openai.com/api/docs/guides/function-calling

確認事項を示す。

- Responses APIの構造化出力には2つの別経路がある。アプリケーション ツールへ接続する場合は関数 呼び出し、ユーザーへ返すテキスト応答をJSON Schemaへ制約する場合は`text.format`を使う。
- `text.format`はメッセージとして生成されるテキスト 出力の形式を制約する。レスポンス全体の`output` 項目列を独自共通形式へ変換する機能ではない。
- モデルが関数 ツールを選ぶと、レスポンスの`output` 配列には`type: "function_call"`の独立項目が入り、`name`、`call_id`、JSON encoded `arguments`を持つ。
- 関数 呼び出し引数を構造化したい場合は、各関数 ツールの`parameters` JSON Schemaと`strict: true`を使う。
- `tool_choice: "auto"`はメッセージ生成またはツール 呼び出しをモデルに選ばせる。`tool_choice: "required"`は1つ以上のツール 呼び出しを要求する。

設計上の判断を示す。

- `text.format`を使って、すべてのAgent レスポンス ステップを`{ action, progress_delta }`のような共通共通形式にはできない。
- ツール 呼び出しが出るステップにも別の構造化テキスト 出力が必ず出ることを前提にしない。
- 本設計ではハーネスが一定の通常レスポンス ステップごとに別のメンテナンス レスポンスを開始し、利用可能ツールを`update_progress`だけに限定したうえで`tool_choice`に同関数を指定する。これはResponses APIによる自動付加ではなく、ハーネスが明示的に追加する関数 呼び出し ステップである。
- エピソード AgentではSQLite 読み取り専用 `query_evidence` 関数 ツール、`tool_choice: "auto"`、Taskエピソード用`text.format: json_schema`を最初から同時指定する。モデルが証跡不足時に関数 呼び出し、十分な時にSchema準拠メッセージを返す通常ツール ループとして扱い、ハーネス側で調査フェーズと最終生成フェーズを切り替えない。

## Conversation 状態

公式ガイドを確認した。

- https://developers.openai.com/api/docs/guides/conversation-state

設計上の判断を示す。

- `previous_response_id`は同一Agent実行の短期継続にだけ利用する。
- レスポンス保存期間やコンテキスト課金、API側状態に依存せず、Task / 継続情報 / Workspaceを独自永続化する。

## バックグラウンド モード

公式ガイドを確認した。

- https://developers.openai.com/api/docs/guides/background

確認事項を示す。

- `background: true`で長時間レスポンスを非同期開始できる。
- GET Responsesで`queued` / `in_progress`をpollできる。
- in-flight レスポンスをキャンセルできる。
- バックグラウンド モードは単一推論の実行方式であり、Task schedulerやハーネスの非同期操作を代替しない。

## 圧縮

公式ガイドを確認した。

- https://developers.openai.com/api/docs/guides/compaction

公式API 参照を確認した。

- https://developers.openai.com/api/reference/resources/responses/methods/compact

### サーバー側 圧縮

- `POST /responses`または`client.responses.create`で、`context_management`に`type: "compaction"`と`compact_threshold`を指定できる。
- rendered トークン 回数が閾値を超えると、同じレスポンス処理内でサーバー側 圧縮が走る。別の`/responses/compact`呼び出しは不要。
- レスポンス ストリームには暗号化された`compaction` 出力 項目が含まれる。
- 圧縮 項目は過去の重要な状態と推論を少ないトークンで次のウィンドウへ運ぶが、不透明であり、人間が解釈する用途ではない。
- ステートレス input-array 連結では通常どおりレスポンス 出力を次の入力へ追加する。最新圧縮 項目より前の項目は削除できる。
- `previous_response_id` 連結では新しいuser メッセージだけを渡し、手動で過去項目を枝刈りしない。
- `store=false`を指定したサーバー側 圧縮はZDR-friendlyと公式ガイドに記載されている。

例を示す。

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

- `POST /responses/compact`または`client.responses.compact`はステートレスで、ZDR-friendlyな明示的圧縮手段である。
- 呼び出し側がmessages、ツール、推論やツール interactionを含む完全なコンテキスト ウィンドウを`input`として渡す。入力は圧縮前の時点で対象モデルのコンテキスト ウィンドウ内に収まる必要がある。
- 戻り値は次の`/responses`へ渡せる新しいcompacted コンテキスト ウィンドウである。
- 出力は圧縮 項目だけとは限らず、以前のウィンドウから保持された項目を含みうる。
- `/responses/compact`の出力を呼び出し側で枝刈りしてはならない。返された`output`全体をそのまま次のResponses 入力へ渡す。

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

- Responses API 圧縮はLLM コンテキスト ウィンドウの圧縮機構であり、Task、オーナー、メールボックス、非同期操作、成果物、Workspaceの正本ではない。
- 不透明な圧縮 項目をTask進捗やハーネスの再開 カーソルの代替にしない。
- サーバー側 圧縮は同一Agent実行内で利用できる。API 圧縮が発生しただけでは実行を閉じない。
- 契約変更、モデル変更、レスポンス 連鎖喪失、長時間停止などでハーネスが新実行を作る場合は、API 圧縮とは別に最小再開 カーソルを作り、Task進捗を含む各正本を再読込する。
- スタンドアロン endpointを利用する場合、その出力全体をAPI向けの次コンテキストとして保持し、ハーネス固有の契約やメールボックス ビューは次の入力構築時に明示的に加える。
