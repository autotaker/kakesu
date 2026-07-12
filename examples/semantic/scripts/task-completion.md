---
kind: script
title: Task Completion
---

# Task Completion

Ownerが作業完了を判断してから、Taskが`completed`へ入るまでの典型的な進行である。([Completion Review導入Episode](episode://design/completion-review))

## Trigger

Ownerが現行ContractのObjectiveを達成し、Acceptanceを満たすOutcomeとEvidenceが揃ったと判断する。

## 標準進行

1. OwnerがCompletion Candidateを提出する。
2. HarnessがOwner、Contract バージョン、required child、Artifact参照を検査する。
3. HarnessがOutcomeとEvidenceのダイジェストを固定する。
4. 独立Acceptance Reviewerを起動する。
5. Reviewerが`accept`、`reject`、`insufficient_evidence`を返す。
6. HarnessがTask状態を遷移する。

## 分岐

### Accept

`reviewing_completion → completed`。親Task Mailboxへ`ChildTaskCompleted`を送る。

### Reject

`reviewing_completion → running`。Ownerへ未達Acceptanceを返す。

### Insufficient evidence

`reviewing_completion → running`。Ownerが追加Evidenceを収集・提出して再Reviewする。Evidence取得に非同期結果が必要な場合だけ、対応する`WaitCondition`で`waiting`へ遷移する。

## 典型的な失敗

- Ownerの説明だけでOutcomeがない。
- 古いContract バージョンで完了候補を出す。
- required childが実行中のまま完了を試みる。
- Reviewerが新要件を追加する。
- 子Taskの完了を親Taskの完了と誤認する。

## 関連

- [Task](../concepts/task.md)
- [Task Ownership](../schemas/task-ownership.md)
