# 開発作業の共通規約

このリポジトリの変更は、変更内容を次の経路へ分類して進める。3経路の分類と例外は本ファイル、製品変更経路の詳細手順は[開発プロセス](docs/development/README.md)を正本とする。分類に迷う場合、または独立レビューで根拠の矛盾、受け入れ条件の意味変更、安全上の含意が見つかった場合は、軽い経路を停止して安全契約変更へ再分類する。

## 作業経路の分類

### 製品変更

製品コード、テスト、ランタイム/build設定、Schema、宣言済み製品依存の追加・削除・バージョン・設定、生成される製品入力または成果物、外部観測可能な挙動のいずれかを変更する場合は、このリポジトリのmain管理`tasks/`に登録したTaskを起点とし、完全な`PLAN → QA_PLAN → DEV → 同一candidateの独立REVIEW/QA → マージ後の環境依存確認`を適用する。

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
5. 製品変更のレビュアー AgentはDEVが固定した同一`candidate_commit`/`candidate_tree`を独立にレビューし、`make check`を完了してmain管理`REVIEW_RESULT.md`へ対象を記録する。QA Agentも同じ案から独立に開始し、相互のPASSを開始条件にしない。
6. main Agentだけが`main`へ`--no-ff`でマージする。承認案と`merge_tree`の同一性を確認し、差異があれば結果を持ち越さない。
7. 案またはマージ後確認のFAILは実装不具合と決めつけず、[QAガイドライン](docs/development/qa.md)に従って原因を分類する。
8. 配下に別の`AGENTS.md`がある場合は、その追加手順も守る。
9. 子Agentの標準起動は内部`agents.spawn_agent`とする。`agent_type`欠落、内部`Spawn Agent`利用不能、または`model/effort`不一致は停止・証跡化し、異なる起動経路で迂回しない。main管理証跡の公開は親が`make evidence-commit`を使い、共通ロックとaction スコープを所有する。
10. 子Agentは原則としてステージ、コミット、マージ、`.git`書き込みを行わない。ただしレビュアー/QA Agentは、自ら軽微と判断した指摘をTask ワークツリーで修正・ステージ・コミットできる。Task ブランチへの取り込み後はその指摘を解消済みとしてPASSにでき、DEV差し戻し、再REVIEW、再QA、`qa_carry_forward`を要求しない。Mainだけが`main`へのmerge/pushを行う。
11. ロールとモデルは`.codex`の正規 契約に従う。mainは`Sol/high`、PLAN/QA/REVIEWは`Terra/medium`に固定し、DEVは承認済みPLANの`luna-xhigh`または`sol-high`を使う。各ロールの`Explorer`は`Luna/medium/read-only`で一件の限定質問だけを扱う。

### 案とQA実施モード

製品変更では、DEVが評価対象を`candidate_commit`（評価対象コミット）と`candidate_tree`（そのtree）として固定し、ケース ID、コマンド/テスト、環境またはフィクスチャ、cache条件、exit、成果物 ダイジェスト、未実施理由を運用証跡へ結び付ける。QA_PLANはDEV開始前に各ケースへ次の一つを理由付きで割り当てる。

- `evidence-review`: candidate-bound証跡、テストの失敗検出能力、ネガティブ ケース、弱体化の有無をQAが独立監査する。
- `focused-rerun`: 高リスクでもhermetic・deterministic・上限付き フィクスチャで受け入れ真実を完全再現できるケースを、QAが独立に限定再実行する。
- `live-e2e`: 実OS権限/auth（sudo/PAMを含む）、実配置、外部作用、実restart/ロールバック/クリーンアップ、環境固有integrationに依存するケースを、承認済み実環境で確認する。環境または安全なクリーンアップが不明ならblockedのままとし、別モードのPASSで代替しない。

高リスク信号、証跡不足、案/tree不一致、影響不明は`evidence-review`のPASSを禁止する。REVIEWとQAは同一案から独立かつ並行に評価し、Mainだけが修正後の`qa_carry_forward`または限定/全面再実行を選ぶ。carry-forwardは[QAガイドライン](docs/development/qa.md)の閉じた`CF-1`から`CF-7`を全て証明した場合だけ許可する。変更は実行されない誤字、空白、コメント、リンク、証跡メタデータに限定し、意味変更は許可しない。影響QAケース集合が空でなければ該当ケースを再実行し、限定できなければ全面再実行とする。QA FAIL、受け入れ条件/QA_PLAN変更、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/ライフサイクル/persistence/エラー/fail-closed、テスト削除/弱体化、影響不明、証跡と評価対象の案/tree不一致はcarry-forwardを禁止する。`merge_tree == candidate_tree`で環境依存ケースがない場合だけ全面的な重複確認を省略でき、環境依存ケースはマージ後もケース単位で確認する。既存Task証跡とLap30 イベント Schema/JSONLは遡及変更しない。

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

起動後は、選択したロールの契約と実際の`model/effort`を照合する。不一致なら子の成果を採用せず停止し、requested/observed値とランタイム条件を証跡化する。`agent_type`または内部`Spawn Agent`が利用できない場合も停止する。限定調査だけは一問専用の`make explorer-agent`を使用できる。ロール対応、ゲート順序、`Explorer`の制約、サンドボックス観測限界は[Agent責務](docs/development/agent-roles.md)を正本とする。

`role` TOMLの`sandbox_mode`は意図する契約であり、ランタイムで観測できた値だけを証跡に記録する。実効サンドボックスをTOMLの宣言だけで保証済みとは扱わない。レビュアー/QAの軽微修正コミットを除き、子の`stage`、`commit`、`merge`、`.git`書込みは禁止する。Mainは`main`への統合を所有する。

## 共通検査

製品変更では次を実行する。

```sh
make check
make task-check TASK=TASK-NNNN
```

安全契約変更と純粋な証跡保守では、変更した契約または証跡が影響する検査、`git diff --check`、リポジトリ所定のスコープ/hook検査を実行する。影響しない製品検査を製品PASSの証拠として扱わない。

Task証跡、滞留、Wiki、Lap30の正本はこのリポジトリのmain ワークツリーに置く。コード用Task ワークツリーからこれらをsparse-checkoutで除外し、証跡公開時は明示したmain ルートへ経路する。
