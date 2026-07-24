# Gitとワークツリー

## 命名

- Task ID: `TASK-NNNN`
- ブランチ: `task/TASK-NNNN-short-slug`
- ワークツリー: `worktrees/TASK-NNNN-short-slug`

## 作成

main Agentはcleanかつ最新の`main`からTask起票と同じトランザクションでワークツリーを作成する。同じTask IDのブランチまたはワークツリーが既にある場合は自動上書きしない。DEVはPLANとQA_PLANの承認まで開始しない。

```sh
make task-start ID=TASK-0001 SLUG=short-slug TITLE='title'
```

このコマンドはmain管理証跡の共通ロックを取り、Taskとバックログを検査・公開してから、ブランチとsparse ワークツリーを作成する。作成失敗時は今回作成した証跡だけを訂正し、既存資源を削除しない。

## DEV中

- DEV Agentは割り当てられたワークツリー以外で製品コードを変更しない。
- コミットは論理的な変更単位で作る。
- 無関係な整形、生成物、ローカル設定を混ぜない。
- main Agent以外は`main`を更新しない。
- force push、履歴改変、破壊的resetを通常手順にしない。
- DEV完了時に`candidate_commit`と`candidate_tree`を固定し、同じtreeをREVIEWとQAへ渡す。案を変更した場合は旧結果を自動で再利用しない。

## マージ

main Agentは`REVIEW_RESULT.md`と`QA_RESULT.md`が同一案を対象とし、`make check`、QA計画の再確認、cleanなワークツリーを確認する。修正後のcarry-forwardは[QAガイドライン](qa.md)の閉じた`CF-1`から`CF-7`を全て証明し、Mainが`QA_RESULT.md`へ記録した場合だけ許可する。影響QAケース集合が空でない、禁止条件が真、または影響が不明ならrerunへfail-closedする。

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

終了後は`make sync`がmerged済みでcleanなワークツリー/ブランチだけを削除する。dirty、競合、赤CIでは削除前に停止する。
