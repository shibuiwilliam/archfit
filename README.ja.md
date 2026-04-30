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
rules evaluated: 17 (0 with findings), findings: 0
overall score: 100.0
  P1: 100.0  P2: 100.0  P3: 100.0  P4: 100.0
  P5: 100.0  P6: 100.0  P7: 100.0
no findings
```

改善点が見つかった場合：

```
archfit 1.0.0 — target . (profile=standard)
rules evaluated: 17 (2 with findings), findings: 2
overall score: 84.0
findings:
  [warn] P3.EXP.001  — repository uses .env files but has no .env.example
  [warn] P6.REV.001 docs/ — deployment artifacts detected but no rollback documentation
```

自動修正：

```bash
archfit fix P3.EXP.001 .       # 特定の検出結果を修正
archfit fix --all .             # 全て修正
archfit fix --dry-run --all .   # プレビュー
```

---

## ルールセット — 17ルール、全7原則をカバー

### `core` パック（14ルール）

| ID | 原則 | 検査内容 | 重大度 |
|---|---|---|---|
| [P1.LOC.001](./docs/rules/P1.LOC.001.md) | 局所性 | ルートに `CLAUDE.md` / `AGENTS.md` | warn |
| [P1.LOC.002](./docs/rules/P1.LOC.002.md) | 局所性 | スライスディレクトリに `AGENTS.md` | warn |
| [P1.LOC.003](./docs/rules/P1.LOC.003.md) | 局所性 | 依存結合度が制限内 | info |
| [P1.LOC.004](./docs/rules/P1.LOC.004.md) | 局所性 | コミットの変更ファイル数が制限内 | info |
| [P2.SPC.001](./docs/rules/P2.SPC.001.md) | 仕様優先 | API境界に機械可読な契約がある | warn |
| [P3.EXP.001](./docs/rules/P3.EXP.001.md) | 明示性 | 設定の文書化（.env、Spring Boot、Terraform、Rails） | warn |
| [P4.VER.001](./docs/rules/P4.VER.001.md) | 検証可能性 | 検証エントリーポイント（26種以上のビルドツール） | warn |
| [P4.VER.002](./docs/rules/P4.VER.002.md) | 検証可能性 | ソースの70%以上にテストファイル | info |
| [P4.VER.003](./docs/rules/P4.VER.003.md) | 検証可能性 | CI設定がある | info |
| [P5.AGG.001](./docs/rules/P5.AGG.001.md) | 危険の集約 | セキュリティファイルが集中 | warn |
| [P5.AGG.002](./docs/rules/P5.AGG.002.md) | 危険の集約 | CIでシークレットスキャナーが動作 | warn |
| [P6.REV.001](./docs/rules/P6.REV.001.md) | 可逆性 | デプロイ → ロールバックドキュメント | warn |
| [P6.REV.002](./docs/rules/P6.REV.002.md) | 可逆性 | デプロイリポジトリにフィーチャーフラグ | info |
| [P7.MRD.001](./docs/rules/P7.MRD.001.md) | 機械可読性 | CLIが終了コードを文書化 | warn |

### `agent-tool` パック（3ルール） — オプトイン

| ID | 原則 | 検査内容 |
|---|---|---|
| [P2.SPC.010](./docs/rules/P2.SPC.010.md) | 仕様優先 | バージョン管理スキーマ（OpenAPI、Protobuf、GraphQL 対応） |
| [P7.MRD.002](./docs/rules/P7.MRD.002.md) | 機械可読性 | ルートに `CHANGELOG.md` |
| [P7.MRD.003](./docs/rules/P7.MRD.003.md) | 機械可読性 | CLIが `docs/adr/` にADR |

ルール定義はYAMLが信頼源。リポジトリの検出言語に該当しないルールは自動スキップ。

---

## 言語・スタック対応

**P4.VER.001**: Go、Node/TS、Python、Rust、Java（Maven + Gradle）、Ruby、PHP、Elixir、Scala、C/C++、Deno、Bazel、Earthly。

**P3.EXP.001**: `.env`、Spring Boot `application-*.yml`、Terraform `*.tfvars`、Rails `config/environments/`。エコシステムコレクタにより、Spring/Railsは実際に検出された場合のみチェック。

**P2.SPC.010**: JSON Schema、OpenAPI/Swagger、Protobuf、GraphQL、Avro、AsyncAPI。

---

## コマンド

```bash
archfit scan [path]                  # 全ルール実行
archfit fix --all .                  # 全て修正
archfit contract check [path]        # コントラクト照合
archfit contract init [path]         # コントラクト生成
archfit diff <baseline.json>         # PR回帰ゲート
archfit init [path]                  # 設定生成（スタック自動検出）
archfit explain <rule-id>            # ルール詳細
archfit list-rules                   # 全ルール一覧
```

### 主要フラグ

| フラグ | デフォルト | 説明 |
|---|---|---|
| `--format {terminal\|json\|md\|sarif}` | `terminal` | 出力形式 |
| `--fail-on {info\|warn\|error\|critical}` | `error` | この重大度以上で exit 1 |
| `--with-llm` | off | LLM（Claude/OpenAI/Gemini）で補強 |
| `--record <dir>` | | JSON + Markdownを保存 |
| `--explain-coverage` | | ルール適用状況を表示 |

### 終了コード

| コード | 意味 |
|:---:|---|
| 0 | 成功 |
| 1 | 閾値以上の検出結果 / コントラクト違反 |
| 2 | 使用方法エラー |
| 3 | ランタイムエラー |
| 4 | 設定エラー |
| 5 | ソフトターゲット未達 |

---

## 自動修正

```bash
archfit fix --all .                  # 修正可能な全てを修正
archfit fix --plan --all .           # プラン確認
```

全修正は自動再スキャンで検証。ログは `.archfit-fix-log.json`。

---

## 適性コントラクト

```bash
archfit contract init .      # コントラクト生成
archfit contract check .     # CIで強制（exit 0/1/5）
```

ハード制約、ソフトターゲット、エリアバジェット、エージェント指示をサポート。

---

## CI連携

```yaml
- run: archfit scan --format=sarif . > archfit.sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: archfit.sarif
```

---

## LLM補強（オプトイン）

```bash
export ANTHROPIC_API_KEY=sk-...
archfit scan --with-llm .
```

| プロバイダー | デフォルトモデル |
|---|---|
| Claude | `claude-sonnet-4-20250514` |
| OpenAI | `gpt-5.4-mini` |
| Gemini | `gemini-2.5-flash` |

ソースコードは送信されません。

---

## Claude Codeスキル

[`.claude/skills/archfit/`](./.claude/skills/archfit/) にスキャン→修正→検証スキル同梱。

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
make build          # ビルド
make test           # テスト
make lint           # リント
make self-scan      # 自己スキャン — exit 0 必須
make generate       # YAMLからルール再生成
```

---

## ライセンス

Apache 2.0 — [LICENSE](./LICENSE)。
