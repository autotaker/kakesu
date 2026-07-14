# Agent責務

## Codex モデル ルーティング

製品リポジトリの`.codex/config.toml`と`.codex/agents/*.toml`をロール契約の正本とする。グローバル設定やモデルエイリアスには依存しない。

| ロール | モデル | 推論 effort | 備考 |
|---|---|---|---|
| main | `gpt-5.6-sol` | `high` | 承認、統合、FAIL分類を所有する。 |
| Planner / QA / レビュアー | `gpt-5.6-terra` | `medium` | 固定ロールであり、異なる上書きを拒否する。 |
| DEV `Luna` | `gpt-5.6-luna` | `xhigh` | 全低リスク条件を満たすTaskだけに使う。 |
| DEV `Sol` | `gpt-5.6-sol` | `high` | 高リスク、横断的、または不明なTaskに使う。 |
| `Explorer` | `gpt-5.6-luna` | `medium` | 一件の限定質問だけを読み取り専用で調査する。 |

ルートは`Explorer`を直接呼べる。ルートからロールを経由した`Explorer`呼び出しも許可するが、最大深さは2、最大thread数は2である。`Explorer`は編集、Git操作、スコープ拡大、再委譲を行わず、短い要約とファイル参照だけを返す。Agentの自己申告をルーティング証跡として信用せず、正規 TOMLとランチャーが出力する一行JSONを検査する。

## main Agent

main Agentは全体の判断者であり、原則として実装しない。

- Task契約の確認とAgentのアサイン
- PLANのレビューと承認
- ブランチとワークツリーの割り当て
- FAIL分類と差し戻し先の最終判断
- `main`へのマージとrevert
- プロセス、Schema、権限境界の変更

## Planner Agent

Planner Agentは受け入れ条件と設計観点を具体化し、`PLAN.md`を作る。PLAN承認前に製品コードを変更しない。Wikiの関連テーマ、現行判断、置換済み判断を確認する。

## DEV Agent

DEV Agentは専用ワークツリーで実装し、テストと必要文書を含めた変更を親Agentへ引き渡す。自分の変更を最終承認しない。運用リポジトリでは担当Taskの`HANDOVER.md`だけを更新できる。子Agentはステージまたはコミットを行わず、ロックを持つランチャー親がスコープ検査、ステージ、フック、検証、コミット、再検証を所有する。

## レビュアー Agent

レビュアー AgentはDEV Agentから独立し、差分、Task、PLAN、QA計画、該当ガイドラインを読む。コード品質だけでなく受け入れ条件、責務境界、障害時の挙動、テストの失敗検出能力を確認する。結果の正本は`REVIEW_RESULT.md`とする。

## QA Agent

QA AgentはDEV Agentから独立する。実装前にQA計画を作り、実装後に認識差を再確認し、マージ済み`main`へ受け入れレビューを行う。FAILの原因を分類し、DEVの責任と自動的に見なさない。

## Wiki Agent

Wiki Agentは`HANDOVER.md`から再利用可能な知識を抽出し、意味 Wikiと判断を自律保守する。Wiki本文のレビューをmain Agentへ要求せず、Wiki Schemaと保守規約に従って検査後に運用リポジトリの`main`へ直接コミットする。

Wiki AgentはTask契約、PLAN、レビュー結果、QA結果、バックログを変更しない。Wiki Schemaの変更が必要な場合は更新せず、取り込み記録へ保留理由を残す。

## 運用リポジトリへの書き込み

Planner、レビュアー、QA、mainなどの書き込みAgentは、次のランチャーから起動する。ランチャーは編集開始前からコミット後の再検査まで共通ロックを保持する。子stdinはclosedであり、子には`.git`書き込みを与えない。子が成功した後だけ、親がaction別の許可ファイルをpre-commitフックへ渡してコミットする。

```sh
make work-agent TASK=TASK-0001 ACTION=plan
make work-agent TASK=TASK-0001 ACTION=review
make work-agent TASK=TASK-0001 ACTION=qa-result
```

`ACTION`は`task | plan | qa-plan | review | qa-result | handover | main-transition | governance`のいずれかとする。`governance`はmain AgentがSchema、フック、Wiki保守規約を変更する場合にだけ使う。Task作成、ワークツリー割り当て、Wiki保守は、それぞれ専用コマンドが同じ共通ロックを使う。

固定ロールへ`PROFILE`、`MODEL`、`EFFORT`を渡す場合、正規値と同一でなければ起動前に失敗する。Wiki Agentは固定ロールを経由しない文書化済みlegacy経路であり、既定で`gpt-5.6-terra` / `medium`を使う。Wiki経路だけは`WIKI_PROFILE`、`WIKI_MODEL`、`WIKI_EFFORT`で上書きできる。グローバル設定を書き換えない。

運用リポジトリの`.codex/config.toml`はmain Agentが生成するアダプターである。製品側を`main`へマージした後、QA前に`make work-config-sync`を実行し、共通ロック下の`ACTION=governance`で別コミットする。`make work-config-sync CHECK=1`は正規 ダイジェストを含む完全一致を検査する。

## 兼任規則

- DEV Agentとレビュアー Agentの兼任は禁止する。
- DEV AgentとQA Agentの兼任は禁止する。
- Planner AgentとQA Agentは兼任可能だが、可能なら分離する。
- main Agentは例外判断とマージを担い、通常実装を兼任しない。
