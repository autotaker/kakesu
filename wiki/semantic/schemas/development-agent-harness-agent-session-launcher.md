---
kind: schema
title: Development Agent Harness Agent Session Launcher
---

# Development Agent Harness Agent Session Launcher

## 問い

既存のcontrol client、loopback bridge、Git credential helperを、一つのcoding-agent childのfail-closedなsession lifecycleへ、実認証情報を露出せずどう束縛するか。

## 固定CLIと初期化

`dev-agent-launcher`のoperationはexact `run --repository owner/repo -- COMMAND [ARG...]`だけである。`--help`と`--version`以外のlauncher option、option順序の変更、重複、欠落、不正又は非canonical lowercaseのrepository、NUL、空commandは、control dial又はchild起動前にusage failureとなる。childはshellを介さず、argv境界とstdioを直接保持する。

起動時はlink-timeで固定されたabsolute cleanなcontrol Unix socketとGit credential helper pathを先に検査する。公開proxy CA、GitHub REST handle、OpenAI handleをこの順で各一回だけ取得し、socket、helper、provider、operation、model、proxy endpointをAgent入力にしない。途中失敗では後段初期化又はchildを開始せず、既に取得したhandleだけを各一回revokeする。

## session構成とchild environment

launcherはliteral `/tmp`直下にfreshな0700 directoryと0600 regular CA fileを一回だけ作り、固定IPv4 loopback bridgeを一つ起動する。child environmentは親の`HOME`、`PATH`、`TERM`、`LANG`/`LC_*`、任意の`CODEX_HOME`だけから再構築する。`GH_TOKEN`と`OPENAI_API_KEY`にはOpaque handleを、HTTP(S) proxyとCA trust変数にはsession専用のloopback/CA値を設定する。

`HOME`又は`CODEX_HOME`を残すことは、Codex credential locationをchildが直接利用できる例外である。launcherはCodex credentialの内容を読み、検証し、複製し、出力し、environmentへ展開しない。親の任意credential、proxy/CA、Git config、SSH、loader又はruntime injection値は継承しない。

Gitにはcommand-scope configでcredential helper列をempty valueでresetしてからabsolute fixed helperだけを設定し、GitHub path-aware credential、proxy、CA、対話prompt無効化を固定する。`GIT_CONFIG_NOSYSTEM=1`と`GIT_CONFIG_GLOBAL=/dev/null`によりsystem/global Git configも除外する。Git helperはGit Smart HTTP readの操作ごとにsingle-use handleを取得し、launcherはGit-read handleを事前発行又はcacheしない。

## bridge、child、cleanup

context cancel又はbridge failureではlauncherがchildを停止・waitし、normal/nonzero child exitでは既にchildのwait結果を受ける。setup/start failureを含む各所有資源のcleanup経路では、bridgeの新規acceptを止めてactive connectionをdrainし、CA directoryを再試行なしで削除してから、発行済みAPI handleを各一回revokeする。normal childは0、通常のnonzero childはそのexit codeを保持し、signal、start、initialization又はcleanup failureは固定nonzeroへ畳む。diagnosticはusage又は固定`session failed`だけであり、handle、repository、path、environment、command、下位errorを含めない。

## 適用限界

launcherとproxy environmentはsecret置換と便利なroutingの境界であり、network enforcementではない。実OSのdefault-deny firewall/network namespace、loopback isolation、Unix socket ownership/peer UID、実credential、実Git/`gh`/Codex/OpenAI client、DNS/TLS、systemd/VPSの配置・restart・rollback・cleanupはlive E2Eで未実施かつblockedである。hermetic testのPASSはこれらを実証しない。

## 関連

- [TASK-0069 HANDOVER](../../../tasks/TASK-0069-agent-session-launcher/HANDOVER.md)
- [Development Agent Harness Loopback Proxy Bridge](development-agent-harness-loopback-proxy-bridge.md)
- [Development Agent Harness API Capability Client](development-agent-harness-api-capability-client.md)
- [Development Agent Harness Git Credential Helper](development-agent-harness-git-credential-helper.md)
- [Development Agent Harness Proxy CA](development-agent-harness-proxy-ca.md)
