# Agent責務

## Codex モデル ルーティング

固定ロールの契約は製品リポジトリの`standalone`な`.codex/agents/*.toml`を正本とする。`project-scoped`な`.codex/config.toml`と運用リポジトリの生成アダプターは、`top-level`の`model`、`effort`、`sandbox`に加えて7件の`[agents.<role>]`と`config_file` mappingを維持する。明示的な`[agents]`ヘッダーと全体委譲上限は置かず、子テーブルが親テーブルを暗黙に作るTOML構造とする。`user-level`設定やモデルエイリアスには依存しない。

| ロール | モデル | 推論 `effort` | 備考 |
|---|---|---|---|
| main | `gpt-5.6-sol` | `high` | 承認、統合、FAIL分類を所有する。 |
| Planner / QA / レビュアー | `gpt-5.6-terra` | `medium` | 固定ロールであり、異なる上書きを拒否する。 |
| DEV `Luna` | `gpt-5.6-luna` | `xhigh` | 全低リスク条件を満たすTaskだけに使う。 |
| DEV `Sol` | `gpt-5.6-sol` | `high` | 高リスク、横断的、または不明なTaskに使う。 |
| `Explorer` | `gpt-5.6-luna` | `medium` | 一件の限定質問だけを読み取り専用で調査する。 |

## 内部`Spawn Agent`の起動契約

子Agentの標準経路は内部の`agents.spawn_agent`である。`task_name`は人間とランタイムが追跡する一意な識別子で、ロール選択には使わない。ロールは`agent_type`で選択し、呼出元と異なるロールを起動する時は、既定の`fork_turns="all"`に依存せず必ず`fork_turns="none"`を指定する。

```text
agents.spawn_agent(
  task_name="task_0003_review",
  agent_type="reviewer",
  fork_turns="none",
  message="DEV差分を承認済みPLANとQA計画に照合し、独立レビューを行う"
)
```

### ロール対応とゲート順序

| 責務 | `agent_type` | `model` / `effort` | 起動・責務上の注意 |
|---|---|---|---|
| main | `main` | `Sol / high` | 承認、統合、FAIL分類、`merge`を所有し、子へ委譲しない。 |
| PLAN | `planner` | `Terra / medium` | PLANの証跡のみ。DEV開始を承認しない。 |
| DEV（低リスク） | `dev-luna` | `Luna / xhigh` | 承認済み`luna-xhigh` PLANのTaskだけを実装する。 |
| DEV（高リスク/不明） | `dev-sol` | `Sol / high` | 承認済み`sol-high` PLANのTaskだけを実装する。 |
| `reviewer` | `reviewer` | `Terra / medium` | DEVが固定した同一案から独立し、差分と`make check`をレビューする。 |
| QA | `qa` | `Terra / medium` | DEVが固定した同一案から独立し、ケース別モードの受入れとFAIL分類を行う。 |
| 限定調査 | `explorer` | `Luna / medium` | 一問、`read-only`、再委譲禁止。 |

mainは`PLAN → DEV`を進めた後、レビュアーとQAを同一`candidate_commit`/`candidate_tree`から独立に並行開始できる。相互のPASSを開始条件にせず、DEVとレビュアー、DEVとQAは兼任させない。ネイティブ`Spawn Agent`への切替を理由に承認、統合、`merge`、FAIL分類を子へ移譲しない。

起動結果で期待する`model/effort`とobserved値が一致することを親が確認する。不一致なら子の成果を採用せず停止し、`role`、requested/observed `model`・`effort`、ランタイム条件を証跡化する。別の`task_name`や`role`名で回避・継続しない。

`agent_type`が非公開/欠落、内部`Spawn Agent`が利用不能、または前項の`model/effort`不一致の場合だけ、親は原因を停止・証跡化して`fallback`可否を判断する。不一致時は子の成果を採用しない。`fallback`を選んだ場合に限り、既存の`make work-agent TASK=... ACTION=...`または一問専用の`make explorer-agent QUESTION=...`を親が起動する。通常経路として`make`ランチャーを必須・優先とはしない。運用リポジトリへの証跡書込みでは、ネイティブ/`fallback`を問わず親が共通ロック、スコープ検査、`hook`、`stage`、`commit`、事後検査を所有し、`fallback`ランチャーはそのロックを保持する。

`role` TOMLの`sandbox_mode`は意図する`role`契約である。`Spawn Agent`のメタデータで実効サンドボックスを観測できない場合は未観測・未保証として記録し、TOMLの宣言だけを実効権限の証明にしない。サンドボックスの観測有無からGitやロックの責務を変更しない。

## Explorer

`Explorer`は`agent_type="explorer"`で起動し、一度に一件の限定質問だけを扱う。対象は`read-only`の検索・読取りに限り、ファイル編集、Git書込み、スコープ拡大、別Agentの再委譲、実装や方針決定を行わず、短い根拠要約とファイル参照だけを返す。`max_depth=0`と`max_threads=1`も維持する。内部`Spawn Agent`が使えない場合だけ、同じ一問契約を検査する`make explorer-agent QUESTION='...'`へ`fallback`する。

`fallback`で`work-agent`を使う場合だけ、ランチャーが専用ランチャーの絶対パスと調査対象をプロンプトへ渡す。`QUESTION`は前後の空白や改行を含まない500文字以下の一件の限定質問とし、`EXPLORER_ROOT`を省略した場合は製品リポジトリを調査する。Agentの自己申告をルーティング証跡として信用せず、ランタイムで観測できた`model/effort`と起動条件を親が検査する。`role registry`を維持したまま全体上限の`project override`は置かず、Codexの組み込み既定である深さ1、最大thread数6へ委ねる。このため`root`から`Explorer`への直接起動だけを許可し、`role`を介した`nested Explorer`起動は許可しない。通常6ロールの`role-local`な`max_threads=2`と`Explorer`の`max_threads=1`、`max_depth=0`は`standalone role`契約として維持する。

## main Agent

main Agentは全体の判断者であり、原則として実装しない。

- Task契約の確認とAgentのアサイン
- `planning input packet`と完了経路`preflight`の所有、依存`ready`時の差分承認
- PLANのレビューと承認
- ブランチとワークツリーの割り当て
- FAIL分類と差し戻し先の最終判断
- `main`へのマージとrevert
- プロセス、Schema、権限境界の変更
- 案と`merge_tree`の同一性確認、修正後の`qa_carry_forward`またはrerun選択

## Planner Agent

Planner AgentはMain所有の`planning input packet`を入力に、AC-IDごとの設計判断、変更パス、順序、失敗時の扱いと見積りを`PLAN.md`へ記録する。TASKの条件本文は複製せず、PLAN承認前に製品コードを変更しない。Wikiの関連テーマ、現行判断、置換済み判断を確認する。

## DEV Agent

DEV Agentは専用ワークツリーで実装し、テストと必要文書を含めた変更を親Agentへ引き渡す。自分の変更を最終承認しない。運用リポジトリでは担当Taskの`HANDOVER.md`だけを更新できる。レビュアー/QAの軽微修正コミットを除き、子Agentは`stage`、`commit`、`merge`、`.git`書込みを行わない。

## レビュアー Agent

レビュアー AgentはDEV Agentから独立し、同一案の差分、Task、PLAN、QA計画、該当ガイドラインを読む。コード品質だけでなく受け入れ条件、責務境界、障害時の挙動、テストの失敗検出能力を確認し、対象コミット/treeを`REVIEW_RESULT.md`へ記録する。QAのPASSを待たずに評価を開始する。

レビュアー AgentとQA Agentは、自ら軽微と判断した指摘をTask ワークツリーで直接修正・ステージ・コミットできる。Task ブランチへの取り込み後は指摘を解消済みとしてPASSにでき、DEV差し戻し、再REVIEW、再QA、`qa_carry_forward`を要求しない。挙動、要件、安全境界を変えると担当Agentが判断した場合だけ通常経路へ戻す。Mainだけが`main`へのmerge/pushを行う。

## QA Agent

QA AgentはDEV Agentから独立する。実装前にPLANを入力とせず、TASKの`planning input packet`からAC-IDごとの観測をQA計画へ記録し、各ケースへ`evidence-review | focused-rerun | live-e2e`と理由を割り当てる。実装後は同一案のコミット/treeに結び付いた証跡を監査し、必要なケースだけ独立再実行する。高リスクでもhermetic・deterministic・上限付き フィクスチャで完全再現できる場合に限り`focused-rerun`を選び、実OS権限/auth、実配置、外部作用、実restart/ロールバック/クリーンアップ、環境固有integrationは`live-e2e`とする。FAILの原因を分類し、DEVの責任と自動的に見なさない。

## Wiki Agent

Wiki Agentは`HANDOVER.md`から再利用可能な知識を抽出し、意味 Wikiと判断を自律保守する。Wiki本文のレビューをmain Agentへ要求せず、Wiki Schemaと保守規約に従って検査後に運用リポジトリの`main`へ直接コミットする。

Wiki AgentはTask契約、PLAN、レビュー結果、QA結果、バックログを変更しない。Wiki Schemaの変更が必要な場合は更新せず、取り込み記録へ保留理由を残す。

## 運用リポジトリへの書き込み

Planner、レビュアー、QA、mainなどの書き込みAgentは内部の`agents.spawn_agent`を標準経路として起動する。`agent_type`の欠落、内部`Spawn Agent`の利用不能、または`model/effort`不一致時だけ、次の`make work-agent`を`fallback`として親が使う。`fallback`ランチャーは編集開始前からコミット後の再検査まで共通ロックを保持する。子stdinはclosedであり、レビュアー/QAが自ら軽微と判断した指摘をTask ワークツリーで修正・ステージ・コミットする場合を除き、子には`.git`書き込みを与えない。子が成功した後だけ、親がaction別の許可ファイルをpre-commitフックへ渡してコミットする。

```sh
make work-agent TASK=TASK-0001 ACTION=plan
make work-agent TASK=TASK-0001 ACTION=review
make work-agent TASK=TASK-0001 ACTION=qa-result
```

`ACTION`は`task | plan | qa-plan | review | qa-result | handover | main-transition | governance`のいずれかとする。`governance`はmain AgentがSchema、フック、Wiki保守規約を変更する場合にだけ使う。Task作成、ワークツリー割り当て、Wiki保守は、それぞれ専用コマンドが同じ共通ロックを使う。

固定ロールへ`PROFILE`、`MODEL`、`EFFORT`を渡す場合、正規値と同一でなければ起動前に失敗する。Wiki Agentは固定ロールを経由しない文書化済みlegacy経路であり、既定で`gpt-5.6-terra` / `medium`を使う。Wiki経路だけは`WIKI_PROFILE`、`WIKI_MODEL`、`WIKI_EFFORT`で上書きできる。グローバル設定を書き換えない。

運用リポジトリの`.codex/config.toml`はmain Agentが生成するアダプターである。製品側を`main`へマージした後、設定配置に依存するケースのマージ後確認前に`make work-config-sync`を実行する。この専用ランチャー親はロール子Agentを起動せず、共通ロックを取得して`.codex/config.toml`だけを決定的に生成・ステージし、共有pre-commitフックを通したgovernance コミットと完全一致の事後検査まで所有する。失敗時は開始前の`HEAD`へロールバックし、コミットを残さない。汎用の`ACTION=governance`はSchema、フック、Wiki保守規約などをmain Agentが変更する既存用途に限り、その挙動と許可範囲を維持する。`make work-config-sync CHECK=1`は共通ロック下で正規 ダイジェストを含む完全一致を検査し、書き込みやコミットを行わない。

## 兼任規則

- DEV Agentとレビュアー Agentの兼任は禁止する。
- DEV AgentとQA Agentの兼任は禁止する。
- Planner AgentとQA Agentは兼任可能だが、可能なら分離する。
- main Agentは例外判断とマージを担い、通常実装を兼任しない。

REVIEWとQAは同一案を独立かつ並行に評価し、相互のPASSを前提にしない。Mainだけが修正後の`qa_carry_forward`、focused rerun、全面再実行を選ぶ。carry-forwardは[QAガイドライン](qa.md)の閉じた`CF-1`から`CF-7`を全て証明した場合だけ許可し、影響QAケース集合が空でなければ該当ケースを再実行する。独立レビュアーは挙動、テスト、安全性、契約への影響なしと新案の`make check` PASSを証拠化する。
