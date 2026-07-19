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

## 2. Task起票

main AgentまたはTask起票担当は次を行う。

1. `TASK-NNNN`を採番し、Taskディレクトリと6つの証跡ファイルを生成する。
2. `TASK.md`へ目的、受け入れ条件、設計観点、完成の定義を記載する。
3. Epic、優先度、依存Taskを`backlog.yaml`へ登録する。
4. Wiki Agentへ関連コンテキストを問い合わせ、関連テーマと判断を`TASK.md`へ記録する。
5. `status: plan`へ進め、Planner Agentをアサインする。

## 3. PLAN

Planner Agentはコードを変更せず、Task契約と関連Wikiを基に`PLAN.md`を作成する。

- 受け入れ条件を検証可能な形へ具体化する。
- 設計選択、代替案、境界、不変条件、移行、障害時の挙動を検討する。
- 変更予定の実装コード、Schema、設定ファイルと概算変更行数を列挙する。
- テスト、フィクスチャ、スナップショット、文書、生成物、ロックファイル、vendorを除外して規則ベースの見積もりポイントを算出する。
- QA Agentが実装前の`QA_PLAN.md`を作成できるだけの期待動作を明示する。
- DEV プロファイルを`luna-xhigh`または`sol-high`から選び、理由とリスクシグナルをフロントマターへ記録する。`Luna`は局所的、明確、機械検証可能でリスクシグナルがない場合だけ選択できる。高リスク、横断的、不明な場合は`Sol`を選択する。

main AgentはPLANをレビューし、曖昧な受け入れ条件、未解決の設計判断、過大なTaskが残る場合は承認しない。承認後にDEV Agent、レビュアー Agent、QA Agent、トピックブランチ、ワークツリーを割り当てる。

QA AgentはDEV開始前に`QA_PLAN.md`を作成する。実装後の再確認で試験手順は修正できるが、期待結果または試験範囲の変更にはmain Agentの承認が要る。

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

運用リポジトリへ証跡を書くロールAgentは編集だけを行い、ステージ、コミット、`.git`書き込みを行わない。ランチャー親は共通ロックを保持したまま、子exit 0、許可スコープ、フック、変更前後の`validate-work`がすべて成功した場合だけコミットする。失敗時の証跡は`commit:null`とし、部分コミットしない。子stdinはclosedであり、会話全文や未加工 ログを保存しない。

レビュアー AgentはDEV Agentと別でなければならない。P0またはP1が残る場合、検査未完了の場合、受け入れ条件の根拠が不足する場合はPASSにしない。結果は対象`candidate_commit`/`candidate_tree`、実行したコマンド、結果、残存リスクとともに`REVIEW_RESULT.md`へ記録する。レビュー修正で案が変わった場合、MainはQAの対象を新案へ再束縛する。

レビュアー/QA Agentが軽微と判断した指摘は、担当AgentがTask ワークツリーで直接修正・ステージ・コミットできる。Task ブランチへの取り込み後は解消済みとしてPASSにでき、DEV差し戻し、再REVIEW、再QA、`qa_carry_forward`を要求しない。挙動、要件、安全境界を変えると担当Agentが判断した場合だけ通常経路へ戻す。Mainだけが`main`へのmerge/pushを所有する。

Main AgentはREVIEWとQAの結果、対象案、`make check`、QA計画再確認を確認し、`--no-ff`で`main`へマージする。レビュー修正後のQA省略は、[QAガイドライン](qa.md)の閉じた`CF-1`から`CF-7`を全て満たす場合に限り、Mainが`qa_carry_forward`として承認する。影響QAケース集合が空でなければ該当ケースを再実行し、限定できなければ全面再実行とする。QA FAIL、受け入れ条件/QA_PLAN変更、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/ライフサイクル/persistence/エラー/fail-closed、テスト削除/弱体化、影響不明、証跡と評価対象の案/tree不一致はcarry-forward禁止である。Mainは旧新コミット/tree、全差分とダイジェスト、空の影響ケース集合、独立レビュアーの確認、新案の`make check`証拠、理由を記録する。`merge_tree`と承認`candidate_tree`を比較し、一致しない場合は結果を持ち越さず影響を再評価する。ルーティング設定を変更したTaskでは、専用ランチャー親が共通ロック下で`make work-config-sync`を生成からgovernanceコミット・事後検査まで所有する。失敗時は開始前`HEAD`へロールバックし、drift検査が失敗する間は次のゲートへ進めない。

## 5. REVIEWとQA

QA AgentはDEV開始前に作成したQA計画を、実装差分とcandidate-boundな証跡に照らして再確認する。QAはケースごとの`qa_execution_mode`に従い、同一案から独立に`evidence-review`、`focused-rerun`、または`live-e2e`を実施する。`evidence-review`は証跡だけの自己申告を受け入れず、コミット/tree 割り当て、テスト弱体化、ネガティブ ケース、完全性を監査する。高リスクでもhermetic・deterministic・上限付き フィクスチャで受け入れ真実を完全再現できる場合だけ`focused-rerun`を使う。実OS権限/auth（sudo/PAMを含む）、実配置、外部作用、実restart/ロールバック/クリーンアップ、環境固有integrationに依存するケースは`live-e2e`とし、環境またはクリーンアップが不明ならblockedのままPASSにしない。

マージ後は`merge_tree == candidate_tree`をMainが確認する。一致し、環境依存ケースがない場合は全面的な重複確認を省略できる。環境依存ケース（install/deploy/config生成、実権限、外部作用、ロールバック等）はマージ後もケース単位で確認する。未実施項目、blocked理由、carry-forwardまたは再実行の判断は`QA_RESULT.md`と`HANDOVER.md`へ記録する。

FAILは`implementation_defect | qa_plan_defect | requirement_gap | environment_issue | regression`へ分類する。差し戻し先やrevertの最終判断はmain Agentが行う。

- mainを壊す、セキュリティ問題、データ破損、主要受け入れ条件未達: 原則revertしてDEVへ戻す。
- 影響が限定的でmainを利用可能に保てる: `type: bug`のTaskを起票する。
- QA計画または試験手順の誤り: QAへ戻して再試験する。
- TaskまたはPLANの曖昧さ: PLANへ戻して合意し直す。

PASSまたはバグ化によって元Taskを閉じられる場合、`QA_RESULT.md`と`HANDOVER.md`を完成させる。Wiki AgentがHANDOVERを取り込んでダイジェスト付きreceiptを直接コミットした後、main Agentが`status: done`をコミットし、ワークツリーとトピックブランチを削除する。

## 6. ブートストラップ例外

プロセス自体を初めて導入するTaskだけは、既存プロセスが存在しないため証跡を後から整備できる。例外理由、実施した代替レビュー、未適用ゲートを`HANDOVER.md`へ明記する。2件目以降のTaskには適用しない。
