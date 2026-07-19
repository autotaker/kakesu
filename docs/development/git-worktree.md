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
- DEV完了時に`candidate_commit`と`candidate_tree`を固定し、同じtreeをREVIEWとQAへ渡す。案を変更した場合は旧結果を自動で再利用しない。

## マージ

main Agentは`REVIEW_RESULT.md`と`QA_RESULT.md`が同一案を対象とし、`make check`、QA計画の再確認、cleanなワークツリーを確認する。修正後のcarry-forwardは非挙動かつ明示した低リスク条件を全て証明し、`qa_carry_forward`としてMainが旧新コミット/tree、diff、影響ケース、再実行証拠、理由を記録した場合だけ許可する。禁止条件または影響不明があればrerunへfail-closedする。

```sh
git switch main
git merge --no-ff task/TASK-0001-example
```

マージコミットでTask境界を残す。squash mergeは標準手順にしない。マージ後に実際の`merge_tree`と承認案 treeを比較し、一致しない場合はQA結果を持ち越さず影響ケースを再評価する。一致して環境依存ケースがない場合は全面確認を重複させないが、install/deploy/config生成、実権限、外部作用、ロールバック等の環境依存ケースはケース単位で確認する。

## 保持と削除

ワークツリーとブランチはマージ直後に削除せず、案と`merge_tree`の照合および環境依存ケースの確認が完了するまで保持する。

- QA PASSまたはバグ化で元Taskを閉じた: ワークツリーを削除し、マージ済みブランチを削除する。
- revertしてDEVへ戻す: 同じワークツリーとブランチを再利用する。
- QAまたはPLANへ戻す: 製品変更が必要になるまでワークツリーを保持する。

終了後は`make worktree-remove TASK=TASK-0001`を使う。`done`ではブランチが製品`main`へマージ済みであることを検査してから削除する。
