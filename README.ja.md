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

Docker でも実行可能：

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

### 出力例

```
archfit 0.1.0 — target . (profile=standard)
rules evaluated: 14 (0 with findings), findings: 0
overall score: 100.0
  P1: 100.0  P2: 100.0  P3: 100.0  P4: 100.0
  P5: 100.0  P6: 100.0  P7: 100.0
no findings
```

改善点が見つかった場合：

```
archfit 0.1.0 — target . (profile=standard)
rules evaluated: 14 (2 with findings), findings: 2
overall score: 84.0
findings:
  [warn] P3.EXP.001  — repository uses .env files but has no .env.example
  [warn] P6.REV.001 docs/ — deployment artifacts detected but no rollback documentation
```

すべての検出結果にはエビデンス、信頼度、修正ガイドが付属します。
多くの検出結果は自動修正可能です：

```bash
archfit fix P3.EXP.001 .       # 特定の検出結果を修正
archfit fix --all .             # 修正可能な全てを一括修正
archfit fix --dry-run --all .   # 変更内容をプレビュー
```

---

## ルールセット — 14ルール、全7原則をカバー

### `core` パック（11ルール） — すべてのリポジトリに適用

| ID | 原則 | 検査内容 | 重大度 |
|---|---|---|---|
| [P1.LOC.001](./docs/rules/P1.LOC.001.md) | 局所性 | リポジトリルートに `CLAUDE.md` または `AGENTS.md` | warn |
| [P1.LOC.002](./docs/rules/P1.LOC.002.md) | 局所性 | 垂直スライスディレクトリが `AGENTS.md` を持っている | warn |
| [P1.LOC.003](./docs/rules/P1.LOC.003.md) | 局所性 | 依存結合度が制限内（最大到達数 ≤10） | info |
| [P1.LOC.004](./docs/rules/P1.LOC.004.md) | 局所性 | コミットの変更ファイル数が制限内（≤8） | info |
| [P3.EXP.001](./docs/rules/P3.EXP.001.md) | 明示性 | 設定の文書化（.env、Spring Boot、Terraform、Rails） | warn |
| [P4.VER.001](./docs/rules/P4.VER.001.md) | 検証可能性 | 検証エントリーポイント（Makefile、pom.xml 等 — [26種対応](#言語スタック対応)） | warn |
| [P4.VER.002](./docs/rules/P4.VER.002.md) | 検証可能性 | ソースディレクトリの70%以上にテストファイル | info |
| [P4.VER.003](./docs/rules/P4.VER.003.md) | 検証可能性 | CI設定がある（GitHub Actions、GitLab CI 等） | info |
| [P5.AGG.001](./docs/rules/P5.AGG.001.md) | 危険の集約 | セキュリティ関連ファイルが集中している | warn |
| [P6.REV.001](./docs/rules/P6.REV.001.md) | 可逆性 | デプロイ成果物 → ロールバックドキュメント | warn |
| [P7.MRD.001](./docs/rules/P7.MRD.001.md) | 機械可読性 | CLIリポジトリが終了コードを文書化 | warn |

### `agent-tool` パック（3ルール） — オプトイン

| ID | 原則 | 検査内容 |
|---|---|---|
| [P2.SPC.010](./docs/rules/P2.SPC.010.md) | 仕様優先 | `$id` 付きバージョン管理スキーマ（OpenAPI、Protobuf、GraphQL も認識） |
| [P7.MRD.002](./docs/rules/P7.MRD.002.md) | 機械可読性 | ルートに `CHANGELOG.md` |
| [P7.MRD.003](./docs/rules/P7.MRD.003.md) | 機械可読性 | CLIが `docs/adr/` にADRを記録 |

ルール定義は `packs/*/rules/` 配下のYAMLで管理され、仕様優先の信頼源です。

---

## 言語・スタック対応

archfit は設計上、言語非依存です。

**P4.VER.001 — 検証エントリーポイント**: Go、Node/TS、Python、Rust、Java（Maven + Gradle）、Ruby、PHP、Elixir、Scala、C/C++（CMake, Meson）、Deno、Bazel、Earthly、汎用タスクランナー。

**P3.EXP.001 — 設定の文書化**: `.env` ファイル、Spring Boot `application-*.yml`、Terraform `*.tfvars`、Rails `config/environments/`。

**P1.LOC.002 — スライスコンテナ**: `packs/`、`services/`、`modules/`、`packages/`、`apps/`、`libs/`、`plugins/`、`engines/`、`components/`、`domains/`、`features/`。

**P2.SPC.010 — 仕様フォーマット**: JSON Schema、OpenAPI/Swagger、Protobuf、GraphQL、Avro、AsyncAPI。

---

## コマンド

```bash
# スキャン
archfit scan [path]                  # 全ルール実行
archfit check <rule-id> [path]       # 単一ルール
archfit score [path]                 # スコアのみ

# 修正
archfit fix [rule-id] [path]         # 自動修正
archfit fix --all .                  # 全て修正

# コントラクト
archfit contract check [path]        # .archfit-contract.yaml と照合
archfit contract init [path]         # 現在のスキャンからコントラクト生成

# 比較
archfit diff <baseline.json>         # PR回帰ゲート
archfit trend                        # スコア推移

# セットアップ
archfit init [path]                  # .archfit.yaml 生成（スタック自動検出）
archfit explain <rule-id>            # ルール詳細
archfit list-rules                   # 全ルール一覧
```

### 主要フラグ

| フラグ | デフォルト | 説明 |
|---|---|---|
| `--format {terminal\|json\|md\|sarif}` | `terminal` | 出力形式 |
| `--fail-on {info\|warn\|error\|critical}` | `error` | この重大度以上で exit 1 |
| `--with-llm` | off | LLM（Claude/OpenAI/Gemini）で説明を補強 |
| `--record <dir>` | | JSON + Markdownをタイムスタンプ付きで保存 |
| `--explain-coverage` | | ルールの適用状況を表示 |

### 終了コード

| コード | 意味 |
|:---:|---|
| 0 | 成功 |
| 1 | 閾値以上の検出結果あり / コントラクトのハード制約違反 |
| 2 | 使用方法エラー |
| 3 | ランタイムエラー |
| 4 | 設定エラー |
| 5 | コントラクトのソフトターゲット未達（ハード制約は満たしている） |

---

## 自動修正

```bash
archfit fix P1.LOC.001 .             # CLAUDE.md を作成
archfit fix --all .                  # 修正可能な全てを一括修正
archfit fix --plan --all .           # 適用せずにプランを確認
```

全ての修正は自動再スキャンにより検証されます。修正アクションは `.archfit-fix-log.json` に記録。

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

### GitHub Code Scanning

```yaml
- run: archfit scan --format=sarif . > archfit.sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: archfit.sarif
```

### PRゲート

```yaml
- run: archfit diff baseline.json   # 新しい検出結果でexit 1
```

---

## LLM補強（オプトイン）

```bash
export ANTHROPIC_API_KEY=sk-...
archfit scan --with-llm .
```

| プロバイダー | デフォルトモデル |
|---|---|
| Claude (Anthropic) | `claude-sonnet-4-20250514` |
| OpenAI | `gpt-5.4-mini` |
| Google Gemini | `gemini-2.5-flash` |

安全性: オプトインのみ、バジェット上限あり、キャッシュ付き、スキャン失敗なし。ソースコードは送信されません。

---

## Claude Codeエージェントスキル

[`.claude/skills/archfit/`](./.claude/skills/archfit/) にスキャン→修正→検証ループを駆動するスキルを同梱。
別のリポジトリで使用するには `.claude/skills/archfit/` をコピーしてください。

---

## インストール

| 方法 | コマンド |
|---|---|
| `go install` | `go install github.com/shibuiwilliam/archfit/cmd/archfit@latest` |
| ソース | `git clone ... && make build` |
| Docker | `docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo` |
| バイナリ | [Releases](https://github.com/shibuiwilliam/archfit/releases) からダウンロード（5プラットフォーム） |

詳細は[インストールガイド](./docs/installation.md)。

---

## 開発

```bash
make build          # ./bin/archfit にビルド
make test           # ユニット + パックテスト
make lint           # gofmt + go vet + golangci-lint
make self-scan      # archfitを自身に実行 — exit 0 必須
make generate       # YAMLからルール定義を再生成
```

テストはネットワークI/O一切なし。self-scanが推進力: archfit自身のコードにフラグが立った場合、その変更は間違い。

---

## archfit が*ではないもの*

- 言語固有リンターの代替ではありません。
- SASTツールではありません。
- ベンチマークではありません。スコアは*あなたの*リポジトリの時系列信号です。
- LLMに依存していません。基本スキャンは決定論的かつオフラインです。

---

## ライセンス

Apache 2.0 — [LICENSE](./LICENSE)。
