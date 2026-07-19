# QAガイドライン

## QA計画

QA AgentはDEV開始前に、Taskの受け入れ条件とリスクから`QA_PLAN.md`を作る。

- 受け入れ条件と試験ケースの対応
- 前提、環境、フィクスチャ、データ
- 正常系、境界値、異常系、回帰
- 操作と期待結果
- 必要な証跡
- 実施不能時の扱い

各ケースはDEV開始前に`qa_execution_mode`を一つだけ割り当て、その理由とfail-closed条件を記録する。

| モード | 許可条件 | QAの最低操作 | fail-closed |
|---|---|---|---|
| `evidence-review` | 低リスクで、candidate-bound証跡だけで期待値を判断できる | ケース ID、`candidate_commit`/`candidate_tree`、コマンド/テスト、環境/フィクスチャ、cache条件、exit、成果物 ダイジェスト、未実施理由、ネガティブ検出能力、テスト弱体化の有無を突合する | 欠落・ダイジェスト/コミット不一致・高リスク信号・影響不明はPASSにしない |
| `focused-rerun` | 高リスクでもhermetic・deterministic・上限付き フィクスチャで受け入れ真実を完全再現できる | QAが独立環境で対象コマンド/テストと明示したネガティブ/境界ケースを再実行し、同一案・環境・結果を記録する | 三条件のいずれか不明、実環境境界が必要、影響が不明なら`live-e2e`またはFAILへ昇格 |
| `live-e2e` | 実OS権限/auth（sudo/PAMを含む）、実配置、外部サービス/side effect、実restart/ロールバック/クリーンアップ、環境固有integrationに依存 | 隔離・承認済み実環境で前提、操作、観測、クリーンアップ/ロールバックを実施する | 環境または安全なクリーンアップが用意できなければblocked。別モードのPASSで代替しない |

高リスク信号には認証認可、秘密、sudo/PAM、IPC、Schema、設定、依存、並行性、ライフサイクル、persistence、エラー/fail-closed、install/deploy/config生成、実権限、外部作用、ロールバックを含む。名称だけでモードを決めず、受け入れ真実がどこで再現されるかを根拠にする。

実装後・マージ前に差分とcandidate-bound証跡を読み、認識差を確認する。利用可能なレビュー指摘は参照できるが、レビュー結果またはPASSをQA開始条件にしない。操作手順の修正はQA Agentが行える。期待結果または試験範囲の変更は、変更理由を改訂履歴へ記録し、main Agentの承認を得る。実装を見た後で受け入れ条件を都合よく変更してはならない。

## 実施対象

QA Agentが軽微と判断した指摘は、自らTask ワークツリーで修正・ステージ・コミットできる。Task ブランチへの取り込み後は解消済みとしてPASSにでき、DEV差し戻し、再REVIEW、再QA、`qa_carry_forward`を要求しない。挙動、要件、安全境界を変えると判断した場合だけ通常経路へ戻す。

QAはDEVが固定した同一`candidate_commit`/`candidate_tree`から、レビュアーと相互のPASSを前提にせず独立に開始する。単体テストの再実行だけで受け入れレビューを代替せず、利用者から見える動作、Plane間契約、運用上の完成条件、証跡の完全性を確認する。QA結果には対象案、QA PLAN改訂、モード別の操作、環境、exit、ダイジェスト、未実施理由を残す。

レビュー修正で案が変わったとき、Mainだけが`qa_carry_forward`、focused rerun、全面再実行を選ぶ。

### `qa_carry_forward`の閉じたチェックリスト

Mainは次の全項目を独立に検査できる証拠で満たした場合だけ`qa_carry_forward`を選べる。一つでも満たさない、または不明ならcarry-forwardを禁止し、影響ケースを再実行する。影響ケースを限定できなければ全面再実行とする。

- `CF-1`: 旧QA結果がPASSで、記録された旧`candidate_commit`/`candidate_tree`に正しく束縛されている。
- `CF-2`: 旧案と新案の全差分を列挙し、差分ダイジェストを記録している。
- `CF-3`: 変更パスと内容が、実行されない誤字、空白、コメント、リンク、証跡メタデータだけに限定される。製品挙動、ランタイム、テスト、Schema、設定、依存、生成物、外部公開契約または安全契約、受け入れ条件、QA_PLANの意味を変更していない。
- `CF-4`: 影響QAケース集合が空である。空でなければcarry-forwardせず、該当ケースを再実行する。
- `CF-5`: 独立レビュアーが挙動、テスト、安全性、契約への影響なしを確認し、新案で`make check`がPASSしている。
- `CF-6`: QA FAIL、受け入れ条件/QA_PLAN変更、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/ライフサイクル/persistence/エラー/fail-closed、テスト削除/弱体化、影響不明、証跡と評価対象の案/tree不一致が全て偽である。
- `CF-7`: Mainが旧新コミット/tree、全差分とダイジェスト、空の影響ケース集合、レビュアー確認、`make check`証拠、carry-forward理由を記録している。

マージ後にMainは`merge_tree == candidate_tree`を確認する。同一かつ環境依存ケースなしなら全面確認を繰り返さない。install/deploy/config生成、実権限、外部作用、ロールバック等の環境依存ケースはマージ後にも限定確認する。

## FAIL分類

| 分類 | 意味 | 標準差し戻し先 |
|---|---|---|
| `implementation_defect` | 合意済みTaskまたはPLANから実装が逸脱 | DEV |
| `qa_plan_defect` | QAの期待値、手順、前提が誤り | QA |
| `requirement_gap` | TaskまたはPLANが曖昧、不足、矛盾 | PLAN |
| `environment_issue` | 環境、データ、試験基盤の問題 | QAまたは基盤Task |
| `regression` | 既存動作を破壊 | DEV、必要ならrevert |

分類と差し戻し先の最終判断はmain Agentが行い、理由を`QA_RESULT.md`または`HANDOVER.md`へ残す。

## revertとバグ化

次は原則revertする。

- `main`がビルド、起動、主要フローを維持できない。
- セキュリティ、データ破損、権限逸脱がある。
- 主要な受け入れ条件を満たさない。
- 影響範囲が不明で、安全に隔離できない。

影響が限定され、mainを利用可能に保て、回避策と追跡がある場合はバグTask化できる。バグTaskは`origin_task`、再現手順、期待と実際、影響、暫定策、優先度を持つ。

## PASSと完了

- 全受け入れ条件の結果と証跡がある。
- 未実施項目と理由が明示され、main Agentが許容している。
- 発見事項が解消、revert、またはバグTask化されている。
- `HANDOVER.md`に成果、既知制約、運用注意、Wiki 取り込み材料がある。
- すべてのケースでモード、案 割り当て、結果、未実施またはblocked理由があり、Mainのcarry-forwardまたは再実行判断が記録されている。
