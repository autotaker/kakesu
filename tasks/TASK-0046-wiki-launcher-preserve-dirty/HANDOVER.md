---
task_id: "TASK-0046"
status: complete
completed_at: "2026-08-01T10:01:58Z"
candidate_commit: "73a8a6f0339240b38b8597a2a15906813e949464"
---

# TASK-0046 HANDOVER

## 成果

- Wiki launcherのedit-only実行が開始前dirty状態と子Agent差分を分離し、成功時は子が変更した許可pathだけを残すようにした。
- 失敗時はHEADを戻してから開始前のtracked bytes/削除/mode、index、untracked fileを復元し、子の変更だけを除去する。
- 子stderrは固定prefix、exit code、bounded/redacted要約だけを既存error欄へ出し、raw stderrと新しい証跡fieldを公開しない。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `node --test --test-name-pattern='wiki edit-only dirty preservation' scripts/task/development-process.test.mjs` | PASS。temporary Git fixtureの9 testを開発中の最終codeで実施 |
| candidate launcherの`make check` | PASS。固定candidate bytesに対して一回完走 |
| `node --check`対象3ファイル / `git diff --check` | PASS |

## 主要な変更

- 開始前dirty pathだけのsnapshotへworking bytes/mode/存在、untracked集合、raw index復元値、意味上のstaged diffを保持する。
- 子実行後の新規dirty pathと開始前dirty pathの状態差を導出し、同一path衝突、stage、HEAD変更をfail-closedにする。
- edit-only専用restoreを追加し、commit-modeと既存clean-start `rollbackWorkRepository`は変更しない。
- launcher nonzeroと、子成功後の実`validateWork()`失敗をfake `codex`で再現し、Task証跡復元、子untracked除去、exit code、redaction、追加field不在を検出する。

## 検証結果

- `make check`: 最終candidateでPASS。最初の試行は実行中にDEV Agentがtest assertionを追加したためbytes不一致でfail-closedし、品質証跡には採用していない。pre-P1 candidateのPASSも最終candidateの証拠へ持ち越していない。
- focused fixtureはstaged+unstaged同居、tracked削除、untracked/mode、許可Wiki成功、同一path衝突、stage後restore、子commit、scope drift、子nonzero、実validation failure、秘密候補redactionをPASSした。
- 差分は許可3 path、418追加・10削除で1,000行以下。dependency、Schema、Wiki本文、Task lifecycle、CLI引数、証跡形式の追加はない。

## 判断・既知の制約

- indexのstage検出はraw index bytesではなく`git diff --cached --binary --no-ext-diff`を比較し、`git status`のstat-cache refreshを誤検出しない。raw bytesは正確なrestoreだけに使う。
- ignored fileは既存の`--exclude-standard`契約どおりsnapshot/scope対象外。dirty symlinkやspecial fileは開始前にfail-closedとする。
- 最終candidateのlauncherをmainのdirty完了証跡に対して`--commit false`で実行し、実Codex Wiki ingestがexit 0、commitなし、errorなしで完了した。開始前のHANDOVER/QA_PLAN/REVIEW/QA bytesとcandidate bindingは保持され、Wiki許可pathだけが残ることをMainが確認した。
