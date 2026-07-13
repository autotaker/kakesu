# Agent責務

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

DEV Agentは専用ワークツリーで実装し、テストと必要文書を含めた変更をコミットする。自分の変更を最終承認しない。運用リポジトリでは担当TaskのDEV欄と`HANDOVER.md`だけを更新できる。

## レビュアー Agent

レビュアー AgentはDEV Agentから独立し、差分、Task、PLAN、QA計画、該当ガイドラインを読む。コード品質だけでなく受け入れ条件、責務境界、障害時の挙動、テストの失敗検出能力を確認する。結果の正本は`REVIEW_RESULT.md`とする。

## QA Agent

QA AgentはDEV Agentから独立する。実装前にQA計画を作り、実装後に認識差を再確認し、マージ済み`main`へ受け入れレビューを行う。FAILの原因を分類し、DEVの責任と自動的に見なさない。

## Wiki Agent

Wiki Agentは`HANDOVER.md`から再利用可能な知識を抽出し、意味 Wikiと判断を自律保守する。Wiki本文のレビューをmain Agentへ要求せず、Wiki Schemaと保守規約に従って検査後に運用リポジトリの`main`へ直接コミットする。

Wiki AgentはTask契約、PLAN、レビュー結果、QA結果、バックログを変更しない。Wiki Schemaの変更が必要な場合は更新せず、取り込み記録へ保留理由を残す。

## 運用リポジトリへの書き込み

Planner、レビュアー、QA、mainなどの書き込みAgentは、次のランチャーから起動する。ランチャーは編集開始前からコミット後の再検査まで共通ロックを保持し、action別の許可ファイルをpre-commitフックへ渡す。

```sh
make work-agent TASK=TASK-0001 ACTION=plan PROFILE=planner
make work-agent TASK=TASK-0001 ACTION=review PROFILE=reviewer
make work-agent TASK=TASK-0001 ACTION=qa-result PROFILE=qa
```

`ACTION`は`task | plan | qa-plan | review | qa-result | handover | main-transition | governance`のいずれかとする。`governance`はmain AgentがSchema、フック、Wiki保守規約を変更する場合にだけ使う。Task作成、ワークツリー割り当て、Wiki保守は、それぞれ専用コマンドが同じ共通ロックを使う。

Wiki Agentは既定で`gpt-5.6-terra`を使う。Codex CLIの既定プロファイルまたはモデルが実行環境と互換でない場合、`PROFILE`、`MODEL`、`WIKI_PROFILE`、`WIKI_MODEL`でランチャー単位に上書きする。グローバル設定を書き換えない。

## 兼任規則

- DEV Agentとレビュアー Agentの兼任は禁止する。
- DEV AgentとQA Agentの兼任は禁止する。
- Planner AgentとQA Agentは兼任可能だが、可能なら分離する。
- main Agentは例外判断とマージを担い、通常実装を兼任しない。
