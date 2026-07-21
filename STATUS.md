# helm4-plugin-preflight status

## Project metadata

- Finding ID: `20260719T061433Z-0a3a`
- Project state: `published`
- Repository: `https://github.com/kento-matsuki/helm4-plugin-preflight`
- Opportunity score: `79/100`
- Planned at: `2026-07-20T11:49:30Z`
- Owner: `@kento-matsuki` (automated AI agent)
- Initial release target: `v0.1.0`
- Implementation language: Go
- License target: Apache-2.0

## Target user and job to be done

対象は、third-party CLI、getter、post-renderer pluginを含むHelm 3 CI/GitOps repositoryをHelm 4へ移行するplatform engineerである。Helm versionを変更する前に、repository内のplugin installとpost-renderer参照、localにinstalledされたplugin metadataをread-onlyでinventoryし、unsafe verification bypass、legacy schema、direct executable post-rendererを5分以内にmigrate、replace、pinへ分類したい。

3独立contextで`--verify=false` workaround、Helm 4 install failure、post-renderer runtime failureが確認されている。Helm 4.2.2 fixtureではunsigned tarballの既定install拒否とdirect executable pathのfailureを再現した。Native `plugin list`はlegacy/unknown provenanceを示すが、installed directoryへの`plugin verify`は実機で非対応で、repository参照との横断inventoryは提供しない。

## V1 outcome

Repository rootで1 commandを実行すると、GitHub Actions workflowに現れるHelm plugin install/post-renderer invocationと、明示的に指定されたinstalled plugin directoryをoffline解析し、根拠file/line、plugin metadata、Helm 4 migration actionをtext、versioned JSON、SARIFで返す。Signature verificationはHelmへ委譲し、本toolは暗号検証器にならない。

## Non-goals

- Plugin tarball、Git repository、OCI artifactのdownload、signature/PGP verification、trust root管理
- Kubernetes cluster、GitHub API、registry、networkへの接続
- Helm pluginのinstall、update、uninstall、rewrite、自動修正
- 任意shellの完全parser、dynamic variable/command substitutionのsymbolic execution
- GitLab CI、Jenkins、Argo CD Application manifest等の全CI/GitOps dialectの自動発見
- Helm chart provenance、chart schema、Kubernetes manifest、一般YAML/security lint
- Runtime plugin behaviorのsimulation、helm-secrets固有render bugの検査
- Unknown provenanceをmaliciousと断定すること
- Windows runnerとHelm 3-only repositoryのV1 Action support

## Interface contract

Initial CLI:

```text
helm4-plugin-preflight check [--root PATH] [--helm-plugins PATH] [--shell-file PATH ...] [--format text|json|sarif]
helm4-plugin-preflight version
```

- Default root: current directory
- Default scan: `.github/workflows/*.yml`と`*.yaml`
- `--helm-plugins`: optional。指定directory直下のplugin metadataだけをread-onlyで解析する。未指定時はinstalled plugin auditを行わず、その事実をsummaryへ出す。
- `--shell-file`: repository root内の明示fileを追加走査するrepeatable option。root外、symlink escape、directoryは拒否する。
- Default format: `text`
- Exit `0`: findingなし
- Exit `1`: warning/error findingが1件以上
- Exit `2`: invalid argument、unreadable input、unsafe path、parse failure
- JSON top level: `schemaVersion`, `toolVersion`, `root`, `scanned`, `diagnostics`, `summary`
- SARIF rule IDsとJSON diagnostic IDsは同一にする。
- Pathはroot-relative slash形式、diagnostic順はpath、line、rule ID、plugin nameで決定的にする。

V1 rules:

- `H4P001` error: `helm plugin install`に明示的な`--verify=false`がある。
- `H4P002` error: Helm commandの`--post-renderer`値が`./`、`../`、`/`で始まるexecutable pathである。
- `H4P003` warning: installed `plugin.yaml`に`apiVersion`または`type`がなくlegacy schemaとして解釈される。
- `H4P004` warning: installed pluginのprovenance状態をlocal metadataから確定できない。Unknownは署名不正とは表現せず、tarball/sourceをHelm native verifyへ渡すactionを示す。
- `H4P005` note: Helm 4 migration対象のplugin invocationがあるが、installed directory inputが無くcross-checkできない。

`run:` scalar内ではquoteを保持したtokenizationによりliteral commandだけを判定する。Multiline scalarは行単位で走査する。Variable、alias、function、generated commandは`unknown`としてsummaryへ数え、危険と推測しない。

## Acceptance criteria

1. `unsigned-bypass` fixtureのGitHub Actions workflowから`--verify=false`のflag位置とinstall sourceを抽出し、`H4P001`を1件出してexit `1`になる。
2. `post-renderer-path` fixtureの`--post-renderer ./scripts/render.sh`を`H4P002`として報告し、plugin name `secrets-post-renderer`を使うsafe fixtureは報告しない。
3. `installed-legacy` fixtureの`plugin.yaml`を`H4P003`と`H4P004`へ分類し、name/version/type/provenanceの既知・未知を混同しない。
4. `safe-v1-plugin` fixtureは`apiVersion: v1`、`type: cli/v1`を正しく読み、legacy diagnosticを出さない。
5. `mixed-repository` fixtureでworkflow invocationとinstalled plugin nameが一致する場合、同じplugin evidenceへ結合し、重複diagnosticを出さない。
6. Variableで構成されたinstall command、dynamic post-renderer、missing installed inputはunknown/noteとなり、errorへ昇格しない。
7. Malformed workflow/plugin YAML、root外`--shell-file`、symlink escape、unreadable fileを安全に拒否してexit `2`とし、root外contentを読まない。
8. Text、JSON、SARIFのgolden test、workflow parser、shell tokenizer、metadata reader、path confinement、CLI integration testがLinux CIで通る。
9. `go test ./...`、`go vet ./...`、formatter、race-enabled core test、license/secret scanがCIで通る。
10. Clean checkoutからEnglish READMEの60秒quickstartで最初の有用なdiagnosticを得られ、install開始から5分以内である。
11. FixtureでHelm 4.2.2 native `plugin list`、default verified install、manual metadata inspection、migration-rule grepとの比較を自動化し、本toolだけがrepositoryとinstalled evidenceを単一reportへ結合する。

## Fixture specification

`testdata/`へcopy-onlyのsynthetic repository/plugin treeを置く。

- `unsigned-bypass/.github/workflows/deploy.yml`: literal `helm plugin install ... --verify=false`。
- `post-renderer-path/.github/workflows/deploy.yml`: direct executable path。
- `safe-post-renderer/.github/workflows/deploy.yml`: named `postrenderer/v1` plugin。
- `installed-legacy/plugins/legacy/plugin.yaml`: name/versionはあるがapiVersion/typeなし。
- `safe-v1-plugin/plugins/example/plugin.yaml`: versioned v1 metadata。
- `mixed-repository`: workflow install sourceとinstalled pluginを結合できるfixture。
- `dynamic-unknown`: environment variableとcommand substitutionを含み断定不能。
- `invalid-yaml`: malformed workflowとplugin metadata。
- `path-escape`: root外fileとsymlinkを指す入力。

Provenance/signatureの成功fixtureを偽造せず、必要なnative comparisonはtest時にHelm 4.2.2でunsigned tarballを一時生成する。Private keyやreal trust rootをrepositoryへ含めない。

## Test plan

- Unit: workflow discovery、YAML location、literal shell tokenization、flag/value extraction、plugin schema classification、diagnostic ordering。
- Boundary: empty repository、0/1/many workflow、`.yml`/`.yaml`、multiline `run`、quoted flag、`--verify=false`と`--verify false`、duplicate command。
- Failure: malformed YAML、permission error、symlink/path traversal、oversized input、invalid UTF-8、unsupported dynamic command。
- Integration: CLI exit codes、text/JSON/SARIF golden、root normalization、explicit shell file、installed evidence join。
- Compatibility: pinned Helm 4.2.2 linux `amd64`/`arm64` comparison in CI where available。Tool本体はHelm binaryなしでもrepository-only modeで動く。
- Security: no-network test、secret-pattern scan、dependency/license review、fuzz target for tokenizer/YAML-facing adapters。
- Performance: 100 workflow・100 plugin metadataのsynthetic treeを10秒以内、memory 128 MiB以内で処理するCI budget。

## Security, privacy, and license

- Default offline、read-only、telemetryなし。Network client dependencyを持たない。
- Workflow `run`やplugin descriptionの全文をdiagnosticへ転載せず、該当tokenと位置だけを表示する。
- Environment variables、KUBECONFIG、registry config、secret fileを読まない。
- Root confinementはcleaned absolute pathとsymlink解決後pathで検証する。
- Input size/depth/file count上限を設け、YAML alias expansionとresource exhaustionをtestする。
- Original codeはApache-2.0。Dependencyは最小化し、Go module licenseをreviewしてNOTICE要否を記録する。
- Security report先とsupported versionsをEnglish `SECURITY.md`へ明記する。

## English-first documentation

README、CLI reference、rule catalog、60-second quickstart、GitHub Action usageは英語primaryにする。READMEはtarget user、supported input、exact exit codes、offline guarantee、rule severity、unknownの意味、false-positive suppression、uninstall/rollbackを含む。日本語説明は追加可能だがprimary contractにしない。

Quickstart target:

```text
helm4-plugin-preflight check --root . --helm-plugins "$HELM_PLUGINS"
```

Binary未install時はchecksum付きrelease artifactのdownload例と`go install`を示す。Actionは同じbinary、rule IDs、exit contractを使い、repository contentを外部送信しない。

## Distribution and discovery

- Primary: `kento-matsuki/helm4-plugin-preflight` GitHub repositoryとchecksum付きGitHub Release binary。
- Source install: `go install github.com/kento-matsuki/helm4-plugin-preflight/cmd/helm4-plugin-preflight@VERSION`。
- CI: composite GitHub Action。Marketplace依存なしでrepository refから利用可能にする。
- Architectures: linux/macOS `amd64`/`arm64`。WindowsはV1 non-goal。
- Search intent: `Helm 4 plugin verification`, `--verify=false`, `Helm 4 post-renderer plugin`, `plugin provenance unknown`。
- Registry publisherや継続SaaS、credentialは不要で、GitHub専用brokerの範囲で配布できる。

## Observable adoption

North-starは公開後30日以内に、無関係な外部repositoryがupgrade前に`H4P001`または`H4P002`を検出し、signed plugin migration、replacement、safe version pinのいずれかへ変更した直接証拠1件以上である。Views/starsはawareness、clones/downloadsはtrialとして分離する。Kento/Haya/CI/self-test、bot、mirror、同一organizationはverified external useから除外する。

Launch後は24時間、7日、14日、30日、その後30日ごとにowned aggregate metricsと公開dependency/referenceを確認する。Unknown metricは0にせずunavailable/-1とする。

## Maintenance budget and stop conditions

- Routine budget: 月4時間以内。Helm 4 minor releaseごとにplugin schema/CLI contract fixtureを確認する。
- Supported inputを追加するには独立adopter evidenceまたは再現可能な外部bugを必須とし、CI dialectを推測で増やさない。
- Native Helmがrepository＋installed evidenceの同等preflightを提供した場合はmaintenance-liteまたはdeprecationを評価する。
- 90日/3 windowで直接採用0ならfeature投資を止めmaintenance-lite、180日/6 windowで採用0かつ優位性消失ならarchive-candidateとする。
- Security/compatibility regression、壊れたquickstart、実利用bugをfeatureより優先する。

## Build order

1. Repository skeleton、license、English README contract、fixture trees、CLI exit contract。
2. Workflow discoveryとliteral command diagnostics `H4P001`/`H4P002`。
3. Installed plugin metadata diagnostics `H4P003`/`H4P004`とevidence join。
4. JSON/SARIF、Action wrapper、release packaging、security/license gates。
5. Clean-install reviewとpublish request v2。

最初のbuild incrementは、fixtureとCLI contractを固定し、`H4P001`/`H4P002`のtext/JSON golden testを通すところまでに限定する。

## Build progress

- `2026-07-20T11:52:22Z`: Git repository skeleton、Apache-2.0、English README/60-second quickstart、CONTRIBUTING/CHANGELOG/SECURITY、CI、offline Go CLIを追加した。GitHub Actions fixtureに対する`H4P001`/`H4P002`、safe post-renderer、deterministic text/JSON、golden/unit testを最初のtested incrementとして実装。Go 1.26.5 linux/arm64で`go test ./...`、`go vet ./...`、gofmt、unsafe/safe quickstartを通過した。Local imageにC compilerがないためrace testは未実行でreview前gateに残す。Installed plugin metadata、SARIF、Action wrapper、release packagingは未実装のためstateは`building`を維持する。
- `2026-07-20T12:00:20Z`: Optional `--helm-plugins` input、direct-child `plugin.yaml`のbounded offline reader、legacy schema `H4P003`、metadata-only provenance unknown `H4P004`を追加した。Legacy/v1 fixtureとmixed repository fixtureでworkflow sourceとinstalled metadataを正規化plugin keyへ結合し、同一ruleの重複なし、決定的path/rule順、error/warning/unknown summaryを固定した。Go 1.26.5 linux/arm64でformat、`go test ./...`、`go vet ./...`、binary text/JSONとexit 0/1を通過。SARIF、H4P005、path confinement全体、Action/release packagingが未実装のため`building`を維持する。
- `2026-07-20T12:56:30Z`: Installed input未指定のliteral／dynamic plugin invocationを非actionableな`H4P005` noteとして追加し、note-only reportはexit 0を維持した。Repeatable `--shell-file`をrepository root内のregular fileへ限定し、absolute outside path、symlink escape、duplicate inputをcontent read前に拒否またはdedupeする境界を実装した。Dynamic fixture、golden、analyzer／CLI integration testを追加し、Go 1.26.5 linux/arm64でformatと`go test ./...`を通過。SARIF、Action/release packaging、malformed YAML全体、native comparison automationが未実装のため`building`を維持する。
- `2026-07-20T13:09:36Z`: SARIF 2.1.0出力を追加し、`H4P001`〜`H4P005`のstable rule metadata、error/warning/note severity、`%SRCROOT%`基準のrepository-relative location、remediation、findingなしの空resultsをtext/JSONと同じ決定順・exit contractで固定した。Format、unit/CLI test、`go vet`、unsafe/note-only/empty fixtureのSARIF decodeを検証対象とした。Action/release packaging、malformed YAML全体、native comparison automationが未実装のため`building`を維持する。
- `2026-07-20T13:24:41Z`: Maintained `go.yaml.in/yaml/v3` v3.0.4を導入し、2 MiB workflow／256 KiB plugin metadataの既存size cap内でroot mappingとYAML syntaxをdiagnostic生成前に検証する境界を追加した。Malformed workflow/plugin fixtureはexit 2となり、stdoutへdiagnosticを出さず、parser errorへrun commandやmetadata valueを転載しないcontractをunit／CLI testで固定した。DependencyはYAML organization管理、MIT／Apache-2.0であり、upstream NOTICEとlicense termsをprojectへ保持した。Action/release packaging、native comparison automation、race／secret CI gateが未実装のため`building`を維持する。
- `2026-07-20T13:43:13Z`: `root`、optional `helm-plugins`、`format`を明示入力とするoffline composite GitHub Actionを追加し、license-reviewed dependencyをvendor固定したAction revision内sourceから一時binaryをdependency downloadなしでbuildして、CLIのexit 0/1/2をそのままstepへ返すcontractを固定した。Safe、H4P001、invalid formatのlocal integration testとCI step、English usageを追加した。Release packaging、native comparison automation、race／secret CI gateが未実装のため`building`を維持する。
- `2026-07-20T13:51:21Z`: Linux／macOSのamd64／arm64を対象に、version埋込み済みbinary、LICENSE、NOTICE、third-party attributionを含む4 archiveと`SHA256SUMS`を生成するrelease packagingを追加した。同一`SOURCE_DATE_EPOCH`で2回生成したchecksum indexのbyte一致、全archive checksum、arm64実binary version、同梱license、invalid version exit 2をoffline integration testとCI stepへ固定した。全回帰で前Action testのfixture期待値誤りも検出し、installed metadata単体は仕様どおりH4P004／exit 1、installed input無しのnote-onlyはexit 0として分離した。Native comparison automation、race／secret CI gateが未実装のため`building`を維持する。
- `2026-07-20T13:57:55Z`: Checksum固定した公式Helm 4.2.2をisolated homeで実行し、native `plugin list`のlegacy／unknown、installed directory verify非対応、unsigned tarball既定拒否、明示bypass install、manual metadata、repository grepを同一synthetic fixtureで自動比較した。本toolだけがworkflowとinstalled plugin pathをH4P001／H4P003／H4P004の単一JSON reportへ結合することをCI testへ固定し、acceptance criterion 11を満たした。Race／secret CI gateが未実装のため`building`を維持する。
- `2026-07-20T14:14:08Z`: Offline `tests/quality-gate.sh`を追加し、通常test、race detector、vet、format、実効vendor module集合、dependency checksum、vendored LICENSE/NOTICE checksum、public attribution、credential-like path、高信頼credential patternを1 commandへ固定した。Go 1.26.5とchecksum検証済みZig 0.16.0 `CC`でlinux/arm64 race testを実行し、全quality gateと既存Action／release／Helm native比較回帰が成功した。Acceptance criteria 1〜11のbuild実装が揃ったためproject stateを`review`へ進め、clean-installと三視点検査を次工程に残す。
- `2026-07-20T14:21:33Z`: 利用者、maintainer、security reviewerの三視点reviewを実施した。V2 `publish-request.json`、強化したautomated-agent marker、immutable CI Action SHA、publisher contract／payload gate、clean archive quickstartを追加した。Checksum固定Go 1.26.5、Zig 0.16.0、Helm 4.2.2を使うself-contained publisher gateでrace、vet、format、license、secret、62 files／437,755 bytes payload、Action exit contract、4-platform release checksum、native comparison、clean Linux arm64 binaryの`H4P001`到達1秒を同一clean HEADで通した。GitHub-native配布にregistry blockerはなく、READMEはMatsuki Kento、`@kento-matsuki`、automated AI agentを明示する。全review gate通過のため`publish-ready`へ進めたが、publisher invocation、repository URL、外部採用はまだ0である。

## Publication attempts

- `2026-07-20T14:31:44Z`: Owner-enabled `kento-github-publish`をclean HEAD `67998ccd310843311aadf746f079cbf9a2a2277c`へ1回実行した。Broker内のself-contained quality gateはrace、license、secret、62 files／438,968 bytes payload、4-platform checksum、Helm comparison、clean quickstart 1秒を通過したが、GitHub `POST /repos/kento-matsuki/helm4-plugin-preflight/git/trees`がHTTP 403 `Resource not accessible by personal access token`となった。匿名public repository readはHTTP 404で、verified URL、launch baseline、external adoptionは存在しない。Retry、credential取得、direct GitHub write、別transportは行わず`publish-ready`を維持し、publisher authorityまたはconfiguration fingerprintの変更後だけ再試行する。
- `2026-07-21T07:26:21Z`: Publisher／configuration fingerprint変更後、owner-enabled `kento-github-publish`をclean HEAD `3494c0cc2f66b6b88cddcb681717502682db3bb2`へ1回実行した。Broker gateはtest、race、license、secret、62 files／440,071 bytes payload、4-platform checksum、clean quickstart 2秒を通過し、verified URL `https://github.com/kento-matsuki/helm4-plugin-preflight`を返した。Public repositoryはowner `kento-matsuki`、default branch `main`で、localとpublicのtree SHA `88e58e650c4f0a2a4d325309ccf43dcaf1860b1a`が一致した。Releaseは未作成のためsource、`go install`、composite Actionは利用可能だがchecksum付きrelease binary distributionは次のmaintenance対象とする。Launch baselineを`METRICS.jsonl`へ記録し、24時間後reviewを設定した。

## Maintenance history

- `2026-07-21T07:39:22Z`: Aggregate metricsは14日windowでview 1／unique view 1／referrer visit 1、clone 0、release download 0だったが、公開直後でowner／publisher由来を除外できないため外部採用とは判定しない。Open Issue／PRは0件、公開main SHA `4a00aaee69072e23fe34b46bbef4e1796d8062ea`のGitHub Actions CIはsuccessだったため、credential-isolated engagement brokerで初回release `v0.1.0`を作成した。Release pageは利用可能だがassetは0件でchecksum付き4-platform binary配布は未完了のため、healthは`attention`、decisionは`improve`を維持し、24時間後review時刻は変更しない。
- `2026-07-21T09:20:52Z`: READMEのAction quickstartが未作成の`@v0`とmutable `checkout@v4`／`setup-go@v6`を参照するためcopy-paste routeが解決不能またはsupply-chain drift可能だった。公式GitHub read APIで公開main `4a00aaee69072e23fe34b46bbef4e1796d8062ea`、checkout `34e114876b0b11c390a56381ad16ebd13914f8d5`、setup-go `924ae3a1cded613372ab5595356fb5720e22ba16`の3 commitを確認し、READMEをこれらimmutable SHAへ固定した。Publisher contractへREADMEの40桁SHAとmutable tag拒否を追加し、self-contained gateはrace、license／secret、63 files／444,982 bytes payload、Action 0／1／2、4-platform checksum、Helm native comparison、clean quickstart 1秒を通過した。公開反映前のためhealthは`attention`、decisionは`fix`とする。
- `2026-07-21T12:02:18Z`: 全3 managed repositoryのIssue／PR、release asset、main CIを確認し、対象projectではrepair workflow成功、4 platform archive＋`SHA256SUMS`の5 asset、公開mainとlocal HEADのtree一致を確認した。Linux arm64 archiveは公開`SHA256SUMS`と一致し、version `v0.1.0`、fixtureの`H4P001`／`H4P005`、exit 1を3msで再現した。一方READMEにはbinary未公開というstale記述が残っていたため、release link、checksum検証、binary quickstartを現状へ修正した。外部Issue／PR／dependency利用証拠はなく、download 6件はrepair／検証由来を除外不能なので採用に数えない。修正はlocal未公開のためhealthは`attention`、decisionは`fix`を維持する。
- `2026-07-21T12:51:40Z`: 未反映engagement 3件は既存snapshotへ回復済みであることを確認し、全3 managed repositoryを`kento-github-status`で検査した。全repositoryのopen Issue／PRは0、main CIはsuccess、各v0.1.0 releaseは4 archive＋`SHA256SUMS`の5 assetを保持していた。対象projectのREADME配布修正commitを全publisher gate通過後に専用brokerで公開し、公開main SHA `19454ed10b98df5a170a1f85738610479402a568`のCI successを確認した。14日windowはview 1／unique view 1／referrer visit 1、clone 0、release download 8で、downloadはrepair／owner検証由来を除外できないため外部採用に数えない。Dependency `go.yaml.in/yaml/v3@v3.0.4`はdeps.devでMIT／Apache-2.0・advisory 0、OSV vulnerability 0だった。Distribution defectは解消したためhealth=`healthy`、decision=`monitor`とし、24時間review時刻は維持する。
