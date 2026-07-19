# 開発作業の共通規約

このリポジトリの変更は、変更内容を次の経路へ分類して進める。3経路の分類と例外は本ファイル、製品変更経路の詳細手順は[開発プロセス](docs/development/README.md)を正本とする。分類に迷う場合、または独立レビューで根拠の矛盾、受け入れ条件の意味変更、安全上の含意が見つかった場合は、軽い経路を停止して安全契約変更へ再分類する。

## 作業経路の分類

### 製品変更

製品コード、テスト、ランタイム/build設定、Schema、宣言済み製品依存の追加・削除・バージョン・設定、生成される製品入力または成果物、外部観測可能な挙動のいずれかを変更する場合は、外部の運用リポジトリ`../agent-harness-work`に登録したTaskを起点とし、完全な`PLAN → QA_PLAN → DEV → 独立REVIEW → マージ後QA`を適用する。

### 安全契約変更

製品成果物および宣言済み製品依存を一切変更しない場合に限り、セキュリティ/権限境界、脅威モデル、受け入れ条件、機能スコープ、依存方針またはTask順序、リソース上限、機能削減順、または必須開発統制の変更を安全契約変更とする。この経路ではTask、PLAN、TASK本文だけから先に作る独立QA_PLAN、独立した計画レビューを必須とする。製品実装がない場合は、製品DEV、`REVIEW_RESULT.md`、`QA_RESULT.md`のPASSを作らない。

### 純粋な証跡保守

次のすべてを満たす場合だけ、専用Task、ワークツリー、PLAN、QA_PLAN、DEV Agent、QA Agent、カウント対象Lap、単独PRを省略できる。

- 変更がバックログ状態、計測算術、SLOC、時間、再試行、FAIL分類、証跡リンク、またはそれらの`append-only correction`だけである。
- 製品変更にも安全契約変更にも該当せず、挙動、ポリシー、受け入れ条件の意味を変えない。
- Main Agentが目的、出典、対象パス、算術または対応関係、除外、影響する検査、訂正またはロールバック方法のchecklistを残す。
- 独立レビュアー1名が出典、算術、状態遷移、ファイル間メタデータ、Schema/parser、差分スコープ、秘密情報不在を確認する。

公開済み証跡を書き換えず、既存の訂正方式で履歴を保存する。影響しない製品テスト群は繰り返さず、可能なら関連製品Taskのマージ後処理へ含める。レビュアーが意味変更または根拠不整合を発見した場合は承認せず再分類する。

## 必須事項

1. 製品変更のTaskごとに`task/TASK-NNNN-short-slug`ブランチと専用ワークツリーを使う。
2. 製品変更では`PLAN / DEV / QA`の各ゲートを飛ばさない。
3. 製品変更のDEV開始前に承認済み`PLAN.md`と独立した`QA_PLAN.md`を用意する。
4. 製品変更ではDEV Agentとレビュアー Agent、DEV AgentとQA Agentを分離する。
5. 製品変更のレビュアー Agentは独立レビューと`make check`を完了し、外部運用リポジトリの`REVIEW_RESULT.md`へ証跡を残す。
6. main Agentだけが`main`へ`--no-ff`でマージする。
7. マージ後QAのFAILは実装不具合と決めつけず、[QAガイドライン](docs/development/qa.md)に従って原因を分類する。
8. 配下に別の`AGENTS.md`がある場合は、その追加手順も守る。
9. 子Agentの標準起動は内部`agents.spawn_agent`とし、`agent_type`欠落、内部`Spawn Agent`利用不能、または`model/effort`不一致を停止・証跡化した後に限り、親が`make work-agent TASK=TASK-NNNN ACTION=<action>`を`fallback`として使う。運用リポジトリへ証跡を書く場合は親が共通ロックを実行全体で保持し、直接並行編集しない。
10. どの経路でも子Agentはステージ、コミット、マージ、`.git`書き込みを行わない。運用リポジトリのコミットは、子成功後に共通ロックを保持するランチャー親がスコープ、フック、検証を通して作成する。
11. ロールとモデルは`.codex`の正規 契約に従う。mainは`Sol/high`、PLAN/QA/REVIEWは`Terra/medium`に固定し、DEVは承認済みPLANの`luna-xhigh`または`sol-high`を使う。各ロールの`Explorer`は`Luna/medium/read-only`で一件の限定質問だけを扱う。

## 子Agentの標準起動

子Agentは内部の`agents.spawn_agent`を標準経路として起動する。`task_name`は追跡用の一意な識別子であり、ロールを選択する値ではない。ロール選択は`agent_type`で行い、呼出元と異なるロールを指定する場合は必ず`fork_turns="none"`を渡す。

```text
agents.spawn_agent(
  task_name="task_0003_dev_docs",
  agent_type="dev-luna",
  fork_turns="none",
  message="承認済みPLANの範囲で指定文書だけを更新する"
)
```

起動後は、選択したロールの契約と実際の`model/effort`を照合する。不一致なら子の成果を採用せず停止し、requested/observed値とランタイム条件を証跡化してから`fallback`可否を判断する。`agent_type`または内部`Spawn Agent`が利用できない場合も、親は原因を記録してから判断する。`fallback`を選べるのはこれらの場合だけであり、親が`make work-agent`（`Explorer`は一問専用の`make explorer-agent`）を使う。ロール対応、順次ゲート、`Explorer`の制約、サンドボックス観測限界は[Agent責務](docs/development/agent-roles.md)を正本とする。

`role` TOMLの`sandbox_mode`は意図する契約であり、ランタイムで観測できた値だけを証跡に記録する。実効サンドボックスをTOMLの宣言だけで保証済みとは扱わない。子の`stage`、`commit`、`merge`、`.git`書込みは禁止し、共通ロック、スコープ検査、`hook`、`stage`、`commit`、事後検査は親（またはmain）が所有する。

## 共通検査

製品変更では次を実行する。

```sh
make check
make task-check TASK=TASK-NNNN
```

安全契約変更と純粋な証跡保守では、変更した契約または証跡が影響する検査、`git diff --check`、リポジトリ所定のスコープ/hook検査を実行する。影響しない製品検査を製品PASSの証拠として扱わない。

外部運用リポジトリの場所は`WORK_ROOT`で上書きできる。既定値は`../agent-harness-work`である。
