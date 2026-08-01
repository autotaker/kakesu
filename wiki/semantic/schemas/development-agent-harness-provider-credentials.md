---
kind: schema
title: Development Agent Harness プロバイダー認証情報の解決
---

# Development Agent ハーネス プロバイダー認証情報の解決

## 問い

外向き通信 トランザクションがOpaque ケイパビリティを消費した後、ブローカー内の実認証情報をOpenAI又はGitHub向けの上流認証情報へどう解決するか。

## プロバイダー分岐

resolverは検証済みの`brokercredentials.Bundle`だけを秘密情報源にする。

- OpenAIはリポジトリが空の場合だけ、バンドル内のAPIキーをネットワークなしで返す。
- GitHubは正規`owner/name`を要求し、リポジトリ名一件だけに限定したinstallation アクセス トークンを呼出しごとに取得する。
- 未知プロバイダー、プロバイダーとリポジトリの組合せ違反、非正規リポジトリは転送方式へ到達する前に拒否する。

resolverは値を同期的callerへ返すため、ケイパビリティ消費済みという順序とtrusted Forwarderだけへの受渡しは`egresstransaction`との合成契約である。resolver自身は値をログ、cache、永続化しない。

## GitHubの単発交換

交換先は`POST https://api.github.com/app/installations/{installation_id}/access_tokens`へ固定する。App JWTは`Authorization: Bearer`だけに使い、本文は`{"repositories":["name"]}`の一件に限定する。API バージョン、受理、Content-Typeも固定する。

permissionは交換時に部分列挙しない。最小権限にするにはGitHub App installation自体を必要なread permissionだけでprovisionする必要がある。トークンはinstallationの権限を越えず、上流操作は別の外向き通信 ポリシーがGET/HEADへ制限する。

`http.Client`はリダイレクト処理を含むため、この境界では注入された`http.RoundTripper`を一回だけ直接呼ぶ。必須タイムアウトをリクエスト コンテキストへ付けるが、TLS、CA、DNS、プロキシ、IP検証を行うdefault 転送方式は作らない。

## レスポンス境界

成功は次のすべてを満たす場合だけである。

- 状態 `201`、JSON media 型、128 KiB以下。
- 単一最上位 objectで、最上位 フィールドの重複とtrailing JSONがない。
- `token`が一つあり、1〜4,096 バイトのvisible ASCIIである。
- `expires_at`が一つあり、RFC3339で評価時刻より後かつ65分以内である。

未知のレスポンス フィールドは無視する。トークンのprefixや40文字長は仮定せず、GitHubのステートレス形式も同じ上限内なら受理する。レスポンス 本文は成功・失敗の全経路でcloseし、転送方式、parse、closeの詳細は固定エラーへ畳み込む。

期限切れは評価直前にUTC時刻を一度だけ取得して判定する。production APIを増やさず、package-private clock dependencyでnow、now直後、65分、65分超の境界をdeterministicにテストする。

## 状態を増やさない初期形

初期実装はGitHub resolveごとにJWT生成と交換を一回だけ行う。cache、更新、singleflight、再試行、backoffは実測された必要性が出るまで追加しない。これによりトークン再利用窓、競合、失効状態を先回りして持たない。

resolver自身の既定formatはvalue receiverで固定ラベルへredactし、バンドルや注入転送方式の内部を診断へ出さない。

## 適用限界

実GitHub AppでのJWT受理、installationと`owner/name`の対応、リポジトリ スコープ、installationの最小permission、新トークン形式、および実TLS/CA/DNS/プロキシ/IP 転送方式は未確認である。fake 転送方式とunit テストのPASSで代替しない。リクエスト forwarding、HTTP サーバー、Git Smart HTTP、push承認も後続境界である。

## 関連

- [TASK-0043 HANDOVER](../../../tasks/TASK-0043-dev-agent-provider-resolver/HANDOVER.md)
- [Development Agent ハーネス ブローカー認証情報](development-agent-harness-broker-credentials.md)
- [Development Agent ハーネス 外向き通信 トランザクション](development-agent-harness-egress-transaction.md)
