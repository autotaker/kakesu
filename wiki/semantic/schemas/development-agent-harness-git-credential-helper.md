---
kind: schema
title: Development Agent Harness Git Credential Helper
---

# Development Agent Harness Git Credential Helper

## 問い

実認証情報をAgent又はGitのcredential cacheへ渡さず、allowlistされたGitHub HTTPS readのために一回限りのOpaque handleだけをGit credential helperへどう供給するか。

## helper入力と出力

`git-credential-dev-agent`は一個のoperation argと4 KiB以下のcredential inputだけを扱う。`get`は、重複・URL属性・NUL/CR・余分なbyteを拒否し、`protocol=https`、`host=github.com`又は`github.com:443`、正規なlowercase `owner/repo.git`の三属性からだけrepositoryを導出する。

成功時はcredential formatで`username=x-access-token`と`password=cap_...`だけをstdoutへ返す。実token、repository、socket path、入力値、下位errorは出力又は診断へ含めない。入力又はcontrolの拒否時は`quit=true`だけを返し、他helper又は対話promptへの探索を停止する。

`store`はbounded inputを読み捨て、保存せず出力なしで成功する。`erase`はcanonicalなOpaque handle一件だけを同じcontrol socketで失効する。未知operationは出力なしで無視し、zero又は複数argは固定usage failureとなる。

## strict control clientと固定socket

clientはconfigure済み`runstatedir/dev-agent-harness/egress.sock`からlink時に固定されたabsolute Unix socketへ、一接続一操作、deadline付きで接続する。environment、credential input、CLI flag、config又はcwdによる接続先のoverride、TCP、fallback、retryは持たない。

`get`は`provider=github`、repository、`operation=github-git-read`のexact Issue wireを送り、唯一のboundedな200 JSON handle responseだけを受理する。`erase`はcanonical handleに対するexact DELETE wireと204 responseだけを受理する。chunked又は余分なheader/body/bytes、malformed response、early EOF、非canonical handle、非成功statusは固定拒否へ縮退する。成功・失敗ともconnectionはcloseし、control protocolの値を診断へ反映しない。

## 非漏洩と適用外

helperはGit config、`credential.useHttpPath`、proxy/CA environment、launcher、clone/fetch/pull、push、credentialのdisk/cache/state保存を変更しない。GitHub App tokenの取得・置換、Capability Registry/control server、Smart HTTP policyも所有しない。発行したhandleの利用時の実token置換とallowlist再検証は既存egress transactionの境界に残る。

hermetic stream、fake dialer、`net.Pipe`の確認はwire、順序、上限、close/deadline、拒否、非漏洩を対象とする。実配置socket、実Git invocation、別UID、GitHub、DNS/TLS、systemd/VPS、restart/rollback/cleanupはlive E2Eとして未確認であり、hermetic PASSで代替しない。

## 関連

- [TASK-0065 HANDOVER](../../../tasks/TASK-0065-git-credential-helper/HANDOVER.md)
- [Development Agent Harness Capability Registry](development-agent-harness-capability-registry.md)
- [Development Agent Harness Egress Transaction](development-agent-harness-egress-transaction.md)
