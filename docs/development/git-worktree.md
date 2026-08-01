# Git ワークツリー運用

製品Taskは `task/TASK-NNNN-short-slug` と `worktrees/TASK-NNNN-short-slug` を使う。

`task-start`はブランチ/ワークツリーとmain側雛形を作るだけで、main コミットを作らない。Plannerが計画を整えたら`planning-gate`がmainへ一回のplanning コミットを作り、Task ブランチ/ワークツリーをfast-forwardする。

DEVはTask ワークツリーで編集し、`candidate-commit`を一度だけ実行する。案 コミットはplanning コミットの直後に置く製品差分だけの一コミットで、Task ワークツリーのpre-commit hookはロック所有の起動処理による`candidate-commit`以外を拒否する。

完了時はMain ワークツリーで`completion-gate`を実行する。staged変更を拒否し、証跡を保存して非fast-forward mergeを検査し、許可された証跡と製品差分を一つのmerge コミットへまとめる。失敗時はabort後に証跡バイトを復元する。
