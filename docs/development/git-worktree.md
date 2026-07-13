# Gitとワークツリー

## 命名

- Task ID: `TASK-NNNN`
- ブランチ: `task/TASK-NNNN-short-slug`
- ワークツリー: `../agent-harness-work/worktrees/TASK-NNNN-short-slug`

## 作成

ワークツリーはPLAN承認後にmain Agentが作成する。開始点は最新の`main`とし、同じTask IDのブランチまたはワークツリーが既にある場合は自動上書きしない。

```sh
make worktree-create TASK=TASK-0001
```

このコマンドは運用リポジトリの共通ロックを取り、登録済みAgentの分離、承認済みPLANとQA計画を検査してから、ブランチとワークツリーを作成する。バックログの割り当ても同じ操作で`main`へコミットする。

## DEV中

- DEV Agentは割り当てられたワークツリー以外で製品コードを変更しない。
- コミットは論理的な変更単位で作る。
- 無関係な整形、生成物、ローカル設定を混ぜない。
- main Agent以外は`main`を更新しない。
- force push、履歴改変、破壊的resetを通常手順にしない。

## マージ

main Agentは`REVIEW_RESULT.md`のPASS、`make check`、QA計画の再確認、cleanなワークツリーを確認する。

```sh
git switch main
git merge --no-ff task/TASK-0001-example
```

マージコミットでTask境界を残す。squash mergeは標準手順にしない。

## 保持と削除

ワークツリーとブランチはマージ直後に削除せず、QA完了まで保持する。

- QA PASSまたはバグ化で元Taskを閉じた: ワークツリーを削除し、マージ済みブランチを削除する。
- revertしてDEVへ戻す: 同じワークツリーとブランチを再利用する。
- QAまたはPLANへ戻す: 製品変更が必要になるまでワークツリーを保持する。

終了後は`make worktree-remove TASK=TASK-0001`を使う。`done`ではブランチが製品`main`へマージ済みであることを検査してから削除する。
