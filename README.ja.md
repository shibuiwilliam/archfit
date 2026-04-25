![archfit logo](./archfit_logo.png)

# archfit

> **コーディングエージェント時代のアーキテクチャ適性評価ツール**
> あなたのリポジトリは、コーディングエージェントが*安全*かつ*迅速*に作業できる形になっていますか？

![CI](https://github.com/shibuiwilliam/archfit/actions/workflows/ci.yml/badge.svg)
![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)

[English](./README.md)

---

多くのツールは*コード*をチェックします。archfit はコードが置かれている**地形**をチェックします。
エージェントが最初に読むエントリーポイント、フィードバックループの速さ、
ひとつの不適切な変更がすべてを静かに壊しうる箇所を評価します。

コーディングエージェント（そして新しく参加する人間の開発者）が、シニアエンジニアに
全ての diff を見てもらわなくても成功できるかどうかを決定する、7つのアーキテクチャ
特性を評価します：

| | 原則 | 問いかけ |
|---|---|---|
| **P1** | 局所性 (Locality) | 変更はリポジトリの狭い範囲だけで理解できるか？ |
| **P2** | 仕様優先 (Spec-first) | 契約はプロダクションの文章ではなく、スキーマや型か？ |
| **P3** | 浅い明示性 (Shallow explicitness) | リフレクションや深い間接参照を追わなくても振る舞いが見えるか？ |
| **P4** | 検証可能性 (Verifiability) | 正しさを数秒でローカルに証明できるか？ |
| **P5** | 危険の集約 (Aggregation of danger) | 認証・秘密情報・マイグレーションは集中して保護されているか？ |
| **P6** | 可逆性 (Reversibility) | あらゆる変更を低コストでロールバックできるか？ |
| **P7** | 機械可読性 (Machine-readability) | エラー・ADR・CLI は機械にも読めるか？ |

archfit はリンターでも SAST スキャナーでも**ありません**。
それらのツールの*上位*に位置し、それらが測定しないアーキテクチャ上の特性をレポートします。

---

## クイックスタート

```bash
# ソースからビルド（Go 1.24+、CGO不要）
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit
make build

# リポジトリに設定ファイルを生成
./bin/archfit init /path/to/your/repo

# スキャン実行
./bin/archfit scan /path/to/your/repo
```

Docker でも実行可能：

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

### 出力例

クリーンなスキャン：

```
archfit dev — target . (profile=standard)
rules evaluated: 10, findings: 0
overall score: 100.0
  P1: 100.0
  P2: 100.0
  P3: 100.0
  P4: 100.0
  P5: 100.0
  P6: 100.0
  P7: 100.0
no findings
```

改善点が見つかった場合：

```
archfit dev — target . (profile=standard)
rules evaluated: 10, findings: 2
overall score: 84.0
  P1: 100.0
  P3: 60.0
  P6: 60.0
  ...
findings:
  [warn] P3.EXP.001  — repository uses .env files but has no .env.example
  [warn] P6.REV.001 docs/ — deployment artifacts detected but no rollback documentation
```

すべての検出結果にはエビデンス、信頼度、修正ガイドが付属します。
多くの検出結果は自動修正可能です：

```bash
# 特定の検出結果を修正
./bin/archfit fix P3.EXP.001 .

# 修正可能な全ての検出結果を一括修正
./bin/archfit fix --all .

# 変更内容をプレビュー
./bin/archfit fix --dry-run --all .
```

---

## コマンド

### スキャンと分析

```
archfit scan [path]                  有効な全ルールを実行（デフォルト: .）
archfit check <rule-id> [path]       単一ルールを実行
archfit score [path]                 サマリーのみ（検出結果リストなし）
archfit report [path]                Markdownレポート（scan --format=md の省略形）
archfit explain <rule-id>            ルールの根拠と修正方法を表示
```

### 比較とトラッキング

```
archfit diff <baseline.json> [current.json]
                                     2つのスキャンの検出結果を比較 — 回帰があれば exit 1
archfit trend                        アーカイブされたスキャンのスコア推移を表示
archfit compare <f1.json> <f2.json> [...]
                                     リポジトリ間のスキャンを並べて比較
```

### 修正

```
archfit fix [rule-id] [path]         検出結果を自動修正（strongエビデンスルール）
```

### 設定とパック

```
archfit init [path]                  .archfit.yaml を生成
archfit validate-config [path]       スキャンせずに設定を検証
archfit list-rules                   登録済み全ルール一覧
archfit list-packs                   パックとルールID一覧
archfit validate-pack <path>         パック構造を検証（AGENTS.md, resolvers/, fixtures/）
archfit new-pack <name> [path]       新しいルールパックを生成
archfit test-pack <path>             パックテストを実行
archfit version                      バージョン表示
```

### 主要フラグ

| フラグ | 説明 | デフォルト |
|---|---|---|
| `--format {terminal\|json\|md\|sarif}` | 出力形式 | `terminal` |
| `--json` | `--format=json` の省略形 | |
| `--fail-on {info\|warn\|error\|critical}` | この重大度以上で exit `1` | `error` |
| `--config <file>` | 設定ファイルのパス | ターゲットの `.archfit.yaml` |
| `--depth {shallow\|standard\|deep}` | スキャン深度（`deep` は検証コマンドを実行） | `standard` |
| `--policy <file>` | 組織ポリシーファイル（JSON） | |
| `-C <dir>` | 実行前にディレクトリを変更 | |
| `--with-llm` | LLMによる説明で検出結果を補強 | off |
| `--llm-backend {claude\|openai\|gemini}` | LLMプロバイダー | 自動検出 |
| `--llm-budget N` | 1回あたりのLLM呼び出し上限 | `5` |

`fix` コマンド専用: `--all`, `--dry-run`, `--plan`, `--json`。
`trend` コマンド専用: `--history <dir>`, `--since <date>`, `--format {terminal|json|csv}`。
`compare` コマンド専用: `--format {terminal|json|csv|md}`, `--sort {overall|name}`。

### 終了コード

| コード | 意味 |
|:---:|---|
| `0` | 成功（または: 全検出結果が `--fail-on` 閾値未満） |
| `1` | `--fail-on` 以上の検出結果あり |
| `2` | 使用方法エラー |
| `3` | ランタイムエラー |
| `4` | 設定エラー |

終了コードは安定性契約の一部です — 詳細は [`docs/exit-codes.md`](./docs/exit-codes.md)。
`1` はクラッシュではなく「JSON出力を読み込むシグナル」です。

---

## ルールセット — 全7原則をカバー

2つのパックに計10ルール。すべて `strong` エビデンス、`experimental` 安定性。

### `core` パック（7ルール） — すべてのリポジトリに適用

| ID | 原則 | 検査内容 |
|---|---|---|
| [`P1.LOC.001`](./docs/rules/P1.LOC.001.md) | 局所性 | リポジトリルートに `CLAUDE.md` または `AGENTS.md` が存在する |
| [`P1.LOC.002`](./docs/rules/P1.LOC.002.md) | 局所性 | 垂直スライスディレクトリが独自の `AGENTS.md` を持っている |
| [`P3.EXP.001`](./docs/rules/P3.EXP.001.md) | 浅い明示性 | 設定の文書化: `.env` ファイル、Spring Bootプロファイル、Terraform tfvars、Rails environments（[詳細は下記](#言語スタック対応)） |
| [`P4.VER.001`](./docs/rules/P4.VER.001.md) | 検証可能性 | 高速な検証エントリーポイントが存在する（Makefile、package.json、go.mod、pom.xml、build.gradle、Gemfile、Cargo.toml [他20種以上](#言語スタック対応)） |
| [`P5.AGG.001`](./docs/rules/P5.AGG.001.md) | 危険の集約 | セキュリティ関連ファイル（認証、秘密情報、マイグレーション、デプロイ）が分散せず集中している |
| [`P6.REV.001`](./docs/rules/P6.REV.001.md) | 可逆性 | デプロイ成果物がある → ロールバックドキュメントが必要 |
| [`P7.MRD.001`](./docs/rules/P7.MRD.001.md) | 機械可読性 | CLIリポジトリが終了コードを文書化している |

### `agent-tool` パック（3ルール） — オプトイン、エージェント向けツール用

| ID | 原則 | 検査内容 |
|---|---|---|
| [`P2.SPC.010`](./docs/rules/P2.SPC.010.md) | 仕様優先 | `$id` 付きバージョン管理されたJSON Schemaを提供（OpenAPI、Protobuf、GraphQL、Avro、AsyncAPIも認識） |
| [`P7.MRD.002`](./docs/rules/P7.MRD.002.md) | 機械可読性 | リポジトリルートに `CHANGELOG.md` が存在する |
| [`P7.MRD.003`](./docs/rules/P7.MRD.003.md) | 機械可読性 | CLIリポジトリが `docs/adr/` にADRを記録している |

追加パック（`web-saas`、`iac`、`mobile`、`data-event`）は計画中です。
100の弱いルールよりも、10の堅牢な `strong` エビデンスルールの方が優れています。

---

## 言語・スタック対応

archfit は設計上、言語非依存です。ルールは言語構文ではなく、アーキテクチャの地形を検査します。
各ルールがスタック横断で認識する内容を以下にまとめます：

### P4.VER.001 — 検証エントリーポイント

Go (`go.mod`)、Node/TypeScript (`package.json`)、Python (`pyproject.toml`)、
Rust (`Cargo.toml`)、Java (`pom.xml`、`build.gradle`、`build.gradle.kts`、
`settings.gradle`、`settings.gradle.kts`)、Ruby (`Gemfile`、`Rakefile`)、
PHP (`composer.json`)、Elixir (`mix.exs`)、Scala (`build.sbt`)、
C/C++ (`CMakeLists.txt`、`meson.build`)、Deno (`deno.json`、`deno.jsonc`)、
Bazel (`BUILD.bazel`)、Earthly (`Earthfile`)、汎用タスクランナー
(`Makefile`、`justfile`、`Taskfile.yml`)。

### P3.EXP.001 — 設定の文書化

4つの設定エコシステムを独立して検査：

| エコシステム | 検出される設定ファイル | 期待されるドキュメント |
|---|---|---|
| Node/Python/Ruby | `.env`、`.env.*` | `.env.example`、`.env.sample`、`.env.template` |
| Spring Boot | `application-*.yml`、`application-*.yaml`、`application-*.properties` | `config/README.md` または `docs/config.md` |
| Terraform | `*.tfvars` | `terraform.tfvars.example` または `example.tfvars` |
| Rails | `config/environments/*.rb` | `config/README.md` または `docs/config.md` |

### P1.LOC.002 — 垂直スライスコンテナ

`packs/`、`services/`、`modules/`、`packages/`、`apps/`、`libs/`、
`plugins/`、`engines/`、`components/`、`domains/`、`features/` — モノレポ
（NX、Turborepo、Lerna）、DDDプロジェクト、Railsエンジン、プラグイン
アーキテクチャ、サービス指向リポジトリをカバー。

### P6.REV.001 — デプロイ成果物

Docker (`Dockerfile`、`docker-compose.yml`、`compose.yml`)、
Kubernetes (`kubernetes/`、`k8s/`)、Helm (`helm/`)、
Terraform (`terraform/`)、AWS CDK (`cdk.json`、`cdk/`)、
Serverless Framework (`serverless.yml`)、Cloud Build (`cloudbuild.yaml`)、
Skaffold (`skaffold.yaml`)、PaaS (Vercel、Netlify、Fly.io、Render、
Railway、Heroku `Procfile`)、CI (GitHub Actions、CircleCI、
GitLab CI、Buildkite)。

### P7.MRD.001 — CLI検出

`cmd/`、`bin/`、`exe/` ディレクトリ内の11言語のソースファイル
(`.go`、`.py`、`.ts`、`.js`、`.rs`、`.rb`、`.java`、`.kt`、`.swift`、
`.php`、`.sh`) に加え、インジケータファイル (`__main__.py`、`cli.go`、
`cli.py`、`cli.ts`、`cli.js`、`cli.rb`) でCLIエントリーポイントを検出。

### P2.SPC.010 — 仕様優先フォーマット

JSON Schema (`schemas/*.schema.json` + `$id`)、OpenAPI/Swagger
(`openapi.yaml`、`swagger.json`)、Protocol Buffers (`.proto`)、
GraphQL (`.graphql`、`.gql`)、Apache Avro (`.avsc`)、
AsyncAPI (`.asyncapi`)。

---

## 自動修正

`archfit fix` はスキャン → 修正 → 検証のループを完結させます。
決定的な修正を持つ全ルールに対応する7つのスタティックフィクサーを搭載：

| ルール | フィクサーが生成するもの |
|---|---|
| P1.LOC.001 | リポジトリルートの `CLAUDE.md` |
| P1.LOC.002 | 各スライスディレクトリの `AGENTS.md` |
| P4.VER.001 | test/lint ターゲット付き `Makefile` |
| P7.MRD.001 | `docs/exit-codes.md` |
| P7.MRD.002 | `CHANGELOG.md`（Keep a Changelog 形式） |
| P7.MRD.003 | `docs/adr/0001-initial-architecture.md` |
| P2.SPC.010 | `$id` 付き `schemas/output.schema.json` |

```bash
archfit fix P1.LOC.001 .             # 1つのルールを修正
archfit fix --all .                  # 修正可能な全てを一括修正
archfit fix --plan --all .           # 適用せずにプランを確認
archfit fix --dry-run P7.MRD.002 .   # 変更内容を表示
archfit fix --json --all .           # 自動化向けJSON出力
```

すべての修正は**自動再スキャンにより検証されます**。検出結果が残るか新しい検出結果が
出現した場合、変更はロールバックされます。修正アクションは `.archfit-fix-log.json` に
記録されます。

LLMアシストフィクサーは `--with-llm` 設定時にスタティックテンプレートを
リポジトリ固有のコンテキストで補強します。

---

## 適性の時系列トラッキング

### Diff — PR回帰ゲート

```bash
# ベースラインと比較（新しい検出結果があれば exit 1）
archfit diff baseline.json current.json
archfit diff baseline.json              # currentをstdinから読み込み
```

### Trend — スコア推移

```bash
# スキャンをアーカイブして推移を表示
archfit scan --json . > .archfit-history/2026-04-25.json
archfit trend --history .archfit-history
archfit trend --since 2026-01-01 --format csv
```

### Compare — リポジトリ間比較

```bash
# 複数リポジトリを並べて比較
archfit compare api.json frontend.json infra.json
archfit compare --format md --sort name *.json
```

---

## エビデンスであり、判決ではない

すべての検出結果は4つの品質を持ちます：

- **重大度 (Severity)** — もし真なら、どの程度深刻か？ (`info` / `warn` / `error` / `critical`)
- **エビデンス強度 (Evidence strength)** — 検出はどの程度決定論的か？ (`strong` / `medium` / `weak` / `sampled`)
- **信頼度 (Confidence)** — 0.0–1.0 の数値
- **修正方法 (Remediation)** — 概要と詳細ガイドへのリンク

archfit は意図的に保守的です：`error` の重大度には `strong` のエビデンスが必要です。
**誤検知はバグとして扱います。**

JSON出力は決定論的に並び替えられ（severity desc, rule_id asc, path asc）、
エージェントが安定した参照を行い、`archfit diff` が信頼性の高い差分を生成できます。

### スコアリング

各検出結果は、その重大度に応じて所属原則のスコアにペナルティを与えます：

| 重大度 | ペナルティ（ルール重みの%） |
|---|---|
| info | 10% |
| warn | 40% |
| error | 80% |
| critical | 100% |

スコアは0–100で、原則別と全体の両方を
`100 × (1 − ペナルティ / 合計重み)` で計算。ルールの追加によりスコアが
膨張することはありません — スコアリングは重み付けで正規化されます。

### メトリクス

検出結果とともに6つの定量メトリクスを計算：

| メトリクス | 原則 | 測定内容 |
|---|---|---|
| `context_span_p50` | P1 | コミットあたりの変更ファイル数の中央値 |
| `parallel_conflict_rate` | P1 | マージコミットの頻度 |
| `verification_latency_s` | P4 | テスト実行時間（deepスキャンのみ） |
| `invariant_coverage` | P4 | error以上の検出結果がないルールの割合 |
| `blast_radius_score` | P5 | パッケージの最大推移到達範囲 |
| `rollback_signal` | P6 | リバートコミットの頻度 |

---

## 設定

archfit はターゲットディレクトリの `.archfit.yaml` を読み込みます。
`--config` で特定ファイルを指定可能：

```bash
archfit scan .                              # デフォルトの探索
archfit scan --config .archfit.all.yaml .   # 明示的に指定
```

設定を生成：

```bash
archfit init .
```

```json
{
  "version": 1,
  "project_type": [],
  "profile": "standard",
  "packs": { "enabled": ["core"] },
  "ignore": []
}
```

リスクティアと期限付き抑制を含む詳細な例：

```json
{
  "version": 1,
  "project_type": ["agent-tool"],
  "profile": "standard",
  "risk_tiers": {
    "high":   ["src/auth/**", "infra/**", "migrations/**"],
    "medium": ["src/features/**"],
    "low":    ["docs/**", "tests/**"]
  },
  "packs": { "enabled": ["core", "agent-tool"] },
  "ignore": [
    {
      "rule": "P1.LOC.002",
      "paths": ["packs/legacy-*"],
      "reason": "Legacy slices on a documented deletion path",
      "expires": "2026-12-31"
    }
  ]
}
```

すべての `ignore` エントリには `reason` と `expires` 日付が必須です。
期限切れの抑制は警告として表示されます — サイレントに腐敗しません。
詳細は [`docs/configuration.md`](./docs/configuration.md)。

### 適性コントラクト

機械的に強制可能な適性目標を求めるチームのために、archfit は
`.archfit-contract.yaml` でコントラクトをサポート：

```json
{
  "version": 1,
  "hard_constraints": [
    { "principle": "overall", "min_score": 80.0, "scope": "**" }
  ],
  "soft_targets": [
    { "principle": "P4", "target_score": 95.0, "deadline": "2026-06-30" }
  ],
  "area_budgets": [
    { "path": "src/auth/**", "max_findings": 0, "owner": "@security-team" }
  ]
}
```

ハード制約はスキャンを失敗させます。ソフトターゲットは努力目標です。
エリアバジェットはディレクトリ単位のSREスタイルの検出結果上限を設定します。

### 組織ポリシー

`--policy` フラグで、全リポジトリに最低スコア・必須パック・重大度オーバーライドを
強制する組織レベルのポリシーを適用：

```bash
archfit scan --policy org-policy.json .
```

---

## 仕組み

```
          +------------------------------+
          |          archfit CLI          |
          +--------------+---------------+
                         |
     +-------------------+---------------------+
     |                   |                      |
+----v------+   +--------v---------+   +--------v-------+
| Collectors|   |    Rule Packs    |   |   Renderers    |
| fs, git,  |   |  core (7 rules)  |   | terminal, json,|
| schema,   |   |  agent-tool (3)  |   | md, SARIF 2.1.0|
| depgraph, |   +--------+---------+   +--------+-------+
| command   |            |                      |
+-----------+  +---------+--------+   +---------v-------+
               |    Fix Engine    |   |   LLM Adapter   |
               | 7 static fixers  |   | Claude | OpenAI |
               | + LLM-assisted   |   | Gemini          |
               +------------------+   +-----------------+
                                        (オプトインのみ)
```

- **Collectors** はファイルシステム、git履歴、JSONスキーマ、依存グラフ（Go）、コマンド実行時間からファクトを収集。観察のみ、判断なし。
- **Rule packs** はルールを宣言し、リゾルバー関数を実装。リゾルバーは読み取り専用 `FactStore` の純粋関数で、I/Oなし。これはarchfit自身のP5（危険の集約）を自分自身に適用したものです。
- **Fix engine** は各検出結果に対して決定的なファイル変更を生成し、再スキャンで検証。LLMフィクサーは `--with-llm` でコンテキスト依存の内容を生成。
- **Renderers** は複数形式で出力。JSONは [`schemas/output.schema.json`](./schemas/output.schema.json) に準拠。SARIF 2.1.0はGitHub Code Scanningと統合。
- **LLM adapter** は単一のネットワーク境界。Claude、OpenAI、Geminiの3バックエンド。`--with-llm` でのみ起動。基本スキャンはAPIキーの有無に関わらず同一。

ルール登録は `cmd/archfit/main.go` で明示的に行います。
リフレクションなし、`init()` 自動検出なし、プラグインマジックなし。

設計根拠：
[ADR 0001](./docs/adr/0001-architecture-overview.md)、
[ADR 0002](./docs/adr/0002-phase2-dogfood-and-sarif.md)、
[ADR 0003](./docs/adr/0003-llm-explanation.md)、
[ADR 0004](./docs/adr/0004-fix-engine.md)。

---

## CI連携

### GitHub Code Scanning向けSARIF

```yaml
- name: Build archfit
  run: go install github.com/shibuiwilliam/archfit/cmd/archfit@latest

- name: Scan
  run: archfit scan --format=sarif . > archfit.sarif

- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: archfit.sarif
```

### 新規検出結果のみでPRゲート

```yaml
- name: ベースライン (main)
  run: archfit scan --json . > baseline.json

- name: 差分 (PR)
  run: archfit diff baseline.json   # 新しい検出結果がある場合 exit 1
```

### CIでの自動修正

```yaml
- name: 修正してコミット
  run: |
    archfit fix --all .
    git diff --quiet || git commit -am "chore: archfit auto-fix"
```

---

## LLMアシスト説明（オプトイン）

静的な修正ガイドは*一般的に何をすべきか*を教えます。`--with-llm` は
*あなたのリポジトリでなぜ*トリガーされたか、*具体的にどう変更すれば*
修正できるかを教えます。デフォルトのスキャンパスには一切触れません。

### 対応プロバイダー

| プロバイダー | 環境変数 | デフォルトモデル | `--llm-backend` |
|---|---|---|---|
| Claude (Anthropic) | `ANTHROPIC_API_KEY` | `claude-sonnet-4-6-20250627` | `claude` |
| OpenAI | `OPENAI_API_KEY` | `gpt-5.4-mini` | `openai` |
| Google Gemini | `GOOGLE_API_KEY` / `GEMINI_API_KEY` | `gemini-2.5-flash` | `gemini` |

自動検出の優先順位: `ANTHROPIC_API_KEY` > `OPENAI_API_KEY` > `GOOGLE_API_KEY`。

```bash
export ANTHROPIC_API_KEY=sk-...
archfit scan --with-llm .                # 検出結果を補強
archfit explain --with-llm P3.EXP.001   # ルールを説明
archfit fix --with-llm --all .           # コンテキスト依存の自動修正
```

### 安全性の保証

- **オプトインのみ。** 基本の `archfit scan` はLLM呼び出しゼロ。
- **コスト上限あり。** `--llm-budget N` で呼び出し数を制限（デフォルト5）。キャッシュヒットは無料。
- **スキャンを失敗させない。** APIエラーは静的修正にグレースフルにフォールバック。
- **最小限のデータ送信。** ルールメタデータ＋検出エビデンスのみ。ソースコード、ファイル内容、git履歴は送信されません。

詳細な契約: [`docs/llm.md`](./docs/llm.md)。

---

## Claude Codeエージェントスキル

archfit は [`.claude/skills/archfit/`](./.claude/skills/archfit/) にClaude Codeエージェント
スキルを同梱。このリポジトリ内でClaude Codeを実行すると自動検出されます。
スキルはスキャン → 修正 → 検証のループを駆動：

1. **実行**: `archfit scan --json .`
2. **読み取り**: `findings[]` 配列
3. **修正**: `archfit fix <rule-id>` または `reference/remediation/` から修正ガイドを読み込む
4. **検証**: 再スキャン — 再スキャンが証明であり、主張ではない

10件のルールごとの修正ガイドが `.claude/skills/archfit/reference/remediation/` に同梱。
各ガイドには決定木があり、自動修正すべき場合とユーザーに確認すべき場合を指示します。

別のリポジトリで使用するには、`.claude/skills/archfit/` をそのプロジェクトの
`.claude/skills/` ディレクトリにコピーしてください。

---

## インストール

### ソースから

```bash
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit
make build
./bin/archfit version
```

**Go 1.24+** が必要。CGO不要。`linux/{amd64,arm64}`、`darwin/{amd64,arm64}`、
`windows/amd64` へのクロスコンパイルが可能。

### リリースバイナリから

```bash
# Linux/macOS
curl -sSL https://github.com/shibuiwilliam/archfit/releases/latest/download/archfit-<version>-linux-amd64.tar.gz \
  | tar xz
./archfit version
```

5プラットフォーム向けのビルド済みバイナリとSHA-256チェックサムが各
[GitHub Release](https://github.com/shibuiwilliam/archfit/releases) に公開されます。

### Docker

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

マルチアーキテクチャイメージ（`linux/amd64` + `linux/arm64`）が
[GitHub Container Registry](https://github.com/shibuiwilliam/archfit/pkgs/container/archfit)
に公開されます。

---

## リポジトリ構成

```
archfit/
├── cmd/archfit/              # CLIエントリーポイント — 明示的な配線、17サブコマンド
├── internal/
│   ├── core/                 # スケジューラー: collectors → FactStore → rules → scores
│   ├── model/                # Rule, Finding, Metric, FactStore, ParseFailure
│   ├── config/               # .archfit.yaml の読み込み + 検証
│   ├── contract/             # 適性コントラクト (.archfit-contract.yaml)
│   ├── policy/               # 組織ポリシー (--policy)
│   ├── collector/            # ファクト収集: fs, git, schema, depgraph, command
│   ├── adapter/
│   │   ├── exec/             # フェイク可能なサブプロセスランナー
│   │   └── llm/              # Claude, OpenAI, Gemini（Clientインターフェース経由）
│   ├── fix/                  # Fixエンジン + 7スタティックフィクサー + LLMフィクサー
│   │   ├── static/           # 決定的フィクサー（埋め込みテンプレート付き）
│   │   └── llmfix/           # LLMアシストフィクサー（--with-llm でオプトイン）
│   ├── packman/              # パック検証 (validate-pack コマンド)
│   ├── rule/                 # ルールエンジンコア
│   ├── report/               # レンダラー: terminal, json, md, sarif
│   └── score/                # 重み付け正規化スコアリング + 6メトリクス
├── packs/
│   ├── core/                 # 7ルール（P1, P3, P4, P5, P6, P7）
│   │   ├── resolvers/        # FactStoreの純粋関数
│   │   ├── fixtures/         # ルールごとのゴールデンリポジトリ + expected.json
│   │   └── pack_test.go      # フィクスチャ駆動テーブルテスト
│   └── agent-tool/           # 3ルール（P2, P7） — オプトイン
├── schemas/                  # バージョン付きJSON Schema: rule, config, output, contract
├── testdata/e2e/             # エンドツーエンドゴールデンテスト
├── .claude/skills/archfit/   # Claude Codeエージェントスキル（自動検出）
│   └── reference/remediation/  # ルールごとの修正ガイド10件
├── .github/workflows/
│   ├── ci.yml                # lint + test + self-scan + cross-build (5プラットフォーム)
│   └── release.yml           # バイナリ + GitHub Release + Docker (ghcr.io)
├── docs/
│   ├── adr/                  # アーキテクチャ決定記録
│   ├── rules/                # ルールごとのドキュメント
│   ├── deployment.md         # デプロイ/ロールバック手順
│   ├── llm.md                # --with-llm の契約
│   └── exit-codes.md         # 終了コードの契約
├── Dockerfile                # マルチステージ: golang:1.24-alpine → scratch
├── .archfit.yaml             # archfit自身の設定（self-scan用）
├── Makefile
├── CLAUDE.md                 # コントリビューター契約
├── CHANGELOG.md              # Keep-a-Changelog 1.1.0 形式
├── CONTRIBUTING.md           # コントリビューションガイド
├── SECURITY.md               # セキュリティポリシー
└── LICENSE                   # Apache 2.0
```

**境界ルール**: `packs/*` は `internal/model` と `internal/rule` をインポートできますが、
I/Oを行うものは**インポートできません**。ルールが新しいファクトを必要とする場合は
Collectorを追加します。[`.go-arch-lint.yaml`](./.go-arch-lint.yaml) で強制。

---

## 開発

```bash
make build            # ./bin/archfit にビルド
make test             # ユニット + パックテスト（-race付き）
make test-short       # 短縮テスト（長いテストをスキップ）
make e2e              # エンドツーエンドゴールデンテスト
make lint             # gofmt + go vet + golangci-lint + go-arch-lint
make self-scan        # archfitを自身に対して実行 — exit 0 必須
make self-scan-json   # 同上、JSONをstdoutに出力
make update-golden    # expected.json を再生成（差分を慎重にレビュー！）
make clean
```

テストはネットワークI/Oを一切行いません。LLMの `Fake` クライアントがテストスイート全体で
使用されます。APIキー不要。

**self-scan** が推進力です：`archfit scan ./` がarchfit自身のコードにフラグを
立てた場合、その変更は間違っています。

---

## コントリビューション

PRを作成する前に [`CLAUDE.md`](./CLAUDE.md) と
[`CONTRIBUTING.md`](./CONTRIBUTING.md) をお読みください。主なルール：

- PR予算：変更行数500行以下、影響パッケージ5つ以下
- 新しいルールごとに：リゾルバー、フィクスチャ + `expected.json`、テーブルテスト、ルールドキュメント、修正ガイド
- `init()` 登録なし、リフレクションなし、グローバル可変状態なし
- `packs/*` 内でのI/O禁止 — Collectorを追加すること
- デフォルトスキャンパスでのLLM呼び出し禁止

---

## セキュリティ

詳細は [`SECURITY.md`](./SECURITY.md)。2点の留意事項：

- archfit はスキャン対象リポジトリに対して `git log` を実行します。信頼できないリポジトリにはサンドボックスを使用してください。
- `--with-llm` はルールメタデータと検出エビデンスをLLMプロバイダーに送信します。**ソースコードとファイル内容は送信されません。** 詳細な契約: [`docs/llm.md`](./docs/llm.md)。

---

## archfit が*ではないもの*

- 言語固有リンター（`golangci-lint`、`eslint`、`ruff`）の代替ではありません。
- SASTツールではありません。Semgrep、CodeQL、Trivyを使ってください。
- ベンチマークツールではありません。スコアは*あなたの*リポジトリの時系列信号です。
- 檻ではありません。抑制は — 理由と有効期限付きで — 意図的に存在します。
- LLMに依存していません。基本スキャンは決定論的かつオフラインです。

---

## ロードマップ

詳細な計画は [`DEVELOPMENT_PLAN.md`](./DEVELOPMENT_PLAN.md)：

- **0.1.0**: 基盤、`core` パック（4ルール）、JSON/Markdown、self-scan。
- **0.2.0**: `init`/`check`/`report`/`diff`、SARIF 2.1.0、`agent-tool` パック、e2eテスト、CI。
- **0.3.x**: マルチプロバイダーLLM（Claude, OpenAI, Gemini）。P3/P5/P6ルール。`--config` フラグ。`archfit fix`（7スタティックフィクサー）。Dockerfile + リリースワークフロー。
- **次のフェーズ**: `web-saas`/`iac`/`mobile`/`data-event` パック、メトリクスパイプライン、追加コレクター（AST、depgraph、command）。
- **1.0**: ルールID凍結、JSONスキーマv1、SARIF認定。

---

## ライセンス

Apache 2.0 — [`LICENSE`](./LICENSE) をご覧ください。
