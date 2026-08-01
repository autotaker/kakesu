# 開発プロセス

開発単位はTaskである。製品変更は `PLAN → QA_PLAN → DEV → 独立REVIEW/QA → completion` の順で進める。Task証跡はmain ワークツリー、製品変更はTask ワークツリーで編集する。

## 計画

MainがTask 雛形、ブランチ、ワークツリーを作成する。`task-start`はmainへコミットしない。PlannerとQAはTASKのplanning 入力 packetだけを入力にし、承認済みのPLANとQA_PLANを作成する。`planning-gate`は変更された計画部分集合を最終状態で検証し、mainへ一回のplanning コミットを作り、Task ブランチ/ワークツリーをそのコミットへ分岐なしで進める。

## DEV

DEVは承認済みPLANの範囲でTask ワークツリーだけを編集する。`candidate-commit`がロックを取得し、変更前のworking バイトを固定して一度だけ`make check`を実行し、その後に製品差分だけの案 コミットを作る。Task ブランチの手動コミットはpre-commit hookで拒否する。案 ブランチはHANDOVERを変更しない。

## REVIEWとQA

レビュアーとQAは同じ案から独立に開始し、相互のPASSを前提にしない。REVIEW_RESULTはレビュアー 識別情報、判断、DEV `make check`の監査事実を記録する。QA_RESULTはQA 識別情報、判断、実行ケースのコマンドと結果だけを記録する。案の識別子はHANDOVERの`candidate_commit`だけを使用する。

## 完了

Main側のHANDOVERに案 コミットを記録し、REVIEW/QA PASSと承認済みQA_PLANを揃える。`completion-gate`はpre-staged変更を拒否し、main側の証跡バイトを保存したうえで`git merge --no-ff --no-commit`を実行する。案 diffは製品差分だけ、merge後のmain-managed差分はTask証跡と許可されたWikiだけに限定し、検証後に一つのmerge コミットを作る。失敗時はmergeをabortし、保存した証跡とインデックスを復元する。

完了後の分岐を残すmergeは、HANDOVERの`candidate_commit`を第2親に持つmainのmerge コミットからGitで導出する。Wiki receiptや追加の完了コミットは標準完了条件ではない。

## 安全契約変更

製品成果物を変更しない安全契約変更は、TASK本文だけから独立QA_PLANと計画レビューを作る。製品用のDEV/REVIEW/QA PASSを代用せず、契約に必要な統制文書だけを検査する。既存安全契約Taskの専用検査は後方互換のため維持する。

既存の振り返りで10 Taskごとにルールの誤検知、検出価値、時間、保守費を見直す。低価値ルールは削除または警告化し、専用checklistやバージョン フィールドを追加しない。

新しい必須ルール/ゲートは、反復した具体的failure、または単発でも重大なセキュリティ・permission・不可逆復旧failureを直接検出する場合だけ追加する。
