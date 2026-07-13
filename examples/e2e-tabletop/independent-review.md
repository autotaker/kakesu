# E2E Tabletop Debugging 独立レビュー

## 最終判定

**PASS**。Schema レビュアー、シーケンス レビュアーの双方が、対象4シナリオを無条件PASSと判定した。

| 検査対象 | 結果 |
|---|---:|
| E2E シナリオ | 4 PASS |
| シーケンス ペイロード | 124 PASS |
| 正規 ドメイン ペイロード | 119 PASS |
| シーケンス Requirement | 4 PASS |
| ネガティブ 変異 | 11 拒否 |
| Idempotent 再配送 | PASS |
| Nested 相関 | PASS |

## 初回レビューで発見した不足と解消

| 発見 | 解消内容 |
|---|---|
| 許可が許可確認から`active`へ直行 | リクエスト、判断、`pending` 有効化、ルールエンジン `ACK`、`active`、準備完了、CLI 再試行を追加 |
| `ToolCall`と外向き通信の因果が不明 | ツール 呼び出し、プロセス、試行、許可確認、許可、トランザクションをID結合 |
| レビュー後のTask 終端確定がない | ReviewingCompletion、レビュー 入力/出力、`TaskCompleted`、エピソード 入力を追加 |
| 子Task生成・親統合が不完全 | 子 Workspace/Task、子 エピソード、メールボックス 消費、親 Integration/レビュー/エピソードを追加 |
| 非同期が`running`のまま完了通知 | 非同期 `completed`永続化、メールボックス 消費、ResumeContext、新実行、TaskResumedを追加 |
| ポリシー 改訂が`ACK`前に`active` | 案、Regression、責任者、`pending` 改訂、`ACK`、`active`を分離 |
| Requirementが宣言だけでvacuous pass | 必須型、順序、フィールド 結合、直接 因果関係を各シナリオへ適用 |
| Generic Schemaでドメイン フィールド不足を検出不能 | 98 メッセージを正規 ドメイン Schemaへ接続し、投影と値を比較 |
| Task状態が任意文字列 | 正規 Taskイベント、状態連続性、許可遷移表を追加 |
| Workspace 分岐/許可継承が未定義 | Workspace Created Schemaで親、モード、ポリシー割り当て、許可非継承を固定 |
| メールボックス 消費が未定義 | consumer、シーケンス、ウォーターマーク、冪等性、状態、時刻をSchema化 |
| 再配送を一律拒否 | 同一操作 フィンガープリントと再配送 メタデータを持つ再配送だけ許可 |

## 独立シーケンス レビュー

4シナリオとも、開始メッセージからTaskエピソード確定まで到達可能である。

- E2E-001: テスト、GitHub 外向き通信 許可、再試行、PR トランザクション、Task完了
- E2E-002: 子生成、子限定許可、子 エピソード、メールボックス 消費、親統合・完了
- E2E-003: 非同期 待機、完了、メールボックス 消費、圧縮後の新実行再開、Task完了
- E2E-004: 迂回 検出事項、ポリシー 候補/regression、責任者、`ACK`後有効化、是正完了

## 独立Schema レビュー

124 シーケンス ペイロードのうち、独立ドメイン 状態を表す119 メッセージは正規 Schemaで検証する。
残りは独立ドメイン 状態を更新しない投影であり、代表例は次である。

- `ExecutionAuditRecord`: 実行後の監査投影
- `ParentIntegration`: メールボックス消費後の親実行内部処理

シーケンスと正規 ペイロードはメッセージ IDだけでなく、Task、Workspace、同名参照/ダイジェスト、
Taskイベントのfrom/to 状態で照合する。

## 機械検査

```sh
node scripts/validate-tabletop-scenarios.mjs
node scripts/test-tabletop-validator.mjs
```

ネガティブ テストは、必須メッセージ欠落、結合 パス欠落、状態欠落、不正cause、誤ったprior cause、
冪等性衝突、正規 割り当て欠落、投影/正規不一致、不正状態遷移を拒否する。

## 非ブロッキングな将来メモ

- 製品実装時はAjv等の完全な下書き 2020-12 検証器でも検証する。
- DB 一意 constraint、並行配送 race、ルールエンジン実装はintegration テストで確認する。
- `shared_readonly` WorkspaceのE2Eを追加するとき、Workspace Created Schemaのモード拡張要否を判断する。

## 004 インシデント責務改定

独立レビュー後、E2E-004は統治主体のインシデント ワークフローへ改定した。ポリシー修正を
Control Planeの是正Taskとして扱わず、High リスク判定、一時封じ込め、Task 停止、
人間のインシデント責任者、ポリシー 改訂 責任者、是正後の人間 再開判断を表現する。

改定後は封じ込め解除、停止済み実行とは別のAgent実行開始、Task再開までを追加した。
E2E-004の人間 インシデント、改訂、Task 再開の各責任者通信は、Governance Planeから
Control Planeの責任者ゲートウェイへリクエストを渡し、ゲートウェイから判断を返す経路に統一する。
Governance Planeから人間への直接通信は許可しない。
Taskツリー フィクスチャによる祖先・発生元・子孫のカスケードと兄弟除外を追加した。最終機械検査は、
`4 scenarios / 124 sequence payloads / 119 canonical domain payloads`でPASSし、
シーケンス レビュアーとスキーマ レビュアーの独立再レビューもともにPASSした。未使用正規 ペイロードは
検査エラーとし、停止・開始双方のAgent実行 コマンド/イベントを正規 Schemaで検証する。

非ブロッキングな実装上の課題として、`AgentRunStarted`から`TaskResumed`まで実行がツール 呼び出しを
配送しないゲート、インシデント固有ネガティブ 変異、責任者 リクエスト 種別と判断の条件制約、
封じ込めの適用・解除時刻制約を追加する。

## 2026-07-13 技術スタック・プロセス境界レビュー

Go コア、Python 記憶、Rust 統治への実装分割と、`control.db`、`evidence.db`、`governance.db`の書き込み 所有権をレビューした。

- ドメイン メッセージの起点/対象 Plane、責任者ゲートウェイ経路、Task/許可/エピソードの状態遷移は変更していない。
- 送信キュー転送、受信キュー 永続 `ACK`、再送、照合はinfrastructure メッセージであり、既存ドメイン シーケンス 投影へ追加しない。
- `EgressBlocked`は統治 集約と統治 送信キューをコミットしてからCLIへ拒否を返し、制御 受信キュー適用後にTaskメールボックスへ一度だけ追加する。
- Plane横断トランザクションを廃止しても、許可 有効化/失効では統治 `ACK`まで操作ゲートを閉じるため安全性を弱めない。
- 記憶 フレームワークのセッション/チェックポイントは正本にせず、既存エピソード ジョブの固定入力、リース、再試行 semanticsを維持する。

Schema レビューでは正規 ペイロードのフィールド、メッセージ 型、状態列挙型に変更がないことを確認したため、Schema 改訂とネガティブ 変異の追加は不要と判定した。ベースライン、11 ネガティブ mutations、冪等 再配送、nested 相関は再実行してPASSした。

## Kakesu 名前空間改名レビュー

製品名をKakesuへ変更し、CLI、パッケージ、crate、Python module、Schema URN、正規 ペイロード、ビューアーを同時移行した。`draft-v0`の実装・永続化開始前であるため、旧`urn:agent-harness:`を`active` aliasとして残さず`urn:kakesu:`へ置換した。

Schemaのフィールド、必須条件、状態列挙型、メッセージ 型、シーケンス order、因果関係、相関は変更していない。したがって新しいネガティブ 変異は追加せず、旧名前空間が`active` 成果物へ残っていないことと、全正規 ペイロードが新名前空間で検証されることを名前変更固有の検査とした。
