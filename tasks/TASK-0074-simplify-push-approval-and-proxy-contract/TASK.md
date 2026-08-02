---
task_id: "TASK-0074"
title: "Push承認と薄いproxyへ安全契約を簡素化する"
status: done
created_at: "2026-08-02"
---

# TASK-0074 Push承認と薄いproxyへ安全契約を簡素化する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

Development Agent Harnessの認証転送とpush承認を、実VPSで必要な最小安全境界へ戻す。pushはref/SHA/manifestではなく「指定repositoryへの次のgit push一回」をagent instance、workspace、repository、短いTTLへ束縛する。Git read、通常のGitHub REST、OpenAI APIはprovider protocolの意味を再実装せず、Opaque capabilityと接続先の境界を確認した後に原則そのままstream転送する。既存の公開証跡は履歴として保持し、旧契約を今後の認可根拠にしない。

### 対象と対象外

#### 対象

- `docs/development/development-agent-harness.md`のpush承認契約をrepository単位のone-shot grantへ置き換える。grantはagent instance/UID、workspace、exact repository、短TTL、一回消費、revokeへ限定し、上流`git-receive-pack`試行開始前に原子的に消費する。
- 侵害Agentが承認済み一回を同じrepositoryへの別内容pushへ使う残余リスクを明示的に受容する。一方、別repository、別agent/workspace、複数回、期限後、GitHub RESTその他操作への転用は拒否する。
- 承認UIはbranch/commit/ref/SHAを参考表示にだけ使い、認可根拠にしない。主表示は「このrepositoryへの次のpush一回」を正確に表す。
- proxyを薄い認証差し替え境界として再定義する。維持する境界はUnix peer identity、host allowlist、Opaque handleのsubject/provider/repository/TTL/use/revoke、実credential置換、TLS CONNECT/ローカルCA、timeout/concurrency/header上限、secret-free auditとする。
- 通常のGit read、GitHub REST、OpenAI APIはmethod/path/query/bodyと上流status/headers/bodyを、HTTP framing・hop-by-hop header・secret境界に必要な最小処理を除き、原則未解釈のstreamとして転送する。pushだけはrepositoryと`git-receive-pack`の最小分類を許すが、本文を解析しない。
- 次の削除対象を明記する: `approvalmanifest`、old/new SHA・ref一覧・force/delete・remote old SHA・Git wire/pkt-line本文照合、strict OpenAI JSON field/model/store/stream検査、GitHub `/repos/{owner}/{repo}` endpoint parser、upload-pack本文/response Content-Typeの意味検査、JSON response検証、2xx限定、response全量buffer/1MiB上限、Policy→Transaction→Exchange→Forwarder間の重複評価・不要抽象層。
- `backlog.yaml`で旧TASK-0070契約をsupersededとして扱い、TASK-0071〜0073は実装履歴を保持しながらmanifest/digest束縛を後続のrepository単位request/decision/grantへ移行する対象とする。次の製品Taskを、薄いproxyとrepository one-shot pushを実VPSで縦断確認する一つのvertical sliceとして最優先にする。
- ルール追加ではなく削除を既定とし、provider意味検査の再導入は実E2Eで反復して観測された具体的不具合に対する最小対策だけを別Taskで判断する。

#### 対象外

- 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動はこのTaskでは変更しない。
- 既存TASK-0070〜0073のTASK/PLAN/HANDOVER/REVIEW/QA/Wiki receiptを遡及編集又は削除しない。
- proxy、approval state、grant、WebAuthn、Tailscale、GitHub App又はVPS設定をこのTaskで実装しない。削除・移行・live E2Eは次の製品Taskで同じvertical sliceとして行う。
- 新しいformat version、receipt、機械check、証跡field、互換wrapper又は将来用の抽象層を追加しない。

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [x] AC-1: 設計書はpush authorizationをexact repositoryへの次の`git-receive-pack`一回として定義し、agent/workspace/repository/短TTL/一回消費/revokeを必須にする一方、ref/SHA/manifest/body一致を要求しない。
- [x] AC-2: 設計書は同一repository内のpush内容差し替えリスクを受容し、別repository・別主体/workspace・複数回・期限後・REST転用を拒否し、GitHub Appのrepository限定write権限を上流安全境界にする。
- [x] AC-3: proxy契約は通常のGit read/GitHub REST/OpenAIを未解釈stream転送とし、維持する安全境界と削除対象を明示する。pushはrepository/receive-packだけ分類し、本文を解析しない。
- [x] AC-4: 承認UIの認可文言と参考情報を分離し、旧TASK-0070をsuperseded、TASK-0071〜0073をrepository単位へ移行対象として、次の一つの製品Taskを薄いproxy＋承認後pushの実VPS vertical E2Eへ優先する。
- [x] AC-5: 差分は設計書、backlog、TASK-0074計画証跡だけで、製品コード/test/config/dependency/Schema/生成物を変更せず、新規check・field・wrapperを追加しない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | ユーザによる要件変更 | 2026-08-02 delegation | repository単位one-shot push、薄いproxy、削除対象、実VPS E2E優先 |
| REF-2 | Development Agent Harness設計 | main `b7cd3cd8723347c7221fcbe036c5b16ae3b61c11` | 置換する旧ref/SHA/manifest契約と維持するOS・credential境界 |
| REF-3 | TASK-0070〜0073公開証跡 | main `b7cd3cd8723347c7221fcbe036c5b16ae3b61c11` | 書換えず履歴として保持する旧実装根拠 |
| REF-4 | 現行proxy実装 | main `b7cd3cd8723347c7221fcbe036c5b16ae3b61c11` | 後続製品Taskで削除する意味検査・buffer・重複層の棚卸し |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| なし | `ready` | N/A | N/A |

### 許可パス

- `backlog.yaml`
- `docs/development/development-agent-harness.md`
- `tasks/TASK-0074-simplify-push-approval-and-proxy-contract/TASK.md`
- `tasks/TASK-0074-simplify-push-approval-and-proxy-contract/PLAN.md`
- `tasks/TASK-0074-simplify-push-approval-and-proxy-contract/QA_PLAN.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | safety contractの対象検査、`git diff --check`、許可path確認を使う。製品PASS証跡は作らない |
| 権限 | `ready` | Planner/QAが計画、Mainが意図・scope・受け入れ経路を承認し、MainだけがGit統合する |
| 依存状態と参照 | `ready` | main `b7cd3cd`の実装・設計・公開証跡を基準とし、外部依存なし |
| 生成物の有無と更新方法 | `ready` | 生成物なし。設計書とbacklogだけを直接更新する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0074-simplify-push-approval-and-proxy-contract` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

現行設計はpush内容をref/SHA/manifestへ完全束縛し、通常APIにもprovider固有の意味検査を重ねた結果、認証転送周辺だけが大きくなった一方で、実VPS上のpull/API/承認後push縦断経路はまだ成立していない。ユーザは同一repository内の一回分のpush内容差し替えを受容し、安全性をrepository限定GitHub App、一回消費、短TTL、主体/workspace束縛へ集中させると決定した。旧契約の実装を積み増さず、不要コードを将来用に残さない順序へ直ちに切り替える。

## 検討すべき設計観点

- capability認可とprovider protocol意味検査を混同せず、実credential非露出とrepository/主体/use境界だけを小さく監査可能にする。
- grantはpush成功時ではなく上流試行前に消費し、失敗時にも再利用しない。結果不明時の再試行は新承認とする。
- HTTP streamingでもhop-by-hop header、credential、redirect、timeout、size/concurrencyによる資源枯渇などproxy固有の安全性は維持する。
- branch/commit参考表示が認可根拠に見えないUI文言を固定する。
- 既存Task証跡を履歴として残しつつ、backlogと新設計を将来の正本にする。
- 次製品Taskは削除と縦断成立を一緒に受け入れ、互換層を増やさない。必要なら追加分約1,000行を目安にし、削除行数で形式的に分割しない。

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [x] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-push-approval-manifest.md`は旧契約の履歴であり、このTaskのWiki ingest時にsuperseded関係を新規知識として記録する候補とする。

### 判断

- push承認の安全単位はref更新内容ではなくrepositoryへの次のpush一回とする。
- proxyは認証情報置換とcapability境界を所有し、provider APIの意味を再実装しない。

### 適用しなかった重要な判断

- なし
