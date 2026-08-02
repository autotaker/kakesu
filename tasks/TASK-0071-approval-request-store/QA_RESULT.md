---
task_id: "TASK-0071"
status: complete
qa_agent: "qa-agent-terra-medium"
decision: pass
tested_at: "2026-08-02T07:29:00Z"
---

# TASK-0071 QA RESULT

## 結果

| ケース ID | コマンド/テスト | 結果 |
|---|---|---|
| QA-001 | `focused-rerun`: `cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/approvalstate` | `PASS` — 一回のrace suite（1.664s）が既存`0700` root、same-root lock競合、relative root、symlink/mode拒否、Closeのidempotenceを検出した。`TestRootRenameDoesNotRedirectStore`はroot rename/symlink置換後も最初に固定したdirectoryだけが更新されることを検出する。Darwin実行であり、Linux側はbuild tagとHANDOVERのcross-compile PASS記録を監査しただけで実行PASSとはしていない。 |
| QA-002 | 同focused-rerun | `PASS` — canonical parse/re-encoding、policy/epoch、future/expired/TTL、capacity、同一bytesを含むduplicate ID、actor入力制約を検出した。API/diff監査でcallerがdigest/state/time/pathをCreateへ注入できないことも確認した。 |
| QA-003 | 同focused-rerun | `PASS` — pending→approved/denied/cancelled、approved→stale、digest mismatch、terminal再遷移、expiry優先、Get/ExpireDue、clock rollbackを検出した。Passkey検証・grant/push接続を追加していないことをcandidate diffで確認した。 |
| QA-004 | 同focused-rerunおよびcandidate source/test監査 | `PASS` — strict JSONのunknown/duplicate/trailing/whitespace/record mismatch、**partial**、**oversize**、temp、symlink、mode、restartを拒否する。`TestPersistenceFailuresAndPoison`は既存snapshotを基準にwrite/file-sync/close（rename前）failureごとのgeneration、records、disk bytes不変を直接assertし、rename/directory-sync不確実時はpoison後の操作拒否とre-openを検出する。 |
| QA-005 | 同focused-rerun | `PASS` — `-race`付きでbounded Create/Get/Approve/ExpireDue/Snapshot/Close競合、nil/closed/poisoned、record/snapshot encoding copy、固定errorの非漏えいを検出した。 |
| QA-006 | candidate diff、README、HANDOVERのcandidate-bound evidence監査 | `PASS` — HEADとHANDOVERの`candidate_commit`は`dc38a17a01223c49af025375366b0d781b1302fa`で一致。差分は許可5パスのみ（+1414/-0）、新規依存、HTTP、credential、Git、external DB、config/deploy/generated/live stateはない。READMEはstate、single-writer、durability/poison、認可境界、後続Taskを記載する。HANDOVERのroot/harness check、distcheck、diff checkは同candidateのPASS証跡として監査し、QAでは重複実行していない。 |
| LIVE-001 | `live-e2e` | `blocked/not run` — 実UID・`/var/lib` permission、VPS/systemd provision/restart/rollback、power-loss durability、verified-decision OS/process boundary、Passkey、grant、pushは対象外で、承認済み実環境とcleanup手順もない。hermetic PASSで代替していない。 |

## 発見事項

- 初回candidateの`test/fixture evidence gap`は、partial/oversize corruptionとwrite/file-sync/closeのpre-rename durability assertionを追加した本candidateで解消した。
- テスト起動時の`pyenv` rehashおよび`.zlogin`の`nice`権限メッセージはexit 0のGo suiteに影響せず、環境通知に分類する。

## 結論

`PASS` — QA-001〜QA-006は、新candidateに対する指定focused-rerunとcandidate-bound evidence reviewを満たした。環境依存のlive-e2eはblocked/not runのままマージ後確認対象として残る。reviewer結果はこの判定の開始条件にしていない。
