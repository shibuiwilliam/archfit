![archfit logo](./archfit_logo.png)

# archfit

> **コーディングエージェント時代のアーキテクチャ適性評価ツール**
> あなたのリポジトリは、コーディングエージェントが*安全*かつ*迅速*に作業できる形になっていますか？

![CI](https://github.com/shibuiwilliam/archfit/actions/workflows/ci.yml/badge.svg)
![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)

[ドキュメント](https://shibuiwilliam.github.io/archfit/) | [English](./README.md)

---

多くのツールは*コード*をチェックします。archfit はコードが置かれている**地形**をチェックします。
エージェントが最初に読むエントリーポイント、フィードバックループの速さ、
ひとつの不適切な変更がすべてを静かに壊しうる箇所を評価します。

7つのアーキテクチャ特性を評価し、コーディングエージェント（と新しい人間の開発者）が
シニアエンジニアの全diff確認なしに成功できるかを判定します：

| | 原則 | 問いかけ |
|---|---|---|
| **P1** | 局所性 (Locality) | 変更はリポジトリの狭い範囲だけで理解できるか？ |
| **P2** | 仕様優先 (Spec-first) | 契約はスキーマや型か、散文ではないか？ |
| **P3** | 浅い明示性 (Shallow explicitness) | リフレクションや深い間接参照を追わずに振る舞いが見えるか？ |
| **P4** | 検証可能性 (Verifiability) | 正しさを数秒でローカルに証明できるか？ |
| **P5** | 危険の集約 (Aggregation of danger) | 認証・秘密情報・マイグレーションは集中して保護されているか？ |
| **P6** | 可逆性 (Reversibility) | あらゆる変更を低コストでロールバックできるか？ |
| **P7** | 機械可読性 (Machine-readability) | エラー・ADR・CLI は機械にも読めるか？ |

archfit はリンターでも SAST スキャナーでも**ありません**。
それらの*上位*に位置し、それらが測定しないアーキテクチャ特性をレポートします。

---

## クイックスタート

```bash
# インストール（Go 1.24+）
go install github.com/shibuiwilliam/archfit/cmd/archfit@latest

# またはソースからビルド
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit && make build

# 設定ファイルを生成（スタック自動検出）
archfit init /path/to/your/repo

# スキャン実行
archfit scan /path/to/your/repo
```

Docker:

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

### 出力例

```
archfit 1.0.0 — target . (profile=standard)
rules evaluated: 27 (0 with findings), findings: 0
overall score: 100.0
  P1: 100.0  P2: 100.0  P3: 100.0  P4: 100.0
  P5: 100.0  P6: 100.0  P7: 100.0
no findings
```

改善点が見つかった場合：

```
archfit 1.0.0 — target . (profile=standard)
rules evaluated: 27 (2 with findings), findings: 2
overall score: 84.0
findings:
  [warn] P3.EXP.001  — repository uses .env files but has no .env.example
  [warn] P6.REV.001 docs/ — deployment artifacts detected but no rollback documentation
```

各検出結果にはエビデンス、確信度、修正ガイドが付属します。
自動修正：

```bash
archfit fix P3.EXP.001 .       # 特定の検出結果を修正
archfit fix --all .             # 全て修正
archfit fix --dry-run --all .   # プレビュー
```

---

## ルールセット — 27ルール、全7原則をカバー

### `core` パック（24ルール） — 全リポジトリ対象

| ID | 原則 | 検査内容 | 重大度 |
|---|---|---|---|
| [P1.LOC.001](./docs/rules/P1.LOC.001.md) | 局所性 | ルートに `CLAUDE.md` / `AGENTS.md` | warn |
| [P1.LOC.002](./docs/rules/P1.LOC.002.md) | 局所性 | スライスディレクトリに `AGENTS.md` | warn |
| [P1.LOC.003](./docs/rules/P1.LOC.003.md) | 局所性 | 依存結合度が制限内（最大到達数 ≤10） | info |
| [P1.LOC.004](./docs/rules/P1.LOC.004.md) | 局所性 | コミットの変更ファイル数が制限内（≤8） | info |
| [P1.LOC.005](./docs/rules/P1.LOC.005.md) | 局所性 | 高リスクパスに `INTENT.md` を宣言 | warn |
| [P1.LOC.006](./docs/rules/P1.LOC.006.md) | 局所性 | エージェントドキュメントの肥大化防止（≤400行、≤10 KB） | warn |
| [P1.LOC.009](./docs/rules/P1.LOC.009.md) | 局所性 | 高リスクスライスにランブック | warn |
| [P2.SPC.001](./docs/rules/P2.SPC.001.md) | 仕様優先 | API境界に機械可読な契約がある | warn |
| [P2.SPC.002](./docs/rules/P2.SPC.002.md) | 仕様優先 | DBマイグレーションの双方向性 | warn |
| [P2.SPC.004](./docs/rules/P2.SPC.004.md) | 仕様優先 | ADRにYAMLフロントマター | info |
| [P3.EXP.001](./docs/rules/P3.EXP.001.md) | 明示性 | 設定の文書化（.env、Spring Boot、Terraform、Rails） | warn |
| [P3.EXP.002](./docs/rules/P3.EXP.002.md) | 明示性 | `init()` によるクロスパッケージ登録なし（Go、AST） | warn |
| [P3.EXP.003](./docs/rules/P3.EXP.003.md) | 明示性 | リフレクション密度の制限（Go、AST） | info |
| [P3.EXP.005](./docs/rules/P3.EXP.005.md) | 明示性 | グローバル可変状態の最小化（Go、AST） | info |
| [P4.VER.001](./docs/rules/P4.VER.001.md) | 検証可能性 | 検証エントリーポイント（26種以上のビルドツール） | warn |
| [P4.VER.002](./docs/rules/P4.VER.002.md) | 検証可能性 | ソースの70%以上にテストファイル | info |
| [P4.VER.003](./docs/rules/P4.VER.003.md) | 検証可能性 | CI設定がある | info |
| [P5.AGG.001](./docs/rules/P5.AGG.001.md) | 危険の集約 | セキュリティファイルが集中 | warn |
| [P5.AGG.002](./docs/rules/P5.AGG.002.md) | 危険の集約 | CIでシークレットスキャナーが動作 | warn |
| [P5.AGG.003](./docs/rules/P5.AGG.003.md) | 危険の集約 | リスク階層ファイルの宣言 | warn |
| [P5.AGG.004](./docs/rules/P5.AGG.004.md) | 危険の集約 | 高リスクパスをCODEOWNERSで保護 | **error** |
| [P6.REV.001](./docs/rules/P6.REV.001.md) | 可逆性 | デプロイ → ロールバックドキュメント | warn |
| [P6.REV.002](./docs/rules/P6.REV.002.md) | 可逆性 | デプロイリポジトリにフィーチャーフラグ | info |
| [P7.MRD.001](./docs/rules/P7.MRD.001.md) | 機械可読性 | CLIが終了コードを文書化 | warn |

### `agent-tool` パック（3ルール） — オプトイン、エージェント利用ツール向け

| ID | 原則 | 検査内容 |
|---|---|---|
| [P2.SPC.010](./docs/rules/P2.SPC.010.md) | 仕様優先 | バージョン管理スキーマ（OpenAPI、Protobuf、GraphQL、Avro 対応） |
| [P7.MRD.002](./docs/rules/P7.MRD.002.md) | 機械可読性 | ルートに `CHANGELOG.md` |
| [P7.MRD.003](./docs/rules/P7.MRD.003.md) | 機械可読性 | CLIが `docs/adr/` にADR |

ルール定義はYAMLが信頼源。リポジトリの検出言語に該当しないルールは自動スキップ。

---

## 言語・スタック対応

archfit は言語非依存設計。スタックに応じて検出を適応：

**P4.VER.001**: Go、Node/TS、Python、Rust、Java（Maven + Gradle）、Ruby、PHP、Elixir、Scala、C/C++（CMake、Meson）、Deno、Bazel、Earthly、汎用タスクランナー。

**P3.EXP.001**: `.env`、Spring Boot `application-*.yml`、Terraform `*.tfvars`、Rails `config/environments/`。エコシステムコレクタにより、Spring/Railsは実際に検出された場合のみチェック。

**P3.EXP.002 / P3.EXP.003 / P3.EXP.005**: Go リポジトリ向け AST コレクタによる静的解析。`init()` 登録、リフレクション密度、グローバル可変状態を検出。

**P1.LOC.002**: `packs/`、`services/`、`modules/`、`packages/`、`apps/`、`libs/`、`plugins/`、`engines/`、`components/`、`domains/`、`features/`。

**P2.SPC.010**: JSON Schema、OpenAPI/Swagger、Protobuf、GraphQL、Avro、AsyncAPI。

---

## コマンド

```bash
# スキャン
archfit scan [path]                  # 全ルール実行（デフォルト: .）
archfit check <rule-id> [path]       # 単一ルール
archfit score [path]                 # スコアのみ
archfit report [path]                # Markdownレポート

# 修正
archfit fix [rule-id] [path]         # 自動修正
archfit fix --all .                  # 修正可能な全てを修正

# コントラクト
archfit contract check [path]        # .archfit-contract.yaml との照合
archfit contract init [path]         # 現在のスキャンからコントラクト生成

# 比較
archfit diff <baseline.json>         # PR回帰ゲート
archfit trend                        # スコア履歴
archfit compare <f1> <f2> [...]      # クロスリポジトリ比較

# PR ゲート
archfit pr-check                     # CI向けPRチェック（回帰検出）

# セットアップ
archfit init [path]                  # 設定生成（スタック自動検出）
archfit explain <rule-id>            # ルール詳細 + 修正ガイド
archfit list-rules                   # 全ルール一覧
```

### 主要フラグ

| フラグ | デフォルト | 説明 |
|---|---|---|
| `--format {terminal\|json\|md\|sarif}` | `terminal` | 出力形式 |
| `--json` | | `--format=json` の省略形 |
| `--fail-on {info\|warn\|error\|critical}` | `error` | この重大度以上で exit 1 |
| `--depth {shallow\|standard\|deep}` | `standard` | スキャン深度 |
| `--with-llm` | off | LLM（Claude/OpenAI/Gemini）で補強 |
| `--record <dir>` | | JSON + Markdownを保存 |
| `--explain-coverage` | | ルール適用状況を表示 |
| `-C <dir>` | | 実行前にディレクトリを移動 |

### 終了コード

| コード | 意味 |
|:---:|---|
| 0 | 成功（検出結果が閾値以下） |
| 1 | 閾値以上の検出結果 / コントラクト違反 |
| 2 | 使用方法エラー |
| 3 | ランタイムエラー |
| 4 | 設定エラー |
| 5 | ソフトターゲット未達（ハード違反なし） |

---

## 自動修正

```bash
archfit fix P1.LOC.001 .             # CLAUDE.md を生成
archfit fix --all .                  # 修正可能な全てを修正
archfit fix --plan --all .           # 適用前にプラン確認
```

全修正は自動再スキャンで検証。検出結果が残るか新たに発生した場合はロールバック。ログは `.archfit-fix-log.json`。

---

## 適性コントラクト

`.archfit-contract.yaml` で機械的に強制可能な適性目標を宣言：

```bash
archfit contract init .      # 現在のスキャンからコントラクト生成
archfit contract check .     # CIで強制（exit 0/1/5）
```

ハード制約、ソフトターゲット、エリアバジェット（SREスタイル）、エージェント指示をサポート。

---

## CI連携

### SARIF で GitHub Code Scanning

```yaml
- run: archfit scan --format=sarif . > archfit.sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: archfit.sarif
```

### PR ゲート

```yaml
- run: archfit diff baseline.json   # 新しい検出結果があれば exit 1
```

---

## LLM補強（オプトイン）

```bash
export ANTHROPIC_API_KEY=sk-...   # または OPENAI_API_KEY / GOOGLE_API_KEY
archfit scan --with-llm .
```

| プロバイダー | デフォルトモデル | `--llm-backend` |
|---|---|---|
| Claude (Anthropic) | `claude-sonnet-4-20250514` | `claude` |
| OpenAI | `gpt-5.4-mini` | `openai` |
| Google Gemini | `gemini-2.5-flash` | `gemini` |

安全性：オプトインのみ、予算上限あり、キャッシュ対応、スキャンを失敗させない。ルールメタデータとエビデンスのみ送信 — **ソースコードは送信されません**。

---

## Claude Codeスキル

archfit は [`.claude/skills/archfit/`](./.claude/skills/archfit/) にスキャン→修正→検証スキルを同梱。
ルールごとの修正判断ツリーとヘルパースクリプトを備えています。
別のリポジトリで使う場合は `.claude/skills/archfit/` をそのプロジェクトの `.claude/skills/` にコピーしてください。

---

## インストール

| 方法 | コマンド |
|---|---|
| `go install` | `go install github.com/shibuiwilliam/archfit/cmd/archfit@latest` |
| ソース | `git clone ... && make build` |
| Docker | `docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo` |
| バイナリ | [Releases](https://github.com/shibuiwilliam/archfit/releases) |

詳細は[インストールガイド](./docs/installation.md)。

---

## 開発

```bash
make build          # ビルド（./bin/archfit）
make test           # ユニット + パックテスト（-race付き）
make lint           # gofmt + go vet + golangci-lint
make self-scan      # 自己スキャン — exit 0 必須
make generate       # YAMLからルール定義を再生成
```

テストはネットワークI/Oを行いません。自己スキャンが強制関数です：`archfit scan ./` が自身のコードにフラグを立てたら、その変更は間違いです。

---

## archfit で*ない*もの

- 言語固有のリンターの代替ではありません。
- SASTツールではありません。
- ベンチマークではありません。スコアは*あなたの*リポジトリの経時変化のためのものです。
- LLMに依存しません。ベーススキャンは決定的でオフラインです。

---

## ライセンス

Apache 2.0 — [LICENSE](./LICENSE)。
