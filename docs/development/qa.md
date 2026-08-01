# QAガイドライン

QAは承認済みQA_PLANに従い、DEVが作った同じ案から独立に実施する。結果はQA_RESULTへケースID、コマンド/テスト、結果を記録し、未実施の場合だけ理由を追記する。

## 実施モード

| モード | 用途 |
|---|---|
| `evidence-review` | 案の証跡とレビュー結果を独立監査する |
| `focused-rerun` | 上限付きで決定的なフィクスチャを再実行する |
| `live-e2e` | 実OS権限、配置、外部作用、restart/ロールバック等を確認する |

環境や安全なクリーンアップがないlive-e2eはblockedとし、別モードのPASSで置き換えない。高リスク、証跡不足、影響不明もPASSにしない。

## 完了前確認

MainはREVIEW/QAの識別情報と判断、承認済みQA_PLAN、HANDOVERのcandidate_commitを確認する。案 ブランチのdiffが製品差分だけであり、案側HANDOVERを変更していないことをGitから検証する。

## マージ後確認

`completion-gate`のmerge コミットは案を第2親に持つ分岐を残すmergeでなければならない。環境依存ケースだけをマージ後に確認し、完了後のmergeと候補の関係はHANDOVERとGit履歴から導出する。Wiki receiptや案 tree/ダイジェストの転記は要求しない。
