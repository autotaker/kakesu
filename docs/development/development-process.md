# 開発プロセス

## 1. 原則

開発単位はTaskである。Taskは目的、受け入れ条件、設計観点、完成の定義を持ち、`PLAN / DEV / REVIEW+QA / merge確認`のゲートを通過する。詳細な待ち状態をバックログへ増やさず、フェーズ内の進捗は証跡ファイルのチェック項目で表す。DEVが固定する評価対象を`candidate_commit`と`candidate_tree`、マージ結果を`merge_tree`と呼ぶ。

```text
backlog
  → plan
  → dev
  → review + qa（同一candidateから独立に並行）
  → merge確認
  → done

任意のフェーズ → blocked
review/qa → dev       実装不具合またはrevert
review/qa → plan      要件や設計の不足
review/qa → qa        QA計画または試験手順の不具合
merge確認 → review/qa candidate/treeの不一致または環境依存確認
```

`status`は`backlog | plan | dev | qa | blocked | done | cancelled`だけを使用する。`blocked`では`resume_status`に復帰先を記録し、復帰先フェーズの承認、担当分離、ブランチ、ワークツリー、コミット証拠などの不変条件を維持する。ブロック理由が不変条件そのものの欠落である場合は、欠落を許容せず、復帰先を前フェーズへ戻して記録する。

`backlog.yaml`の`change_class`は`product | safety_contract`だけを使用する。フィールドがない既存Taskは`product`として扱い、未知値はfail-closedする。`safety_contract`は製品成果物を変更しないTaskだけに指定し、Task契約の対象外宣言、独立計画レビュー、Main承認、実際のmerge差分を照合する。分類を変更する場合はTask、PLAN、QA_PLANを再承認し、理由と時刻を記録する。

## 2. Task起票

main AgentまたはTask起票担当は次を行う。

1. `TASK-NNNN`を採番し、Taskディレクトリと6つの証跡ファイルを生成する。
2. `TASK.md`の`planning input packet`へ目的、対象外、AC-ID付き受け入れ条件、安定した参照、依存状態、許可パス、`preflight`結果、未決事項を記載する。この`packet`をPlannerとQAへ同じ内容で渡し、ほかの証跡へ本文を複製しない。
3. Epic、優先度、依存Taskを`backlog.yaml`へ登録する。
4. Wiki Agentへ関連コンテキストを問い合わせ、関連テーマと判断を`TASK.md`へ記録する。
5. `status: plan`へ進め、Planner Agentをアサインする。

## 3. PLAN

main Agentは計画開始前に完了経路を`preflight`する。完了checker、必要な権限、依存の状態と参照、生成物の有無と更新方法、割当ワークツリー、`Lap`ログの書込・Schema検証・`repository annotation`を実際のコマンドまたは参照で確認し、結果を`planning input packet`へ記録する。`Lap`ログは既存Schema/JSONLを変更せず、開始記録を書いて検証できた後だけ開始済みとする。未解決の`preflight`があれば計測を開始せず、`not_started`または`blocked`としてMainへ戻す。

Planner Agentはコードを変更せず、`planning input packet`と関連Wikiを基に`PLAN.md`を作成する。TASKの条件本文は再掲せず、各AC-IDに設計判断、変更パス、実施順序、失敗時の扱いを対応させ、変更予定と見積りを記録する。DEVプロファイルは`luna-xhigh`または`sol-high`から選び、理由とリスクシグナルをフロントマターへ記録する。`Luna`は局所的、明確、機械検証可能でリスクシグナルがない場合だけ選択でき、高リスク、横断的、不明な場合は`Sol`を選択する。

QA AgentはPLANを入力にせず、同じTASKのpacketからDEV開始前に`QA_PLAN.md`を独立作成する。条件本文や設計を再掲せず、各AC-IDに観測方法、実施モード、必要証跡、fail-closed条件を対応させる。実装後の再確認で試験手順は修正できるが、期待結果または試験範囲の変更にはmain Agentの承認が要る。

安全契約のv2完了契約はPLANフロントマターの`safety_contract_version: 2`で明示的に選ぶ。`safety_contract_planned_paths`と`safety_contract_generated_paths`をリポジトリ相対ファイルパス配列として必ず記録し、変更しない種別は空配列にする。承認後、DEV開始前に`make task-preflight TASK=TASK-NNNN`を実行し、許可外パス、欠落、配列内または配列間の重複をfail-closedで解消する。バージョン未指定の既存安全契約はこの新契約を暗黙に要求されない。

依存が未`ready`でも、安定した参照に基づく`dependency-independent planning`だけは進められる。依存待ちは`active planning`時間と分けて記録し、依存前に固定できない値を推測で埋めない。依存が`ready`になったらMainは固定参照と差分を`dependency-ready reconciliation`として`packet`へ追記する。差分がAC、設計、スコープ、QAの期待または範囲を変える場合は、DEV前にPLANとQA_PLANを再承認する。

`active planning`が10分を超えたら、`境界不明 | 依存不安定 | API不明 | 資料不足 | 期待不一致 | tool/permission`から原因を記録し、文章の磨き込みだけを続けず、Mainへ不足解消またはblocked判断を返す。dependency待機はこの10分に算入しない。

main AgentはPLANと`TASK-first` QA_PLANをレビューし、`packet`の欠落や矛盾、未解決の設計判断、過大なTask、未完了の`preflight`または`reconciliation`が残る場合はDEVを承認しない。承認後にDEV Agent、レビュアー Agent、QA Agent、トピックブランチ、ワークツリーを割り当てる。

## 4. DEV

DEV Agentは割り当てられたワークツリーだけで実装し、Task外の変更を混在させない。

1. 承認済みPLANとQA計画を確認する。QA計画には全ケースの`qa_execution_mode`（`evidence-review | focused-rerun | live-e2e`）と理由がある。
2. 該当する言語別ガイドラインと配下の`AGENTS.md`を確認する。
3. 実装、テスト、文書更新を行う。
4. `make check`を実行する。
5. 親Agentが変更スコープを確認して案の`candidate_commit`/`candidate_tree`を固定する。
6. 各ケースについてケース ID、コマンド/テスト、環境またはフィクスチャ、cache条件、exit、成果物 ダイジェスト、未実施理由を`HANDOVER.md`へ記録し、同じ案へ結び付ける。
7. レビュアー AgentとQA Agentへ同一案の差分と証跡を渡す。両者は相互のPASSを待たず独立に評価する。

DEV中に新しいリスクシグナルを発見した場合、`Luna` Agentは変更を止める。`luna-xhigh`から`sol-high`への変更についてシグナル、理由、main Agentの承認者と時刻をPLANへ記録し、承認後に`Sol` Agentで再開する。`Sol`から`Luna`への降格は行わない。

main管理証跡を書くロールAgentは編集だけを行い、ステージ、コミット、`.git`書き込みを行わない。ただし、レビュアー/QA Agentは自ら軽微と判断した指摘をTask ワークツリーで修正・ステージ・コミットできる。Mainは共通ロックを保持したまま、許可スコープ、フック、`validate-work`がすべて成功した場合だけ`make evidence-commit`で公開する。失敗時は部分コミットを続行しない。

レビュアー AgentはDEV Agentと別でなければならない。P0またはP1が残る場合、検査未完了の場合、受け入れ条件の根拠が不足する場合はPASSにしない。結果は対象`candidate_commit`/`candidate_tree`、実行したコマンド、結果、残存リスクとともに`REVIEW_RESULT.md`へ記録する。レビュー修正で案が変わった場合、MainはQAの対象を新案へ再束縛する。

レビュアー/QA Agentが軽微と判断した指摘は、担当AgentがTask ワークツリーで直接修正・ステージ・コミットできる。Task ブランチへの取り込み後は解消済みとしてPASSにでき、DEV差し戻し、再REVIEW、再QA、`qa_carry_forward`を要求しない。挙動、要件、安全境界を変えると担当Agentが判断した場合だけ通常経路へ戻す。Mainだけが`main`へのmerge/pushを所有する。

Main AgentはREVIEWとQAの結果、composite 案、`make check`、QA計画再確認を確認する。両PASS後に`make task-pr`で`ready` PRを作成し、merge コミット方式のauto-mergeを有効にする。レビュー修正後のQA省略は、[QAガイドライン](qa.md)の閉じた`CF-1`から`CF-7`を全て満たす場合に限り、Mainが`qa_carry_forward`として承認する。影響QAケース集合が空でなければ該当ケースを再実行し、限定できなければ全面再実行とする。QA FAIL、受け入れ条件/QA_PLAN変更、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/ライフサイクル/persistence/エラー/fail-closed、テスト削除/弱体化、影響不明、証跡と評価対象の案/tree不一致はcarry-forward禁止である。Mainは旧新コミット/tree、全差分とダイジェスト、空の影響ケース集合、独立レビュアーの確認、新案の`make check`証拠、理由を記録する。マージ後は承認案とのtree同一性を確認し、一致しない場合は結果を持ち越さず影響を再評価する。

## 5. REVIEWとQA

QA AgentはDEV開始前に作成したQA計画を、実装差分とcandidate-boundな証跡に照らして再確認する。QAはケースごとの`qa_execution_mode`に従い、同一案から独立に`evidence-review`、`focused-rerun`、または`live-e2e`を実施する。`evidence-review`は証跡だけの自己申告を受け入れず、コミット/tree 割り当て、テスト弱体化、ネガティブ ケース、完全性を監査する。高リスクでもhermetic・deterministic・上限付き フィクスチャで受け入れ真実を完全再現できる場合だけ`focused-rerun`を使う。実OS権限/auth（sudo/PAMを含む）、実配置、外部作用、実restart/ロールバック/クリーンアップ、環境固有integrationに依存するケースは`live-e2e`とし、環境またはクリーンアップが不明ならblockedのままPASSにしない。

マージ後は`merge_tree == candidate_tree`をMainが確認する。一致し、環境依存ケースがない場合は全面的な重複確認を省略できる。環境依存ケース（install/deploy/config生成、実権限、外部作用、ロールバック等）はマージ後もケース単位で確認する。未実施項目、blocked理由、carry-forwardまたは再実行の判断は`QA_RESULT.md`と`HANDOVER.md`へ記録する。

FAILは`implementation_defect | qa_plan_defect | requirement_gap | environment_issue | regression`へ分類する。差し戻し先やrevertの最終判断はmain Agentが行う。

- mainを壊す、セキュリティ問題、データ破損、主要受け入れ条件未達: 原則revertしてDEVへ戻す。
- 影響が限定的でmainを利用可能に保てる: `type: bug`のTaskを起票する。
- QA計画または試験手順の誤り: QAへ戻して再試験する。
- TaskまたはPLANの曖昧さ: PLANへ戻して合意し直す。

製品変更をPASSまたはバグ化によって閉じられる場合、`QA_RESULT.md`と`HANDOVER.md`を完成させる。Wiki AgentがHANDOVERを取り込んでダイジェスト付きreceiptを直接コミットした後、main Agentが`status: done`をコミットし、ワークツリーとトピックブランチを削除する。

安全契約変更の完了判定は、承認済みPLANとTASK-first QA PLAN、独立計画レビューのPASS、Mainの分類承認、対象検査のPASS、no-ff merge、第2親の案 treeとmerge treeの一致、許可された統制文書差分を要求する。v2では候補差分の全パスをPLANの予定パスと生成パスの和集合へ束縛し、宣言した生成パスの全てが削除でない候補差分として存在することも要求する。生成専用パスとして許可するのは`docs/99-glossary-index.md`だけである。PLANとQA PLANの`change_class`は`safety_contract`とし、`planning_reviewed_by`は担当レビュアーに一致させる。`PLAN.md`には`planning_review_decision`、`planning_reviewed_at`、`classification_approved_by`、`classification_approved_at`、空でない`classification_approval_reason`も記録し、計画レビュー、PLAN、QA PLAN、分類承認の時刻順を維持する。`HANDOVER.md`の`safety_checks`は`process_tests`、`contract_scope`、`docs_lint`、`make_check`だけを持ち、すべて`pass`とする。`safety_check_digest`は`safety_candidate_tree`、`safety_merge_tree`、上記順の検査名と結果を`key=value`の改行区切りで正規化し、末尾改行を含めてSHA-256を計算する。名前変更またはコピーを含む差分は安全契約経路で拒否する。製品用のREVIEW/QA PASS、製品用の完了HANDOVER、Wiki取込記録は作成しない。

## 6. ブートストラップ例外

プロセス自体を初めて導入するTaskだけは、既存プロセスが存在しないため証跡を後から整備できる。例外理由、実施した代替レビュー、未適用ゲートを`HANDOVER.md`へ明記する。2件目以降のTaskには適用しない。
