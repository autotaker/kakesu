# Governance Plane Schema カタログ — draft-v0

Workspace セキュリティ ポリシー、CASB ルール、認証情報 スコープ、外向き通信 監査、一時 許可、恒久ポリシー 改訂、責任者が判断する対象と要否を所有する。人間との送受信は所有せず、すべてControl Planeの責任者ゲートウェイを経由する。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `workspace-security-profile.schema.json` | Workspaceポリシー割り当て、認証情報 スコープ、ネットワーク ケイパビリティ、`pending` 改訂 |
| `casb-policy.schema.json` | ポリシー ID/バージョン、対象 キー、ルール priority、match、許可/拒否、deny-overrides |
| `casb-rule.schema.json` | 宛先/プロトコル/ポート、method/パス/本文、分類、認証情報、TTL/使用 上限 |
| `egress-request-binding.schema.json` | 正規 リクエスト ダイジェストを構成する全フィールド |
| `egress-attempt.schema.json` | Workspace/Task/Agent 来歴、割り当て、分類、ルール結果 |
| `egress-challenge.schema.json` | Workspace、割り当て、理由、許可/責任者 適格性、期限切れ |
| `egress-rule-decision.schema.json` | 一致 ルール、ポリシーバージョン、許可/拒否、理由 |
| `egress-capture-manifest.schema.json` | captured/秘匿済み 範囲、切り詰め、検査 上限、保持/ピン留め |
| `outbound-transaction.schema.json` | 意図、forwarded、`completed`、`failed`、結果 unknown |
| `policy-grant.schema.json` | Workspace-scoped 一時 ルール、起点 判断、limits、有効化 |
| `policy-revision.schema.json` | ジョブ、固定 候補、提案、責任者、CAS、`pending`/`active`、`ACK` |
| `policy-revision-job.schema.json` | 検出事項 結合、入力 スナップショット、判断前候補固定、リース/再試行 |
| `governance-authority-request.schema.json` | 許可 / 改訂を責任者へ提示する不変 ペイロード |
| `governance-authority-decision.schema.json` | 回答者、承認/拒否、根拠、decided 時刻 |
| `policy-agent-input.schema.json` | 許可/改訂評価の固定入力 スナップショット |
| `egress-audit-input.schema.json` | 試行、ルール 判断、キャプチャ マニフェスト、ポリシー割り当ての固定スナップショット |

## P1

| Schema | 固定する内容 |
|---|---|
| `dns-resolution.schema.json` | Workspace-scoped DNS 来歴、TTL、解決済み IP |
| `credential-binding.schema.json` | ブローカー センチネル、Workspace、プロバイダー 主体、リソース スコープ |
| `policy-regression-result.schema.json` | 再実行 データセット、見逃し、過剰拒否、網羅率 |
| `policy-candidate.schema.json` | 改訂 ジョブに固定する候補 参照/ダイジェスト、基底 バージョン、固定 タイムスタンプ |
| `security-incident.schema.json` | 検出事項、リスク、封じ込め、改訂を結ぶインシデント ライフサイクル |
| `incident-risk-assessment.schema.json` | ルール 下限、レビュアー推奨、effective リスク、人間 ゲート |
| `incident-containment.schema.json` | 許可 失効、外向き通信制限、Workspace 凍結、固定グラフ上の祖先・発生元・子孫Task 停止集合 |
| `incident-authority-request.schema.json` | High/Critical インシデント dispositionとTask 再開の人判断要求 |
| `incident-authority-decision.schema.json` | keep `suspended` / 再開 / キャンセルの認証済み判断 |
| `policy-finding.schema.json` | benign / 迂回 / suspicious / 不十分 証跡 |

## 現在のAPI アダプター

許可判断、外向き通信 レビュー、ポリシー 改訂 判断、責任者の判断 ツールは`../api/`の合成バンドルに含まれる。候補 ルール本文は構造化 出力へ含めず、`casb-policy.schema.json`で別に検証する。
