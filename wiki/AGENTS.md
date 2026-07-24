# Wiki Agent規約

Wiki本文とDecisionの保守はWiki Agentの責任である。人間またはmain Agentによる本文レビューを通常ゲートにしない。main AgentはSchema、検証規則、権限境界だけを変更する。

## 許可された変更

- `semantic/**`
- `decisions/**`
- `ingestions/**`
- `index.json`

`SCHEMA.md`、この`AGENTS.md`、`../schemas/**`、`../tasks/**`、`../backlog.yaml`を変更してはならない。Schema変更が必要ならingest記録の`deferred`へ理由を残す。

## ingest手順

1. 指定Taskの`HANDOVER.md`と必要なTask証跡を読む。
2. HANDOVERのSHA-256を計算し、同じdigestのingest記録があれば変更なしで終了する。
3. 既存SemanticページとDecisionを検索し、同化、調節、新規作成の順で判断する。
4. 一Taskだけの事情、未確認の主張、単なる作業要約をSemantic Wikiへ昇格させない。
5. Decisionを置換する場合、旧Decision本文を改変せず、新Decisionから`supersedes`で参照する。
6. `ingestions/TASK-NNNN.json`を作成する。
7. repository rootで`make wiki-index`を実行する。
8. repository rootで`make work-check`を実行する。
9. 許可範囲だけが変更されたことを確認し、`wiki: ingest TASK-NNNN`で`main`へ直接コミットする。pre-commit hookを迂回してはならない。

Wiki Agent起動ラッパーが共通書き込みロックを保持する。`.githooks/pre-commit`はコミット前に許可パス、Decision不変条件、Schema、Taskゲート、HANDOVER digestを検査する。

## 品質規則

- Semanticページは一つの中心的な問いに答える。
- 観測事実、推論、未解決を混同しない。
- 反例と適用限界を削除しない。
- Task固有の根拠へ相対リンクする。
- Wiki本文とfrontmatterへ同じ情報を重複させない。
- Decision本文は確定後に意味を変えない。
