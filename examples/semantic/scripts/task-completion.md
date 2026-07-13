---
kind: script
title: Task Completion
---

# Task 完了

オーナーが作業完了を判断してから、Taskが`completed`へ入るまでの典型的な進行である。([完了レビュー導入エピソード](episode://design/completion-review))

## トリガー

オーナーが現行契約の目的を達成し、受け入れ条件を満たす結果と証跡が揃ったと判断する。

## 標準進行

1. オーナーが完了案を提出する。
2. ハーネスがオーナー、契約バージョン、必須 子、成果物参照を検査する。
3. ハーネスが結果と証跡のダイジェストを固定する。
4. 独立受け入れ条件レビュアーを起動する。
5. レビュアーが`accept`、`reject`、`insufficient_evidence`を返す。
6. ハーネスがTask状態を遷移する。

## 分岐

### 受理

`reviewing_completion → completed`。親Taskメールボックスへ`ChildTaskCompleted`を送る。

### 拒否

`reviewing_completion → running`。オーナーへ未達受け入れ条件を返す。

### Insufficient 証跡

`reviewing_completion → running`。オーナーが追加証跡を収集・提出して再レビューする。証跡取得に非同期結果が必要な場合だけ、対応する`WaitCondition`で`waiting`へ遷移する。

## 典型的な失敗

- オーナーの説明だけで結果がない。
- 古い契約バージョンで完了候補を出す。
- 必須 子が実行中のまま完了を試みる。
- レビュアーが新要件を追加する。
- 子Taskの完了を親Taskの完了と誤認する。

## 関連

- [Task](../concepts/task.md)
- [Task 所有権](../schemas/task-ownership.md)
