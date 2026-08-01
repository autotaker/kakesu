---
task_id: "TASK-0053"
status: complete
completed_at: "2026-08-01T14:58:08Z"
candidate_commit: "0c395f8f167f2081966604a0df7bb37ca6c6f0b3"
---

# TASK-0053 HANDOVER

## 成果

- 既存broker credential snapshotを、同一reader・同一failure domainの固定6 filesへ拡張した。
- 既存credential検証後だけCA certificate/keyを`proxyca.New`へ委譲し、Bundleにはparsed Authorityだけを保持した。
- Linux hardlink拒否、6-file atomicity、CA failure folding、動的clock、非漏洩accessor、並行利用の回帰検出を追加した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| rebase後のexact candidate root `make check` | sandbox内の未cached`hatchling` DNS失敗を`environment_issue`と分類し、同一candidateをnetwork許可付きで再実行して`PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 ./internal/brokercredentials`（最終hook修正後） | `PASS`（2.54s） |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 ./internal/brokercredentials ./internal/providercredentials`（Main先行確認） | `PASS` |
| DEV focused race / harness `make check` / `make distcheck` | `PASS`（candidate固定前。exact candidate raceは独立QAが一回実行） |
| README textlint / `git diff --check` | `PASS` |

## 主要な変更

- `brokercredentials.Load`のbasenameを既存4件に`proxy-ca-cert.pem`と`proxy-ca-key.pem`を加えた固定6件へ拡張し、全検証成功後だけBundleを返す。
- `proxyca.ClockFunc`が呼出し時の`nowUTC`を参照し、CA入力bytesをBundleへ保持せず、nil/zero/corruptでnilとなる`ProxyCAAuthority()`だけを追加した。
- Linux regular-file policyに`nlink == 1`を加え、CA fileにもowner/mode/size/no-follow/FIFO/metadata-raceの既存reader境界を適用した。
- brokercredentialsのsecurity/atomicity testsを拡張し、6-file移行で影響するprovidercredentials既存fixtureだけへCA二fileを追加した。
- READMEへstartup snapshotとtrusted composition境界を追記した。

## 検証結果

- candidate `0c395f8f167f2081966604a0df7bb37ca6c6f0b3`を固定した。base `a1d73f89ead7a990f72011b860e402f2ea56dce6`からの製品差分は6 files、追加299行・削除9行であり、旧candidate `1aa67b35fa45a26b73445e28034f261237e849f2`との`tools/`差分は空である。
- exact candidateのroot `make check`はsandbox内の未cached build依存を取得できず一度停止した。製品差分を変更せず、同じcandidate・commandをnetwork許可付きで再実行してPASSした。
- 6-file missing/empty、CA file policy、Linux hardlink/読込中metadata変化、CA parse/key/curve/CA属性/期限、ErrLoad/no partial、Authority copy/host/clock、既存JWTとの並行利用をfailure-detecting testへ結び付けた。
- providercredentialsの既存fixtureは6-file必須化に追随し、影響二packageの通常testがPASSした。

## 判断・既知の制約

- CA専用reader/parserは作らず、file policyはbrokercredentials、certificate/keyの意味はproxycaをそれぞれ正本とした。
- `ProxyCAAuthority()`は検証済みAuthorityへのpointerを返し、Authority自体をcloneしない。Authorityのfieldは非公開で、copyされるのは`PublicCertificatePEM()`が返す公開CA bytesだけである。
- providercredentials test fixtureの追随は実装中に判明した直接回帰であり、製品挙動や期待値を広げずCA二file生成だけに限定した。
- planning gateが必須DEV profile frontmatterの欠落を見逃し、completionで初めて検出したため、PLAN訂正1 commitとcandidate rebaseが追加された。履歴改変や未検証結果のcarry-forwardは行っていない。
- CA生成/書込み/rotate/reload/watch、Agent trust、実OS user/permission、listener composition、実GitHub/OpenAI/network、VPS live E2Eは未実装・未確認であり、hermetic PASSで代替しない。
