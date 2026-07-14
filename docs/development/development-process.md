# 開発プロセス

## 1. 原則

開発単位はTaskである。Taskは目的、受け入れ条件、設計観点、完成の定義を持ち、`PLAN / DEV / QA`の3フェーズを通過する。詳細な待ち状態をバックログへ増やさず、フェーズ内の進捗は証跡ファイルのチェック項目で表す。

```text
backlog
  → plan
  → dev
  → qa
  → done

任意のフェーズ → blocked
qa → dev       実装不具合またはrevert
qa → plan      要件や設計の不足
qa → qa        QA計画または試験手順の不具合
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

1. 承認済みPLANとQA計画を確認する。
2. 該当する言語別ガイドラインと配下の`AGENTS.md`を確認する。
3. 実装、テスト、文書更新を行う。
4. `make check`を実行する。
5. 親Agentが変更スコープを確認してコミットした後、レビュアー Agentへコミット済み差分と証跡を渡す。
6. レビュアーの指摘を修正し、再検査を依頼する。

DEV中に新しいリスクシグナルを発見した場合、`Luna` Agentは変更を止める。`luna-xhigh`から`sol-high`への変更についてシグナル、理由、main Agentの承認者と時刻をPLANへ記録し、承認後に`Sol` Agentで再開する。`Sol`から`Luna`への降格は行わない。

運用リポジトリへ証跡を書くロールAgentは編集だけを行い、ステージ、コミット、`.git`書き込みを行わない。ランチャー親は共通ロックを保持したまま、子exit 0、許可スコープ、フック、変更前後の`validate-work`がすべて成功した場合だけコミットする。失敗時の証跡は`commit:null`とし、部分コミットしない。子stdinはclosedであり、会話全文や未加工 ログを保存しない。

レビュアー AgentはDEV Agentと別でなければならない。P0またはP1が残る場合、検査未完了の場合、受け入れ条件の根拠が不足する場合はPASSにしない。結果は`REVIEW_RESULT.md`へ記録する。

main AgentはレビューPASS、`make check`成功、QA計画再確認を確認し、`--no-ff`で`main`へマージする。レビュー対象コミットとマージコミットを証跡へ記録し、ゲートが両コミットの祖先関係を検査する。ルーティング設定を変更したTaskでは、マージ後QAの前に`make work-config-sync`を実行する。専用ランチャー親が共通ロックを生成前からgovernance コミットと事後検査の完了まで保持し、共有フックへ`.codex/config.toml`だけの許可範囲を渡す。ロール子Agentや汎用`ACTION=governance`にはアダプターを書かせない。失敗時は開始前`HEAD`へロールバックし、drift検査が失敗する間はQAへ進めない。

## 5. QA

QA Agentはマージ済み`main`を対象に受け入れレビューを実施する。実装前に作成したQA計画を、実装差分とレビュー結果に照らして再確認してから実行する。

FAILは`implementation_defect | qa_plan_defect | requirement_gap | environment_issue | regression`へ分類する。差し戻し先やrevertの最終判断はmain Agentが行う。

- mainを壊す、セキュリティ問題、データ破損、主要受け入れ条件未達: 原則revertしてDEVへ戻す。
- 影響が限定的でmainを利用可能に保てる: `type: bug`のTaskを起票する。
- QA計画または試験手順の誤り: QAへ戻して再試験する。
- TaskまたはPLANの曖昧さ: PLANへ戻して合意し直す。

PASSまたはバグ化によって元Taskを閉じられる場合、`QA_RESULT.md`と`HANDOVER.md`を完成させる。Wiki AgentがHANDOVERを取り込んでダイジェスト付きreceiptを直接コミットした後、main Agentが`status: done`をコミットし、ワークツリーとトピックブランチを削除する。

## 6. ブートストラップ例外

プロセス自体を初めて導入するTaskだけは、既存プロセスが存在しないため証跡を後から整備できる。例外理由、実施した代替レビュー、未適用ゲートを`HANDOVER.md`へ明記する。2件目以降のTaskには適用しない。
