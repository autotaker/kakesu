---
task_id: "TASK-0002"
status: complete
completed_at: "2026-07-14T12:37:29+10:00"
---

# TASK-0002 HANDOVER

## 完了証跡

- 独立Reviewerは製品commit `35fcaf209a65d17acd5d215298ff6175b0671572`をreviewし、最終判定は`pass`。
- main Agentは同commitを正確な2-parent `--no-ff` merge `a0661057b56461c1a0e0f01a326d487b094e5ea1`として製品`main`へ統合した。
- main-owned work adapterはcommit `27d04061f5f7858e59e61998d28c673d12e31b6c`、digest `5566794aaf22a890ef432e4e45b63a3a07e1bcb3eaf0cf620a47883c705c3445`で同期済み。
- `QA_RESULT.md`はQA-001〜QA-013の最終`pass`。process tests 29/29、`UV_OFFLINE=1 make check`、`make work-check`、`make task-check TASK=TASK-0002`、adapter CHECK（`make work-config-sync ... CHECK=1`、`changed=false`）はすべてpassした。

## 最終routing contract

- 製品repositoryのproject-scoped TOMLを正本とし、main=`gpt-5.6-sol/high`、PLAN/QA/REVIEW=`gpt-5.6-terra/medium`、DEV=`gpt-5.6-luna/xhigh`または`gpt-5.6-sol/high`、Explorer=`gpt-5.6-luna/medium/read-only`へ固定する。
- Luna DEVは明確・局所的・機械検証可能で高risk signalが一つもない場合だけ許可する。横断性、契約、Schema、security、concurrency、migration、不明点のいずれかがあればSolへ倒し、promotionは履歴とmain承認を要し、降格は許可しない。
- fixed roleと異なるmodel/profile/effort overrideは起動前にfail closedとする。互換overrideはroleを経由しない明示legacy経路だけに限定する。
- Explorerは明示launcherから一回につき一つのbounded read-only質問だけを受け、編集、Git操作、scope拡大、再委譲を行わない。root→role→Explorerまでの最大depth 2、最大threads 2とし、簡潔な根拠要約とfile referenceだけを返す。
- work repositoryのadapterは正本から決定的に生成し、digestと完全一致検査でdriftを拒否する。global設定、未文書include、model alias、Agentの自己申告には依存しない。

## 親所有sync / rollback

- 専用`work-config-sync` launcher親が共通lockを全工程で保持し、`.codex/config.toml`だけを生成・scope検査・stageし、共有hook、commit、post-checkまで所有する。子Agentとgeneric governance経路へadapter同期やcommit authorityを渡さない。
- child failure、scope/stage/commit drift、hook failure、pre/post-check failureはすべてfail closedとし、開始前HEAD、index、worktree、untracked状態へrollbackして`commit:null`を記録する。no-opとCHECKは書込みもcommitもしない。
- launcher childのstdinは必ずcloseし、子のexit 0だけでcommitせず、親がscope、HEAD、stage、hook、検証結果を確認する。

## 再利用可能な知識

- role routingは正規TOML、launcher入力検査、決定的な正負テスト、簡潔なlauncher traceを組み合わせると監査できる。Explorerについては自己申告ではなくlauncher traceを信頼する。
- 複数repository rootで設定を共有するときは、一方を正本、他方をmain-owned生成adapterとし、digest付きdrift検査をfail closedにすると所有境界を保てる。
- 書き込み子とlock所有親を分離し、親だけがscope検査、hook付きcommit、再検証、rollbackを行うことでAgent責務とcommit authorityが一致する。
- read-only sandboxだけではExplorerの境界は十分でない。固定prompt、明示launcher、closed stdin、no-child設定、負例テストを重ねる。

## 初回運用の教訓

- Close child stdin: 子プロセスのstdinをcloseしないと、入力待ちでlauncherが停止し得る。
- Trust launcher trace rather than self-report for Explorer: model、sandbox、質問数、commit不在はAgentの自己報告ではなくtraceで判定する。
- An outer evidence-writer lock prevents nested lock-owning checks. そのcheckは外側lock解放後に適切な親実行contextで行う。
- QA product build outputs require a suitable execution context with write access to their output paths. 今回の制約はmain再実行で解消し、製品acceptance failureではない。
- Generic work-agent prompts currently reread too much context and consumed excessive tokens. Prompt/context minimization is a follow-up improvement rather than a product acceptance failure.

## Wiki取り込み候補

- `wiki/semantic/scripts/task-delivery.md`
- `wiki/semantic/schemas/work-repository-boundary.md`
- `wiki/semantic/schemas/codex-agent-model-routing.md`（新規候補）
- `wiki/decisions/DECISION-0003-codex-agent-model-routing.md`（新規候補）
