# Kakesu用語索引

<!-- このファイルはdocs/glossary.ymlから自動生成する。直接編集しない。 -->

Kakesu内で固有の意味を持つ用語について、標準表記、短い説明、定義元をまとめる。表記規則と表記揺れの正本は[glossary.yml](glossary.yml)とする。

| 標準表記 | 正式名 | 説明 | 定義 |
|---|---|---|---|
| Kakesu | `Kakesu` | 長期記憶を次の行動へ活かす、ローカル動作の自律AIハーネス。 | [設計書](00-kakesu.md#1-目的) |
| ハーネス | `Harness` | 状態、排他、再開、Workspace、配送、監査を管理するKakesuの実行基盤。 | [設計書](00-kakesu.md#3-全体アーキテクチャ) |
| Agent | `Agent` | 能力プロファイルを持ち、Taskのオーナーになれる継続的な論理主体。 | [設計書](01-domain-model.md#1-agent) |
| Task | `Task` | 単一オーナーが目的と受け入れ条件に対して責任を持つ作業単位。 | [設計書](01-domain-model.md#2-task) |
| Workspace | `Workspace` | Task専用の隔離された論理作業領域であり、セキュリティポリシーの適用単位。 | [設計書](01-domain-model.md#3-workspace) |
| オーナー | `Owner` | あるTaskの遂行と完了案の提出に責任を持つAgent。 | [設計書](01-domain-model.md#オーナー排他) |
| Agent実行 | `Agent Run` | あるAgentがあるTaskを処理する一回の実行セッション。 | [設計書](01-domain-model.md#4-agent実行) |
| Task契約 | `Task Contract` | Taskの目的、受け入れ条件、指示、優先順位などを固定する契約。 | [設計書](01-domain-model.md#2-task) |
| 目的 | `Objective` | Taskが達成すべき目的。実行中の都合だけでは変更しない。 | [設計書](01-domain-model.md#taskの成立条件) |
| 受け入れ条件 | `Acceptance` | Taskを完了として受理できるかを判断する受け入れ条件。 | [設計書](01-domain-model.md#taskの成立条件) |
| 結果 | `Outcome` | Taskの終端時に確定する結果。完了時は受理済みレビューを必要とする。 | [設計書](01-domain-model.md#結果制約) |
| 受付 | `Intake` | 人間や外部システムからルートTaskの申請を受け付ける境界。 | [設計書](02-task-lifecycle.md#2-ルート-task生成) |
| Task申請 | `Task Proposal` | 親子関係、目的、受け入れ条件、予算を含むTask作成申請。 | [設計書](02-task-lifecycle.md#3-子-task生成) |
| Task進捗 | `Task Progress` | オーナーが申告するTask内の現在地と未完了項目。 | [設計書](01-domain-model.md#task進捗) |
| 台帳 | `Ledger` | Task進捗をTODO形式で保持する永続台帳。 | [設計書](02-task-lifecycle.md#5-task進捗) |
| SubAgent | `SubAgent` | 親Taskから委譲された子Taskを所有するAgent。 | [設計書](00-kakesu.md#subagent) |
| 質問 | `Ask` | 判断責任を移さず、子Taskのオーナーが親へ助言を求める通信。 | [設計書](02-task-lifecycle.md#質問) |
| 上位判断依頼 | `Escalation` | 現在のTask契約では決められない判断責任を上位へ移す通信。 | [設計書](02-task-lifecycle.md#上位判断依頼) |
| 完了案 | `Completion Candidate` | オーナーがTaskの完了条件を満たしたとして提出する完了案。 | [設計書](02-task-lifecycle.md#7-完了候補と受け入れ条件-レビュー) |
| 完了レビュー | `Completion Review` | 完了案を現行の受け入れ条件と照合する独立レビュー。 | [設計書](01-domain-model.md#5-完了レビュー) |
| 受け入れ条件レビュアー | `Acceptance Reviewer` | 完了案だけを評価し、Taskを実装または修正しない短命なレビュアーAgent。 | [設計書](04-built-in-agents.md#3-受け入れ条件レビュアー-agent) |
| Taskエピソード | `Task Episode` | 終端したTaskから作る、時系列の長期記憶単位。 | [設計書](08-long-term-memory.md#3-taskエピソード) |
| Plane | `Plane` | 独立した責務、状態所有権、通信境界を持つ論理コンポーネント群。 | [設計書](00-kakesu.md#31-実装境界とplane間配送) |
| Control Plane | `Control Plane` | Task、再開、メールボックス、人間との通信境界を管理するPlane。 | [設計書](00-kakesu.md#24-人間との通信境界をcontrol-planeへ一元化) |
| Work Agent Plane | `Work Agent Plane` | 目的理解、計画、実装、調査、委譲、完了案提出を担うPlane。 | [設計書](00-kakesu.md#2-基本原則) |
| Execution Plane | `Execution Plane` | Workspace内のコマンドやツール実行を担うPlane。 | [設計書](13-technology-stack.md#1-選定結果) |
| Governance Plane | `Governance Plane` | 外向き通信、認証情報、許可、監査、ポリシーを強制するPlane。 | [設計書](07-governance.md#1-信頼境界) |
| Memory Plane | `Memory Plane` | 証跡からTaskエピソードと意味記憶を構築するPlane。 | [設計書](08-long-term-memory.md#1-目的) |
| メールボックス | `Mailbox` | Task宛ての非同期結果や統治イベントを順序付きで保持する受信箱。 | [設計書](06-tools-and-async.md#13-メールボックスイベント) |
| 送信キュー | `Outbox` | Plane内の状態変更と同じトランザクションで保存する送信待ちメッセージのキュー。 | [設計書](13-technology-stack.md#5-plane間通信) |
| 受信キュー | `Inbox` | 受信メッセージを重複排除して永続化する受信キュー。 | [設計書](13-technology-stack.md#5-plane間通信) |
| 非同期操作 | `Async Operation` | 同期期限を超えて継続し、後から結果を返す操作の永続記録。 | [設計書](11-data-model.md#async_operations) |
| 継続情報 | `Continuation` | Taskを後から安全に再開するために固定する待機理由と再開情報。 | [設計書](11-data-model.md#continuations) |
| 外向き通信の許可確認 | `Egress Challenge` | 外向き通信がルールで拒否されたことと、許可申請の対象を固定する記録。 | [設計書](07-governance.md#6-外向き通信-試行と許可確認) |
| 一時許可 | `Policy Grant` | 特定のWorkspace、Task、通信内容、期限へ束縛した一時許可。 | [設計書](07-governance.md#11-一時許可) |
| ポリシー割り当て | `Policy Binding` | Workspaceへ適用するポリシーとバージョンの割り当て。 | [設計書](07-governance.md#11-実装永続化境界) |
| 人間の責任者 | `Human Authority` | 人間の判断が必要なときに、認証済み回答を返す責任者。 | [設計書](07-governance.md#13-責任者) |
| 責任者ゲートウェイ | `Authority Gateway` | 人間の責任者への依頼と回答を一元管理するControl Planeの境界。 | [設計書](00-kakesu.md#24-人間との通信境界をcontrol-planeへ一元化) |
| ポリシーエージェント | `Policy Agent` | 外向き通信の許可判断と恒久ルール改定案を作る独立Agent。 | [設計書](07-governance.md#10-ポリシーagent) |
| 外向き通信監査エージェント | `Egress Audit Agent` | 通過後の外向き通信を調査し、見逃しや異常を検出するAgent。 | [設計書](07-governance.md#16-監査事後検知ポリシー改善) |
| インシデント | `Incident` | 安全性または統治上の異常に対して封じ込めと復旧を要する事象。 | [設計書](02-task-lifecycle.md#インシデントによるtask-tree-suspension) |
| 封じ込め | `Containment` | インシデントの影響を起点Task、その祖先、子孫へ限定して停止する処置。 | [設計書](02-task-lifecycle.md#インシデントによるtask-tree-suspension) |
| 証跡 | `Evidence` | 再開、完了判定、監査、長期記憶を裏付ける不変または検証可能な証跡。 | [設計書](08-long-term-memory.md#4-証跡-レイヤー) |
| 意味Wiki | `Semantic Wiki` | 概念、スキーマ、スクリプト、ケースパターンを関係付きで保持する意味記憶。 | [設計書](09-semantic-wiki-schema.md) |
