---
kind: schema
title: Development Agent Harness ブローカー認証情報
---

# Development Agent Harness ブローカー認証情報

## 問い

Agentへ実認証情報を見せず、ブローカーだけがGitHub App秘密鍵とOpenAI APIキーを読み、短命なGitHub App JWTを生成する境界をどう作るか。

## 固定ディレクトリ境界

ブローカーは次の固定6ファイルだけを一つのstartup snapshotとして読む。

- `github-client-id`
- `github-installation-id`
- `github-private-key.pem`
- `openai-api-key`
- `proxy-ca-cert.pem`
- `proxy-ca-key.pem`

実効UIDはroot以外とし、ディレクトリと各ファイルは同じUIDが所有する。グループとその他利用者の権限を拒否し、ディレクトリは所有者のread/execute、ファイルは所有者のreadを要求して実行bitを拒否する。

Linuxではディレクトリを一度FDとして開き、固定basenameだけを`openat`する。各ファイルは`O_NOFOLLOW|O_CLOEXEC|O_NONBLOCK`で開き、link countが1の通常ファイル、サイズ上限、読込み前後のFDメタデータ不変を確認する。FIFO、symlink、hardlink、所有者・権限違反、読込み中の変更は固定errorへ集約する。非Linux実装は開発test用であり、本番のTOCTOU保証として扱わない。

## 入力の検証

text値は末尾LFを0又は1個だけ許し、それ以外はvisible ASCIIだけを受理する。GitHub installation IDはleading zeroなしの正の10進`int64`とする。OpenAI APIキーはprefixを仮定しない。

秘密鍵は余剰dataのない単一PEM blockだけを受理する。PKCS#1又はPKCS#8のunencrypted RSA、2048〜8192 bit、`rsa.PrivateKey.Validate`成功を必要とする。CA certificate/keyは同じfile policyを通過後にだけ`proxyca`へ委譲し、自己署名ECDSA P-256 CA、CA用途、鍵との一致、期限を検証する。6入力のいずれかが欠ける、空、又は検証不能ならLoadは固定errorで失敗し、部分的なBundleを返さない。読込み後のBundleはraw PEM又はprivate keyを返さず、検証済みProxy CA Authorityへの非漏洩accessorだけを持つ。Authorityの公開certificate exportはcopyであり、呼出し時clockで行うleaf発行と既存JWT利用は相互に状態を共有しない。

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

この境界は秘密又はCA fileの生成・書込み・rotate・reload・watch・削除、Agent trust、GitHub installation token交換、cache、resolver接続、listener composition、HTTP通信を実装しない。実OS user/permission、実GitHub/OpenAI/network、VPS live E2E、実配置のrestart/rollback/cleanupも未確認であり、hermetic testやcross-compileのPASSで代替しない。

## 関連

- [TASK-0042 HANDOVER](../../../tasks/TASK-0042-dev-agent-broker-credentials/HANDOVER.md)
- [TASK-0053 HANDOVER](../../../tasks/TASK-0053-dev-agent-broker-proxy-ca-files/HANDOVER.md)
- [Development Agent Harness Egress Transaction](development-agent-harness-egress-transaction.md)
- [Development Agent Harness Proxy CA](development-agent-harness-proxy-ca.md)
