# Wiki Agent規約

Wiki本文とDecisionの保守はWiki Agentの責任である。人間またはmain Agentによる本文レビューを通常ゲートにしない。main AgentはSchema、検証規則、権限境界だけを変更する。

Wiki AgentはMainが内部`agents.spawn_agent(task_name=..., agent_type="wiki", fork_turns="none", ...)`で起動する。Mainは対象Taskと許可パスを先に固定し、Wiki Agentを同時に複数起動しない。Wiki Agentは指定されたWikiパスの編集だけを行い、別Agent起動、ステージング、コミット、merge、`.git`書込みを行わない。

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
7. Wiki Agentは変更パスをMainへ引き継ぎ、検証やGit操作を行わず終了する。
8. Mainは変更スコープを確認し、repository rootで`make evidence-commit TASK=TASK-NNNN ACTION=wiki MESSAGE='wiki: ingest TASK-NNNN'`を一度だけ実行する。
9. この共通ロック付き公開トランザクションがdirty Wiki差分からの索引生成、生成後の最終スコープ検査、`work-check`、ステージング、単一コミット、pushを所有する。スタンドアロンの`make wiki-index`は保守用generatorであり、標準公開には使用しない。pre-commit hookを迂回してはならない。

Wiki receiptはMainが明示的にingestを依頼した場合だけ作成する成果物であり、Task完了条件ではない。Wiki依頼がないTaskはWiki Agentもreceiptもなしで完了できる。Mainは同時writerを作らず、Wiki Agent終了後の入力スコープを確認してから上記トランザクションで公開する。`.githooks/pre-commit`はコミット前に許可パス、Decision不変条件、Schema、Taskゲート、HANDOVER digestを検査する。

## 品質規則

- Semanticページは一つの中心的な問いに答える。
- 観測事実、推論、未解決を混同しない。
- 反例と適用限界を削除しない。
- Task固有の根拠へ相対リンクする。
- Wiki本文とfrontmatterへ同じ情報を重複させない。
- Decision本文は確定後に意味を変えない。
