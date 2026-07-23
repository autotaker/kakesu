---
kind: schema
title: Codex Agent Model Routing
---

# Codex Agent Model Routing

## 問い

固定した役割割当、限定的な調査委譲、二つのrepository root間の設定同期を、誰が検証・書込み・commitするかを曖昧にせずにどう運用するか。

## 正本と固定割当

製品repositoryのproject-scoped TOMLを正本とする。mainは`gpt-5.6-sol/high`、PLAN・QA・REVIEWは`gpt-5.6-terra/medium`、Explorerは`gpt-5.6-luna/medium`でread-onlyとする。DEVは、明確で局所的かつ機械検証可能で、risk signalがない場合だけLunaを選べる。それ以外、または横断性、契約、Schema、security、concurrency、migration、不明点がある場合はSolを選ぶ。LunaからSolへのpromotionは履歴とmain Agentの承認を要し、降格しない。

固定roleと異なるmodel、profile、effortのoverrideは起動前に拒否する。互換overrideはfixed roleを経由しない、明示されたlegacy経路に限る。

起動時の`task_name`は追跡用の識別子であり、roleを選択しない。role選択は`agent_type`で行う。内部`agents.spawn_agent`を標準経路とし、`agent_type`の欠落、内部spawnが利用不能、または起動後に観測したmodel/effortが契約と不一致の場合だけ、停止してrequested値・observed値・ランタイム条件を証跡化したうえで限定的なfallbackを判断する。不一致の子成果を採用して処理を継続してはならない。

呼出元と異なるroleを起動するときは`fork_turns="none"`を明示する。既定値`all`は呼出元の設定を継承し、名前付きroleによる契約適用を拒否し得るためである。PLAN→DEV→REVIEW→QAの順次gateとDEVからReviewer・QAを分離する境界は、この起動経路の選択とは別の不変条件として維持する。

## Explorer境界

Explorerは明示launcherから、一回につき一つのboundedなread-only質問だけを受ける。rootから直接起動し、Codex組込み既定のdelegation上限（depth 1、threads 6）の範囲で運用する。編集、Git操作、scope拡大、再委譲を行わず、簡潔な根拠要約とfile referenceだけを返す。read-only sandboxだけで完結せず、固定prompt、closed stdin、no-child設定、負例試験、launcher traceでこの境界を検証する。設定ファイルの`sandbox_mode`は意図する契約を表すが、実効sandboxがランタイムで観測されるまでは保証済みと扱わない。

`[agents.<role>]` child tableは暗黙に親`agents`を作る。したがって明示`[agents]` header、そこに置くglobal `max_depth` / `max_threads` override、旧feature/keyを除去しても、7 roleの`config_file` mappingは削除しない。通常roleはrole-localの`max_depth` keyを持たず、Explorerだけはrole-local `max_depth = 0`と`max_threads = 1`でno-childを明示する。同じdepth keyを一律に扱わず、通常roleの重複key再導入、Explorerのno-child緩和、role欠落・追加、mapping変更を別の構造検査・エラー理由で拒否する。

## 生成adapterとcommit境界

work repositoryのadapterは正本から決定的に生成し、digest付き完全一致検査でdriftを拒否する。global設定、未文書include、model alias、Agentの自己申告を入力にしない。canonical parserをdigest計算、adapter render、adapter checkの共通入口とし、禁止設定の再導入、role registry drift、mapping driftはadapterの生成・同期より前にfail closedする。これによりdriftした正本からadapterを更新しない。

専用sync launcherの親が共通lockを全工程で保持し、生成対象、scope検査、stage、shared hook、commit、post-checkを所有する。子Agentとgeneric governance経路はadapter同期やcommit authorityを持たない。子Agentはstage、commit、merge、`.git`書込みを行わない。child failure、scope・stage・commit drift、hook failure、pre/post-check failureでは、開始前のHEAD、index、worktree、untracked状態へrollbackし、`commit:null`を記録する。no-opとCHECKは書込みもcommitもしない。

## 検証と環境帰属

routingは正規TOML、launcher入力検査、決定的な正負試験、簡潔なlauncher traceで監査する。Explorerの境界は自己申告ではなくtraceで判定する。

設定値の有無だけを照合する試験ではchild registryの意図しない削除を見逃すため、明示parent/global overrideと旧feature/keyの再導入、各通常roleへの値を問わない`max_depth`注入、Explorerのdepth/thread緩和、role欠落・追加、mapping変更を個別fixtureで検証する。組込み既定のdepth 1 / threads 6を許可し、nested depth 2 / threads 7を拒否する。parser、digest、render、checkの全入口が同期前に同じく拒否することを確認する。

独立QA環境でlock所有者の確認や製品build出力への書込みが許可されない場合、その未完走を環境所見として記録する。lock解放後または必要なwrite権限を持つmain実行環境で同じ必須commandが成功したとき、それは製品受け入れfailureや自動的なDEV帰責にしない。

## 関連

- [QA FAIL attribution](../case-patterns/qa-fail-attribution.md)
- [MultiAgentV2 role startup](../../decisions/DECISION-0004-multiagentv2-role-startup.md)
- [TASK-0002 HANDOVER](../../../tasks/TASK-0002-codex-agent-model-routing/HANDOVER.md)
- [TASK-0003 HANDOVER](../../../tasks/TASK-0003-multi-agent-v2-nested-explorer/HANDOVER.md)
- [TASK-0004 HANDOVER](../../../tasks/TASK-0004-remove-redundant-agent-depth-limits/HANDOVER.md)
- [TASK-0032 HANDOVER](../../../tasks/TASK-0032-reset-default-agent-config/HANDOVER.md)
