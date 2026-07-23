---
task_id: "TASK-0009"
title: "Linux Workspace namespace・cgroup・process identity/lifecycle基盤を実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0009 Linux Workspace namespace・cgroup・process identity/lifecycle基盤を実装する

## 目的

Linux P0の後続強制機能が利用する、Task Workspaceごとのnamespace、cgroup、process identity/lifecycle基盤を提供し、管理対象プロセスを確実に回収する。

## 背景

Workspaceはpolicyの適用主体であり、Task/Agentの自己申告で隔離を代替できない。`docs/07-governance.md`はLinux P0を初期正本とし、P0を全て強制できないOSプロファイルを提供しないとする。

## スコープ

### 対象

- Linux capability probeとprofile admission、workspace root生成、user/PID/IPC/mount/network namespaces、cgroup、rootless/no_new_privs/capability drop、process identityとPID cleanup。
- Workspace IDをnamespace/cgroup/process identityへ一意に束縛し、生成・終了・cleanup失敗を監査する。
- `empty` WorkspaceのみのP0実行adapterとworkspace-created境界イベント。

### 対象外

- Landlock/LSM allowlistによるFS強制、mount boundaryを越えるFS escape試験（TASK-0022）。
- DNS/proxy/firewall規則、外向き通信経路の強制、HTTPS parsing/canonicalization（TASK-0017/TASK-0023）。
- `fork`/`shared_readonly`、snapshot/quota/seccomp/device制御などP1以降、macOS/Windows native profile、一般的なコンテナorchestration。
- root helperによる昇格を前提とした代替経路。

## 受け入れ条件

- [ ] Linux P0 profileは必要なkernel機能を起動前にprobeし、一つでも不足すれば実行を開始せず理由を返す。
- [ ] profileはnamespace/cgroup/process identityに必要なkernel機能を起動前にprobeし、一つでも不足すればworkloadを開始せず理由を返す。
- [ ] sandbox processはrootless、`no_new_privs`、capability dropで起動され、子孫を含めてTask終了/失敗/cancel時に回収される。
- [ ] workspace IDは生成したnamespace/cgroup/process identityとTask IDに一意に対応し、生成・終了・cleanup失敗の監査記録が残る。
- [ ] namespace/cgroup/create/execの一つでも失敗すればcommandを実行せず、reverse-order cleanupは冪等である。

## 検討すべき設計観点

- 実行基盤と、TASK-0022のFS強制、TASK-0017のnetwork route、TASK-0023のHTTPS境界との責務分離。
- namespace/cgroupの失敗を部分的な実行として続行しないfail-closed設計。
- process lifecycleと論理Workspace lifecycleを分離し、cleanupの冪等性を担保する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- `docs/00-kakesu.md` §§2.1–2.2, 4: Workspaceと外部境界。
- `docs/01-domain-model.md` §3: logical Workspaceと物理実行resourceの分離。
- `docs/07-governance.md` §2: Linux P0のnamespace/cgroup/process lifecycle基盤。
- `schemas/draft-v0/execution-plane/workspace-created.schema.json`: `empty`とgrant非継承。

### 判断

- Decision: Linux専用の実行基盤を先に提供し、FS強制とnetwork規則は専用Taskへ分離する。必要なnamespace/cgroupを確立できない環境ではprofileをadvertiseしない。

### 適用しなかった重要な判断

- Docker依存だけの抽象化はworkspace identity/lifecycle証拠を検証不能にするため不採用。FSまたはnetworkのbest-effort代替も本Taskの完了として扱わない。
