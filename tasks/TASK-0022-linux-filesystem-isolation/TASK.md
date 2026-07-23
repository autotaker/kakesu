---
task_id: "TASK-0022"
title: "Linux P0 Landlock/LSM多層ファイルシステム隔離を実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0022 Linux P0 Landlock/LSM多層ファイルシステム隔離を実装する

## 目的

TASK-0009のmount namespace基盤にLandlock/LSM allowlistを重ね、Task Workspaceと明示read-only runtime allowlist以外への読み書きをfail-closedで拒否する。

## 背景

mount boundaryだけではbind mount・procfs・path traversal等のescapeを十分に強制できない。`docs/07` §2のLinux P0はmount namespaceとLandlock/LSMの多層強制を要求する。依存: TASK-0006、TASK-0009。

## スコープ

### 対象

- Landlock/LSM allowlist、mount boundaryとの多層FS強制、Linux profile probe/admission、監査可能なfail-closed。
- host credential、管理socket、非allowlist host path、mount/procfs/path traversal escapeのnegative tests。

### 対象外

- namespace/cgroup/process lifecycle生成（TASK-0009）、DNS/proxy/firewall（TASK-0017）、HTTPS interception（TASK-0023）。
- `shared_readonly`、snapshot/quota/seccomp/device制御などP1以降。

## 受け入れ条件

- [ ] profile probeは必要なLandlock/LSM enforcement能力とmount boundaryを検証し、不足hostではworkloadを開始せずfail-closed理由を返す。
- [ ] sandboxはTask Workspaceと明示read-only runtime allowlist以外を読み書きできず、host credential・Core/Governance socketを参照できない。
- [ ] mount、symlink、`..`、procfs、bind/magic-link等のescape負例はすべて拒否される。
- [ ] enforcement/create失敗はcommand実行前にcleanupされ、profile admission・拒否・失敗が監査可能である。

## 検討すべき設計観点

- mount namespaceは可視性、Landlock/LSMはpath権限の別強制層として扱い、一方の成功でP0をadvertiseしない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- `docs/07-governance.md` §2、`docs/11-data-model.md` §2、`docs/12-implementation-and-tests.md` §5、`docs/13-technology-stack.md` §§2,4。依存: TASK-0006、TASK-0009。

### 判断

- mount boundaryとLandlock/LSM allowlistを併用し、unsupported hostはbest-effortにせず拒否する。

### 適用しなかった重要な判断

- mount namespaceだけ、パス文字列filterだけ、unsupported hostでのwarning起動は不採用。
