---
kind: schema
title: Development Agent Harness ブローカー認証情報
---

# Development Agent Harness ブローカー認証情報

## 問い

Agentへ実認証情報を見せず、ブローカーだけがGitHub App秘密鍵とOpenAI APIキーを読み、短命なGitHub App JWTを生成する境界をどう作るか。

## 固定ディレクトリ境界

ブローカーは次の固定4ファイルだけを読む。

- `github-client-id`
- `github-installation-id`
- `github-private-key.pem`
- `openai-api-key`

実効UIDはroot以外とし、ディレクトリと各ファイルは同じUIDが所有する。グループとその他利用者の権限を拒否し、ディレクトリは所有者のread/execute、ファイルは所有者のreadを要求して実行bitを拒否する。

Linuxではディレクトリを一度FDとして開き、固定basenameだけを`openat`する。各ファイルは`O_NOFOLLOW|O_CLOEXEC|O_NONBLOCK`で開き、通常ファイル、サイズ上限、読込み前後のFDメタデータ不変を確認する。FIFO、symlink、所有者・権限違反、読込み中の変更は固定errorへ集約する。非Linux実装は開発test用であり、本番のTOCTOU保証として扱わない。

## 入力の検証

text値は末尾LFを0又は1個だけ許し、それ以外はvisible ASCIIだけを受理する。GitHub installation IDはleading zeroなしの正の10進`int64`とする。OpenAI APIキーはprefixを仮定しない。

秘密鍵は余剰dataのない単一PEM blockだけを受理する。PKCS#1又はPKCS#8のunencrypted RSA、2048〜8192 bit、`rsa.PrivateKey.Validate`成功を必要とする。読込み後のバンドルはraw PEM又はprivate keyを返さない。

## GitHub App JWT

JWT headerは`alg=RS256`と`typ=JWT`だけ、payloadは次の3 fieldだけとする。

- `iat`: `now.UTC().Unix()-60`
- `exp`: `now.UTC().Unix()+540`
- `iss`: GitHub App client ID

`iat`と`exp`は文字列や小数でなくJSON数値の整数Unix秒である。RS256のPKCS#1 v1.5署名は決定的なため、同じ秒の同じclaimから同じJWTが得られてよい。nonce、PSS、random claimは追加しない。

## Goの既定formatを封鎖する

Goは非公開fieldも`fmt`の既定struct整形で表示する。pointer receiverだけに`Format`を実装すると、`*bundle`を値copyして整形する経路がredactionを迂回する。

このバンドルはvalue receiverの`Format`で固定labelだけを返す。testはpointerと値copyの双方へ代表的なformat verbを適用し、OpenAI APIキー、client ID、RSA materialが出ないことを確認する。`String`や秘密を含むmarshal面は追加しない。

## 適用限界

この境界は秘密ファイルの作成・rotate・削除、GitHub installation token交換、cache、resolver接続、HTTP通信を実装しない。実UbuntuのUID隔離と実GitHubでのJWT受理も未確認であり、unit testやcross-compileのPASSで代替しない。

## 関連

- [TASK-0042 HANDOVER](../../../tasks/TASK-0042-dev-agent-broker-credentials/HANDOVER.md)
- [Development Agent Harness Egress Transaction](development-agent-harness-egress-transaction.md)
