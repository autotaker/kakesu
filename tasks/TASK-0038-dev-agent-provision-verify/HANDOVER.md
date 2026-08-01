---
task_id: "TASK-0038"
status: complete
completed_at: "2026-08-01T03:54:49Z"
candidate_commit: "1ffd4825f6672bb1522ed9df5387dc51f2cc86d8"
---

# TASK-0038 HANDOVER

## 成果

- Ubuntu provision manifestをOS適用前にread-onlyで検証する`verify-provision` CLIを追加した。
- 一度だけopenしたfile descriptorのtype、mode、size、byte countを検査し、pathを再openしない。
- `provision.Build`との完全byte一致だけを受理条件とし、独立parserや別schemaを追加しなかった。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `make check`（candidate launcher） | PASS |
| `cd tools/dev-agent-harness && ./configure && make check && make distcheck` | PASS |
| `git diff --check` | PASS |

## 主要な変更

- `internal/provision/verify.go`に128 KiB上限のsingle-FD readerとcanonical byte照合を追加した。
- setup commandへ固定引数順、固定success summary、非漏洩error classのadapterを追加した。
- valid、1 byte追加/変更/削除、config/root mismatch、file policy、read error、read中mode変更、path非再open、副作用不在を通常testで確認した。
- READMEへread-only consumerとOS適用処理との境界を追記した。

## 検証結果

- `make check`: PASS（candidate `1ffd4825f6672bb1522ed9df5387dc51f2cc86d8`）
- harness check/distcheck: PASS
- candidate差分: 5 files、追加408行・削除1行

## 判断・既知の制約

- root権限、実OS、systemd、executor、network、IPCを含まないためlive E2Eは不要。
- candidate検査の初回はREADMEの用語出現閾値だけでFAILし、glossaryを増やさずREADMEの重複語を削ってから同じ全検査をPASSした。
- 実際のmanifest適用とrollbackは後続Taskの別security boundaryである。
