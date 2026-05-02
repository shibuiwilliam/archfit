リポジトリを詳しく調査します。リポジトリの内容を網羅的に確認しました。`README`、`PROJECT.md`、`CLAUDE.md`、`DEVELOPMENT_PLAN.md`、`IDEA.md`、`.archfit.yaml`、ルールドキュメント例(`docs/rules/P1.LOC.001.md`)を読み込み、現状の到達点とギャップを把握しました。以下、深く網羅的かつ論理的な評価と改善提案を提示します。

---

# archfit 改善提案 — 評価ツールとしての品質と網羅性を引き上げるために

## 0. 総合評価:現状のポジション

archfit は v0.3.x で、**設計思想・メタ一貫性・運用ガードレール**は同種ツールの中でも極めて高い水準にあります。Phase 0 の整合性ギャップ(YAML/Go ドリフト、JSON スキーマ違反、YAML パーサ、`FilesChanged`、ドキュメント整合、`internal/adapter/fs`、self-scan in CI)が**全て閉じた**と PROJECT.md にあるのは率直に立派です。CLI コマンド面・出力フォーマット・LLM アダプタ・fix エンジン・契約検査も既に揃っています。

一方で、**評価ツールとしての品質**という観点では、次の本質的な課題があります:

| 観点 | 現状 | 問題 |
|------|------|------|
| ルール網羅性 | 17 ルール / 7 原則 | 1原則あたり ~2.4 ルール。**P3 は 1 ルールのみ**。原典(IDEA.md / 元レポート §3-9)で挙げた重要概念の多くが未カバー |
| 検出深度 | ほぼファイル存在 + git ログサンプル | AST 解析がなく、**「中身が伴っているか」を見られない**。コンプライアンス・シアター(形だけ整える)を簡単に通してしまう |
| 重大度分布 | `warn`/`info` のみ。`error`/`critical` ゼロ | 「無視しても CI は通る」ツールに見え、組織導入時に弱い |
| Evidence の幅 | `strong`(file presence)中心 | 行動・履歴・実測ベースの evidence が不足、シグナルが薄い |
| キャリブレーション | コーパスは Phase 1 で計画中 | 各ルールの精度・再現率の実測値がない。閾値が経験則 |
| メトリクス | 定義はあるが実体化が部分的 | `verification_latency_s` などは `--depth=deep` 限定、`context_span_p50` は P1.LOC.004 経由のみ。**評価軸の主役にすべき値が脇役** |
| 言語・スタック | Go / TS / Python に偏重 | Java/Ruby/Rust/Swift/Kotlin/IaC で**応用範囲外の repo はスコアが過大評価される** |
| スコア意味論 | 17 ルールで 0-100 の連続値 | 1 ルールが 5-10 点動かす粒度。**「100点 = エージェント時代に最適化されている」を担保できない** |
| 自己選択バイアス | self-scan = 100、agent-tool 適用 | 設計者と評価対象が同じ。**外部 repo での経験的妥当性が未証明** |

要約すると、archfit は **「整合性のある雛形」としては完成度が高いが、「エージェント時代のアーキテクチャ評価ツールとしての密度・深さ・実証性」が次の課題**です。以降、この 3 軸に沿って具体策を提示します。

---

## 1. ルール網羅性のギャップと埋め方

### 1.1 原則ごとの偏在

現状 17 ルールの principle 別分布:

```
P1 Locality        : 4 (LOC.001-004)
P2 Spec-first      : 2 (SPC.001, SPC.010)
P3 Explicitness    : 1 (EXP.001)        ← 異常に少ない
P4 Verifiability   : 3 (VER.001-003)
P5 Aggregation     : 2 (AGG.001, AGG.002)
P6 Reversibility   : 2 (REV.001, REV.002)
P7 Machine-read    : 3 (MRD.001-003)
```

P3 が 1 つしかなく、しかも `.env.example` の有無という浅い検査です。**P3 = 浅い明示性**は本来「どれだけリフレクション・メタプログラミング・暗黙登録が使われていないか」を見る原則であり、エージェントの推論コストに直結します。ここの空白は致命的です。

### 1.2 元レポートで重要としながら未実装の概念

原典(添付レポート §3-9)で挙げられ、archfit が**意図的に取り組むべきだが未着手**の概念を表で整理します:

| 概念 | 元レポートの位置 | 現状 | 実装可能性 |
|------|------------------|------|------------|
| INTENT.md per context | §1, §3.3 | ❌ | 容易(file presence + 簡易構造チェック) |
| Branded / Nominal Types | §4.2 | ❌ | 中(言語別の AST 必要) |
| "Parse, don't validate" 境界 | §4.2 | ❌ | 中(import + 使用箇所サンプリング) |
| Schema registry for events | §4.3 | ❌ | 容易(設定検出) |
| Outbox + CDC | §4.3 | ❌ | 中 |
| ADR YAML frontmatter 必須化 | §4.4 | △(MRD.003 で有無のみ) | 容易 |
| Risk-tier file (`risk_tiers.yaml`) | §6.1 | ❌ | 容易 |
| 境界の四重一致 (dir ↔ CODEOWNERS ↔ team ↔ IAM) | §3.1 | ❌ | 中 |
| Path-based CODEOWNERS for high-risk | §6.1 | △(AGG.001 のみ) | 容易 |
| Idempotency-Key | §6.5 | ❌ | 中(コードサンプリング) |
| 短命クレデンシャル / OIDC | §6.2 | ❌ | 中(CI / IaC 検出) |
| Property-Based Testing | §5.3 | ❌ | 容易(依存検出) |
| Snapshot / golden fixture | §5.2 | ❌ | 容易 |
| Screenshot diff (UI) | §5.2 | ❌ | 容易(設定検出) |
| Bidirectional migration | §4.1 | ❌ | 中 |
| Expand/contract migration | §6.1 | ❌ | 中(差分解析) |
| Feature flag library | §6.1 | ✅(REV.002) | — |
| Canary / blue-green | §6.1 | ❌ | 中 |
| Runbook per context | §1 | ❌ | 容易 |
| State machine for long-lived flows | §9.1 | ❌ | 中〜難 |
| Two-tier (stable core vs regenerable edge) | §9.2 | ❌ | 中 |
| Boring-tech bias | §9.3 | ❌ | 難(主観) |
| Replay harness for bugs | §4.1 | ❌ | 中 |
| Structured error (`code`/`details`/`remediation`) | §8.3 | ❌ | 中 |
| `.agent-trace/` の運用 | §8.4 | ❌ | 容易 |
| Pit-of-success default(timeout/retry/PII mask) | §6.4 | ❌ | 中 |

**ざっと25項目あり、現状の 17 ルールが 40〜50 ルール規模に拡張可能**であることがわかります。すべてを一気にやるべきではありませんが、**この 25 項目をバックログとして明示**すべきです。

### 1.3 提案する追加ルール 30 件(最初の拡張ターゲット)

以下、Phase 1.5 として実装を提案する追加ルール群を、実装難度・evidence 強度別に列挙します。フォーマットは既存の archfit 規約に合わせています。

#### P1 — Locality(+5 ルール、計 9)

```yaml
- id: P1.LOC.005
  title: "High-risk paths declare INTENT.md"
  detect: file_presence on paths declared in .archfit.yaml risk_tiers.high
  evidence: strong  severity: warn
  rationale: |
    高リスク領域は「なぜ存在するか」「禁止事項」がコードの近くに必要。
    エージェントは tribal knowledge を持たないため、INTENT.md がないと
    grep の結果から推論するしかなく、危険操作を見落とす。

- id: P1.LOC.006
  title: "AGENTS.md / CLAUDE.md not bloated"
  detect: line_count <= 400, byte_count <= 10240 for AGENTS.md / CLAUDE.md
  evidence: strong  severity: warn
  rationale: |
    重要ルールは肥大化したファイルでは埋もれる(原典 §3.3)。
    archfit 自身がこの制約を CLAUDE.md §13 で課しており、
    一般 repo にも同様に適用する。

- id: P1.LOC.007
  title: "Boundary 4-way alignment (dir / CODEOWNERS / team / package)"
  detect: cross-reference top-level dirs ↔ CODEOWNERS patterns ↔ go.mod packages
  evidence: medium  severity: warn
  rationale: |
    境界名が複数のソースで一致しているほどエージェントの探索コストが下がる。
    名前のドリフトは権限管理の脆弱化のサインでもある。

- id: P1.LOC.008
  title: "Cross-cutting infrastructure not duplicated across slices"
  detect: detect duplicate auth/logging/config setup files in multiple slices
  evidence: medium  severity: info
  rationale: |
    DRY と局所性のバランス点。基盤コードは集約、ロジックは局所がエージェントにとって最適。

- id: P1.LOC.009
  title: "runbook.md exists for each high-risk slice"
  detect: file_presence in high-risk paths
  evidence: strong  severity: warn
  rationale: |
    rollback / 運用注意点が変更セルの近くにあるべき。原典 §8.2。
```

#### P2 — Spec-first(+5 ルール、計 7)

```yaml
- id: P2.SPC.002
  title: "Database migrations are bidirectional"
  detect: |
    For each migration file detected (sqlmig, alembic, knex, golang-migrate,
    diesel, sqlx, atlas, etc.), check that an equivalent down migration exists.
  evidence: strong  severity: warn

- id: P2.SPC.003
  title: "Migrations follow expand/contract pattern (no destructive changes alone)"
  detect: |
    AST/regex scan of migration files for DROP COLUMN / DROP TABLE / NOT NULL ADD
    without a paired prep migration in prior commits.
  evidence: medium  severity: warn

- id: P2.SPC.004
  title: "ADR uses YAML frontmatter"
  detect: |
    For each file under docs/adr/, parse frontmatter and require
    status, date, deciders, supersedes (optional), context, decision.
  evidence: strong  severity: info

- id: P2.SPC.005
  title: "Branded / nominal types used for domain identifiers"
  detect: |
    Language-specific:
      - TS: detect `type X = string & { readonly __brand: ... }` patterns
      - Rust: newtype `pub struct UserId(String)` patterns in domain dirs
      - Go: `type UserID string` patterns + linter for cross-type assignment
      - Python: typing.NewType usage
    Threshold: presence in domain layer if domain layer exists.
  evidence: medium  severity: info

- id: P2.SPC.006
  title: "Boundary parsing (Zod/Valibot/pydantic/struct-tag validators) at API edge"
  detect: |
    Detect parsing library imports + usage at HTTP handler entry points.
    Heuristic: handlers without parsing call → flag.
  evidence: weak  severity: info
```

#### P3 — Shallow explicitness(+6 ルール、計 7)— **最重要**

```yaml
- id: P3.EXP.002
  title: "No init() based cross-package registration (Go-specific)"
  detect: |
    AST: count init() functions that mutate package-level state in
    OTHER packages (typically via blank imports `_ "x/y"` for side effect).
    Threshold: >0 → flag with locations.
  evidence: strong  severity: warn  applies_to: { languages: [go] }

- id: P3.EXP.003
  title: "Reflection / metaprogramming density bounded"
  detect: |
    Per-language:
      - Go: count of `reflect.` calls per kloc in source dirs
      - Python: __metaclass__, __init_subclass__, decorator factories
      - Java: Spring AOP, reflective DI
      - TS: decorators with side effects
    Threshold (configurable): >5 per kloc in domain code → flag.
  evidence: medium  severity: info

- id: P3.EXP.004
  title: "Single-implementation interfaces flagged (Go-specific)"
  detect: |
    For each interface in /internal/, count implementations.
    If 1 implementation and not in a test or seam package → candidate violation.
    Allowlist: io.Reader-style standard interfaces, mocking patterns.
  evidence: weak  severity: info  applies_to: { languages: [go] }

- id: P3.EXP.005
  title: "Global mutable state minimized"
  detect: |
    AST scan: package-level `var` (non-const, non-once-init, non-channel)
    in /internal/ or src/.
    Threshold: per package, count > N → flag.
  evidence: medium  severity: info

- id: P3.EXP.006
  title: "Service locator / string-keyed DI avoided"
  detect: |
    Heuristic: detect usage of `container.Get("name")`, `Service Locator` patterns.
  evidence: weak  severity: info

- id: P3.EXP.007
  title: "Indirection depth (interface → factory → impl) bounded"
  detect: |
    Build call graph; for each entry point, measure depth from cmd/main to
    domain logic via interface boundaries. Threshold: median > 6 → flag.
  evidence: medium  severity: info
```

#### P4 — Verifiability(+4 ルール、計 7)

```yaml
- id: P4.VER.004
  title: "Property-based testing present in critical paths"
  detect: |
    Per-language dep detection:
      - Go: gopter, rapid
      - TS: fast-check
      - Python: hypothesis
      - Rust: proptest, quickcheck
      - Java: jqwik
      Plus usage in test files of /internal/score, /internal/contract or domain dirs.
  evidence: strong  severity: info

- id: P4.VER.005
  title: "Snapshot / golden fixture testing used"
  detect: |
    Files matching */fixtures/**/expected.* or */snapshots/**, or
    deps: vcr, govcr, snapshot-testing libs.
  evidence: medium  severity: info

- id: P4.VER.006
  title: "Verification latency budget declared and respected"
  detect: |
    .archfit.yaml has verification: block; --depth=deep measures actual times
    and compares to declared budgets.
  evidence: strong (deep) / weak (shallow)
  severity: warn

- id: P4.VER.007
  title: "Test pyramid balance (unit ≥ 5× integration; not E2E-only)"
  detect: |
    Count test files matched by patterns: *_test.go, *.test.ts, test_*.py.
    Heuristic classification by directory naming (unit/, integration/, e2e/)
    or by content cues (test framework imports).
    Flag when ratio inverted.
  evidence: weak  severity: info
```

#### P5 — Aggregation of danger(+5 ルール、計 7)

```yaml
- id: P5.AGG.003
  title: "Risk tier file (.archfit risk_tiers OR docs/risk_tiers.yaml)"
  detect: |
    .archfit.yaml has risk_tiers section OR docs/risk_tiers.yaml exists.
  evidence: strong  severity: warn

- id: P5.AGG.004
  title: "High-risk paths protected by CODEOWNERS"
  detect: |
    For each path declared as risk_tiers.high, verify that CODEOWNERS
    matches it with a non-default reviewer.
  evidence: strong  severity: error   # ← 初の error severity
  rationale: |
    高リスク領域に reviewer 設定がないのは構造的脆弱性。
    Strong evidence (file pattern matching), so error severity is justified.

- id: P5.AGG.005
  title: "Idempotency-Key handling on write APIs"
  detect: |
    For repos exposing HTTP endpoints (detected via OpenAPI / framework deps),
    grep for `Idempotency-Key` header handling in HTTP middleware/handlers
    OR a documented idempotency strategy in /docs.
  evidence: weak  severity: info

- id: P5.AGG.006
  title: "Long-lived static credentials avoided (OIDC / Workload Identity)"
  detect: |
    Per IaC / CI:
      - GitHub Actions: workflows must use `permissions:` + `id-token: write`
        for cloud auth; flag if `aws-actions/configure-aws-credentials` 
        uses `aws-access-key-id` instead of `role-to-assume`.
      - Terraform: AWS provider with hardcoded access_key.
  evidence: medium  severity: warn

- id: P5.AGG.007
  title: "Pit-of-success defaults (HTTP client timeout/retry, logger PII mask)"
  detect: |
    Cross-reference framework usage:
      - Go: net/http.Client without Timeout in /internal/
      - TS: fetch/axios without timeout/retry config
      - Logger: structured logging library presence + masking config
    Heuristic.
  evidence: weak  severity: info
```

#### P6 — Reversibility(+3 ルール、計 5)

```yaml
- id: P6.REV.003
  title: "Feature flag actually wired to changed code"
  detect: |
    For repos with feature flag library, check at least N% of files
    in /src/features/* or named features import the flag library.
  evidence: medium  severity: info

- id: P6.REV.004
  title: "Deployment pipeline supports canary / blue-green / staged rollout"
  detect: |
    K8s manifests with `strategy: RollingUpdate` + `maxSurge` / `maxUnavailable`,
    Argo Rollouts, Flagger, AWS deploy-config canary, GitHub Actions deploy
    with environment gates.
  evidence: medium  severity: info

- id: P6.REV.005
  title: "Soft-delete pattern (deleted_at) for user-facing entities"
  detect: |
    Schema files (Prisma, sqlc, migrations, ORMs) — flag tables holding
    PII or business data without deleted_at / archived_at.
  evidence: weak  severity: info
```

#### P7 — Machine-readability(+2 ルール、計 5)

```yaml
- id: P7.MRD.004
  title: "Errors carry structured shape (code/details/remediation)"
  detect: |
    Per-language: detect error type definitions and check for fields named
    code/Code, details/Details, remediation/Remediation OR similar JSON tags.
  evidence: medium  severity: info

- id: P7.MRD.005
  title: "Logs are structured (JSON / logfmt)"
  detect: |
    Logger library detection:
      - Go: zap, slog (with JSONHandler), zerolog
      - Python: structlog, python-json-logger
      - Node: pino, winston (json transport)
    Plus a brief usage check.
  evidence: medium  severity: info
```

これで **30 ルール追加 → 計 47 ルール、すべての原則が 5-9 ルールの幅でカバー**される厚みになります。

### 1.4 Severity 分布の正常化

現状すべて `warn`/`info` という偏りは、**「ツールが組織に拒否される最大要因」**です。次のように調整するべきです:

```
Target distribution (47 rules):
  critical:   1  (例: 平文の長命クレデンシャルがコミットされている)
  error:      4-6 (P5 高リスクパス未保護、CODEOWNERS 未設定 等)
  warn:      ~25  (大半)
  info:      ~15  (規約レベル、低 evidence)
```

ルール:
- `critical` は `Rule.Validate()` で `evidence_strength: strong` 必須にする(既存ルール)
- `error` も `strong` のみ
- `warn` は `medium` 以上
- `info` は `weak` でも可

これで「**重い指摘は嘘がつけない**」が保証されます。

---

## 2. 検出深度の不足を埋める

### 2.1 AST collector の導入(最大の構造的改善)

CLAUDE.md は「`internal/collector/ast/` は roadmap にない」と明言していますが、これは**評価ツールの本質的限界を作り出している**判断であり、再考すべきです。

理由:
- P3.EXP.002〜007、P2.SPC.005、P5.AGG.005、P7.MRD.004 など、**真に価値のあるルールは AST なしには書けない**
- ファイル存在ベースは「形だけ整える」を簡単に許す(コンプライアンスシアター)
- archfit の差別化ポイントは「**深く見る**」ことであるべき

提案する設計:

```
internal/collector/ast/
├── ast.go                    # FactStore に AstFacts を追加
├── treesitter/               # tree-sitter ベース言語横断
│   ├── treesitter.go         # tree-sitter-go バインディングのラッパ
│   ├── go.go
│   ├── typescript.go
│   ├── python.go
│   ├── rust.go
│   ├── java.go
│   └── ruby.go
├── goast/                    # Go 専用の go/parser ベース(精度高)
│   └── goast.go
└── README.md
```

**重要な設計原則**:
- AST 解析は **`--depth=standard` で軽量モード**(構造のみ)、`--depth=deep` で**詳細解析**(関数本体、call graph)
- 巨大 repo 対策に**ファイルサイズ上限・タイムアウト・キャッシュ**(SHA-256 で content-hashed)
- AST 解析失敗は `ParseFailureFinding` を返す(silently ignore しない)
- **Go は標準 `go/parser`、それ以外は tree-sitter**(精度とエコシステムのトレードオフ)

依存追加の justification:
- `github.com/smacker/go-tree-sitter` (BSD-2)
- 各言語の `tree-sitter-<lang>`

PROJECT.md §3.6 で `internal/adapter/fs` 導入で boundary が引き締まったのと同じ規律で、**`internal/collector/ast/` を追加する ADR を切る**ことを推奨します。

### 2.2 Ecosystem collector の拡張

PROJECT.md §6.2 で `internal/collector/ecosystem` の導入が言及されていますが、これを**フレームワーク・デプロイ環境の網羅的カタログ**に育てるべきです:

```go
type Ecosystem struct {
    // CI
    GitHubActions   *GitHubActionsInfo  // workflows, permissions, OIDC usage
    GitLabCI        *GitLabCIInfo
    CircleCI        *CircleCIInfo
    
    // Deployment
    Kubernetes      *K8sInfo            // manifests, strategy
    Helm            *HelmInfo
    ArgoRollouts    bool
    Flagger         bool
    AWSDeploy       *AWSDeployInfo
    
    // Frameworks
    Spring          *SpringInfo
    Rails           *RailsInfo
    Django          *DjangoInfo
    NextJS          *NextJSInfo
    
    // IaC
    Terraform       *TerraformInfo      // version, providers, OIDC
    Pulumi          *PulumiInfo
    CDK             *CDKInfo
    
    // Migrations
    Migrators       []MigratorInfo      // tool, dir, up/down detection
    
    // Feature flags
    FlagLibraries   []string            // launchdarkly, unleash, flagd, etc
    
    // Secret scanners
    SecretScanners  []string            // gitleaks, trufflehog, etc
}
```

これにより、**ルール側はファクト読み出しに集中**でき、新規ルール開発のリードタイムが大幅に短縮されます。

### 2.3 履歴ベース(sampled)evidence の拡充

現状 `sampled` を使っているのは P1.LOC.004 のみ。git 履歴は宝の山です:

| ルール案 | 検出 | Evidence |
|---------|------|----------|
| `P1.LOC.010` Median PR size in 30 days | `git log --numstat` 集計 | sampled |
| `P4.VER.008` Test changes accompany code changes | `git log` で test file 比率 | sampled |
| `P6.REV.006` Revert frequency healthy (not too high, not zero) | `git log --grep=revert` | sampled |
| `P7.MRD.006` Conventional commits used | コミットメッセージ pattern match | sampled |

**`sampled` evidence のラベルを増やす**ことで、評価軸の幅が一気に広がります。

### 2.4 Cross-rule synthesis(超ルール)

個別ルールの寄せ集めではなく、**複数ルールの合致から推論する meta-finding**を導入:

```yaml
- id: META.001
  title: "Compliance theater suspected"
  detect: |
    P1.LOC.001 passed (CLAUDE.md exists) AND
    P1.LOC.006 fails (CLAUDE.md > 400 lines) AND
    P1.LOC.002 passed at <50% coverage
  severity: warn
  evidence: medium
  rationale: |
    ルートにファイルはあるが内容が薄い・形だけのパターン。

- id: META.002
  title: "High-risk paths not under defense in depth"
  detect: |
    P5.AGG.003 passed (risk tiers declared) AND
    (P5.AGG.001 fails OR P5.AGG.004 fails OR P5.AGG.002 fails)
  severity: error
  evidence: medium
```

これは現アーキテクチャに `internal/synth` パッケージを 1 つ追加すれば実装可能です(全ルール実行後の 2-pass)。

---

## 3. キャリブレーション・実証の不足

### 3.1 「100点」の意味が空虚

現在の self-scan score 100 は「17 ルールで 0 finding」しか意味しません。**ルールが少ない → 高スコアが取りやすい → スコアの説得力が弱い**という構造的問題。

提案:**ルール拡張に伴い、自分のスコアも下がる**ことを受け入れる。Phase 1.5 で 47 ルールに拡張すると、archfit 自体のスコアも 100 → ~88 程度に着地するはず(P3 の AST ベースルール、Idempotency-Key、PBT などで指摘が出る)。これを**ドキュメント化して見せる**ことが、ツールの誠実さの証明になります。

具体的に:

```
docs/self-scan/
  v0.3.x.json    # 17 rules, score 100
  v0.4.0.json    # 47 rules, score 88 ← 期待される
  v0.5.0.json    # 50 rules, score 91 ← 改善履歴
  README.md      # スコア低下と改善を時系列で説明
```

これは「**減点はルールの拡張による**」と注釈しつつ、**archfit 自身が改善し続ける repo であることを示す**最強のマーケティング素材です。

### 3.2 キャリブレーションコーパスの即座の構築

PROJECT.md §6.2 で計画されていますが、**Phase 0 の次は即これ**にすべきです。理由:

- ルールを足すたびに、外部 repo での精度・再現率がわからないと、**閾値が机上の空論**のまま
- false-positive 率を測らないと、`error` severity を増やせない

具体提案:

```
calibration/
├── corpus.yaml                  # 30 repos のリスト、license / scope / type 注釈付き
├── ground_truth/
│   ├── kubernetes/              # 各 repo に対する手動アノテーション
│   │   └── expected_findings.yaml
│   └── ...
├── scripts/
│   ├── nightly_run.sh           # 30 repos に対して archfit scan
│   ├── compute_metrics.py       # precision/recall/F1 per rule
│   └── publish_report.sh
└── reports/
    ├── 2026-05-01.md            # 各ルールの精度推移
    └── ...
```

候補となる 30 repo(Apache 2.0 / MIT が中心):

| カテゴリ | 候補 |
|---------|------|
| Web / SaaS | strapi/strapi, supabase/supabase, posthog/posthog |
| バックエンド単体 | gin-gonic/gin, fastapi/fastapi, ktor/ktor |
| 大規模モノレポ | grafana/grafana, vercel/next.js |
| CLI ツール | charmbracelet/bubbletea, junegunn/fzf, sharkdp/bat |
| データ基盤 | apache/kafka, ClickHouse/ClickHouse |
| IaC | hashicorp/terraform-aws-modules |
| モバイル | facebook/react-native, expo/expo |
| MLOps | mlflow/mlflow |
| エージェント関連 | langchain-ai/langchain, microsoft/autogen |
| 模範例 | golang/go, rust-lang/rust |
| 反面教師 | (匿名化) 内部レガシー風 fixture を構築 |

この**コーパスは公開**(`calibration/`)し、夜間 CI で結果を `docs/calibration/` に publish する。これは archfit にとっての「**売れる差別化ポイント**」になります。

### 3.3 Confidence の動的調整(Adaptive Rule Engine の前哨)

PROJECT.md で「Adaptive Rule Engine」が research track にありますが、もっと軽量な版を先行実装可能です:

```go
type RuleStats struct {
    RuleID            string
    TotalFires        int
    TruePositives     int   // user accepted fix
    FalsePositives    int   // user suppressed with reason
    AverageConfidence float64
}
```

`.archfit-stats.json`(オプトイン、`--telemetry-local`)に記録し、`fix --apply` の成否や `ignore` の蓄積から、ルールの**実効 confidence を動的調整**。最初はローカル計測のみで、組織テレメトリは Phase 3 以降。

---

## 4. スコアリング・メトリクスの再設計

### 4.1 スコアの粒度問題

17 ルールで 0-100 → 1 ルール = 5-10 ポイント動く粒度。これは:
- PR ごとの感度が高すぎる(ノイズ)
- ベースライン管理が難しい
- 「100 点」が達成可能な目標になり、保守化を招く

提案:**スコアを 3 層に分ける**

```json
{
  "scores": {
    "overall": 78.4,
    "by_principle": { "P1": 80, "P2": 85, ... },
    "by_dimension": { "P1.LOC": 75, "P1.LOC.changes": 80, ... },
    "by_severity_class": {
      "critical_pass_rate": 1.0,     // critical findings 全部 0 か
      "error_pass_rate": 0.95,
      "warn_pass_rate": 0.80,
      "info_pass_rate": 0.60
    }
  }
}
```

そして**「Overall score だけを見るな」**を README で強く打ち出す。代わりに **Severity Pass Rate**(critical/error が 100% であることが本質)を一次指標に。

### 4.2 メトリクスの first-class 化

PROJECT.md にあるメトリクス(`context_span_p50` など)は、**ルールの副産物ではなく一次評価軸**として扱うべきです。出力上は:

```json
{
  "metrics": {
    "context_span": { "p50": 4, "p90": 12, "n_samples": 87, "trend_30d": "-2" },
    "verification_latency_s": {
      "lint": 3.2, "typecheck": 8.1, "unit": 45.0,
      "budget_ratio": 0.92  // 宣言予算に対する充足度
    },
    "invariant_coverage": {
      "declared": 23, "machine_enforced": 18, "ratio": 0.78
    },
    "blast_radius": {
      "max_score": 7.5, "p90": 3.2,
      "high_risk_path_ratio": 0.12
    }
  }
}
```

そして **trend サブコマンドの主役を score ではなくメトリクス**にする:

```
$ archfit trend --metric=verification_latency_s.unit
2026-03-01: 32.1s
2026-04-01: 45.0s   ← 悪化
2026-05-01: 28.4s
```

### 4.3 「rule fired = score down」の決別

スコア式の見直し:

現状(推定): `score = 100 - Σ(weight × severity)`

提案: **applied weight × confidence × evidence**

```
contribution_i = passed_i × weight_i × evidence_factor_i
score = 100 × Σ contribution_i / Σ weight_applied_i

evidence_factor:
  strong → 1.0
  medium → 0.85
  weak   → 0.7
  sampled → 0.8
```

これで**弱い証拠で減点しすぎる**ことを防ぎ、ルール追加の自由度が上がります。

---

## 5. 出力・機械可読性の改善

### 5.1 Provenance / Reproducibility

現在の output schema は「何が見つかったか」を返しますが、**「どう見つけたか」**の追跡性が弱い。エージェントが結果を信用するには provenance が必要。

提案する追加フィールド:

```json
{
  "schema_version": "1.0.0",
  "scan_id": "20260502T1134Z-abc123",
  "tool": {
    "name": "archfit",
    "version": "0.4.0",
    "build_commit": "a1b2c3d",
    "build_date": "2026-05-01T00:00:00Z"
  },
  "config": {
    "config_file": ".archfit.yaml",
    "config_hash": "sha256:...",
    "profile": "standard",
    "packs_enabled": ["core", "agent-tool"],
    "rules_evaluated": 47,
    "rules_skipped_inapplicable": 5
  },
  "environment": {
    "git": { "commit": "f0e9d8c", "branch": "main", "dirty": false },
    "depth": "standard",
    "duration_s": 12.4,
    "collectors_run": ["fs", "git", "ast", "ecosystem"]
  },
  "findings": [...]
}
```

**`scan_id` は `record/diff/trend` の主キー**として使う。

### 5.2 Finding の evidence 拡充

現状の evidence は `map[string]any` で自由形式。提案:**型付き evidence variants** をスキーマに加える。

```yaml
# schemas/output.schema.json (抜粋)
finding:
  properties:
    evidence:
      oneOf:
        - $ref: "#/definitions/FilePresenceEvidence"
        - $ref: "#/definitions/FilePatternEvidence"
        - $ref: "#/definitions/AstPatternEvidence"
        - $ref: "#/definitions/GitSampleEvidence"
        - $ref: "#/definitions/CommandResultEvidence"
        - $ref: "#/definitions/CrossReferenceEvidence"
```

これにより、エージェントが evidence の種類で**自動分岐**できるようになります(file 系は手動で確認、ast 系は信頼して自動修正、など)。

### 5.3 SARIF の確度向上

PROJECT.md §6.4 で「SARIF certified」がゴールに挙がっていますが、SARIF の `properties` を有効活用すべきです。GitHub Code Scanning の dashboard でフィルタするのに使われます:

```json
{
  "ruleId": "P5.AGG.004",
  "properties": {
    "principle": "P5",
    "evidence_strength": "strong",
    "stability": "stable",
    "auto_fixable": true,
    "tags": ["security", "high-risk-path", "agent-impact-high"]
  }
}
```

`agent-impact-high` というタグは特に**エージェント時代の評価ツール**としての差別化要素。

---

## 6. エージェント統合の深化

### 6.1 SKILL.md → 実行型スキルへの転換

現状の `.claude/skills/archfit/` は静的ドキュメント中心。エージェントを**真に駆動する**には、scripts/ に「決定木を実装したスクリプト」を置くべきです:

```
.claude/skills/archfit/
├── SKILL.md
├── scripts/
│   ├── triage.sh           # archfit scan --json | filter critical/error
│   ├── plan_remediation.sh # 各 finding に対する fix 順序を出力
│   ├── apply_safe_fixes.sh # auto-fix 可能なものから順次
│   └── verify_loop.sh      # fix → re-scan → 改善確認のループ
```

`scripts/triage.sh`(例):

```bash
#!/usr/bin/env bash
set -euo pipefail
archfit scan --json . | jq '
  .findings 
  | map(select(.severity == "critical" or .severity == "error"))
  | sort_by(.confidence) | reverse
  | .[0:5]
'
```

これでエージェントが**「次に何を直すか」を archfit に問い合わせる**設計になり、スキルが教科書ではなく**作業端末**になります。

### 6.2 Per-rule remediation の統一テンプレート

`reference/remediation/<rule-id>.md` のフォーマットが現状不統一です。エージェントが安定して読み取れるよう、**全ルール共通の構造化フォーマット**を強制:

```markdown
# P5.AGG.004 — High-risk paths protected by CODEOWNERS

## decision_tree

condition: risk_tiers.high declared in .archfit.yaml?
  yes:
    condition: CODEOWNERS file exists?
      yes:
        action: add patterns for each high-risk path
        autonomy: auto_with_user_review
        difficulty: low
      no:
        action: create CODEOWNERS, add patterns
        autonomy: auto_with_user_review
        difficulty: low
  no:
    action: ask user to declare risk_tiers first
    autonomy: ask_user
    difficulty: requires_input

## minimal_fix
```yaml
# CODEOWNERS
/src/auth/        @your-org/security
/src/billing/     @your-org/payments
/migrations/      @your-org/data-platform
```

## verification
re-run: archfit check P5.AGG.004 .
expected: passes
```

エージェントは `decision_tree` を機械的に降りる。`autonomy` フィールドで人間確認の要否がわかる。**これが本当の意味でのエージェントスキル**です。

### 6.3 Skill 自体の archfit スコア化

archfit は agent-tool として `.claude/skills/archfit` 自体を評価対象に含めるべきです:

```yaml
- id: P7.MRD.010
  title: "Agent skill files (.claude/skills/*/) follow size limits"
  detect: |
    For each SKILL.md in .claude/skills/*/, check ≤ 400 lines, ≤ 10 KB.
    For each file in reference/remediation/*.md, check ≤ 100 lines.
  evidence: strong  severity: warn
```

これにより「**archfit が他の agent-tool repo を評価するときの目利きが鋭くなる**」と同時に、archfit 自身のスキルも自動チェックされます。

---

## 7. 言語・スタック対応の不均衡

### 7.1 現状の偏り

README から推察される主要対応:
- ✅ Go, Node/TS, Python, Rust, Java(Maven/Gradle)
- △ Ruby(Rails 検出のみ)、PHP、Elixir
- ❌ Swift / Kotlin(モバイル)、Scala 深く、C/C++、Dart/Flutter
- ❌ IaC(Terraform/CDK/Pulumi)の深い解析
- ❌ データ基盤(Avro/Kafka/Schema Registry)

### 7.2 適用範囲の明示化(applies_to の徹底)

PROJECT.md §6.2 で `Languages()` が `FactStore` に追加されたとありますが、これを全ルールに**徹底適用**すべきです:

```yaml
# packs/core/rules/P3.EXP.002.yaml
applies_to:
  languages: [go]                    # 必須
  detected_by: [go.mod, *.go files]
  min_files: 5                       # 検出に必要なファイル数下限
```

そして、適用外の言語の repo に対しては `--explain-coverage` で:

```
$ archfit scan --explain-coverage repo-rust/
rules evaluated: 12 / 47
  skipped: P3.EXP.002 (Go-specific, no .go files)
  skipped: P2.SPC.005 (no domain layer detected)
  ...
```

を明示。これにより**「Go ルールしか動いてないのに 100 点」**といった誤解を防ぎます。

### 7.3 `mobile` / `iac` / `data-event` pack の優先付け

PROJECT.md §6.3 Phase 2 で「first external pack」が選択肢として挙がっていますが、優先順位は次を提案:

1. **`iac`** ← 最優先。Terraform/CDK は**エージェントが実害を出しやすい**領域、かつ既存の OPA/Conftest と連携できる
2. **`data-event`** ← 第2優先。スキーマレジストリ/idempotency 系は archfit の差別化が効く
3. **`mobile`** ← 第3。モバイルはローカル検証ループの貢献が顕著だが、対応コストが高い
4. `web-saas` ← 後回しでよい(既存ツールが多い)
5. `desktop` ← 当面対象外

### 7.4 `iac` pack の最初の 8 ルール(具体例)

```yaml
- id: P5.IAC.001
  title: "IaC layered (raw / hardened module / blueprint / app stack)"
  detect: directory structure under infra/, terraform/, cdk/
  evidence: medium  severity: info

- id: P5.IAC.002  
  title: "Policy-as-code in CI (Conftest, Checkov, Terrascan, tfsec)"
  detect: CI workflow + dep
  evidence: strong  severity: warn

- id: P5.IAC.003
  title: "Drift detection configured (driftctl / terraform plan in CI)"
  detect: CI workflow
  evidence: medium  severity: info

- id: P5.IAC.004
  title: "Apply restricted (plan-only for agents/PRs)"
  detect: workflow split between plan (PR) and apply (manual/main)
  evidence: medium  severity: warn

- id: P6.IAC.001
  title: "State backend remote and locked"
  detect: terraform backend != local
  evidence: strong  severity: error  # 高重要度

- id: P5.IAC.005
  title: "Secrets via reference, not literal"
  detect: hardcoded `password = "..."` in .tf files
  evidence: strong  severity: error

- id: P2.IAC.001
  title: "Modules versioned (source = git ref or tag)"
  detect: source = "github.com/.../?ref=" or registry version
  evidence: strong  severity: warn

- id: P4.IAC.001
  title: "Plan output validated in CI (terratest / static check)"
  detect: CI step running terraform validate / plan
  evidence: medium  severity: info
```

これを**`packs/iac/` として external pack の最初の試金石**にすると、外部 pack 公開ワークフローの検証も同時にできます。

---

## 8. 運用機能・操作性

### 8.1 PR mode を Phase 2 から Phase 1.5 に前倒し

PROJECT.md §6.3 で `archfit pr-check` は Phase 2 ですが、**実際の運用価値はこれが一番大きい**ので前倒し推奨。理由:

- スコアの絶対値より、**PR 単位の差分**のほうがレビュアー(人・エージェント両方)に刺さる
- baseline.json の管理が雑だと運用で破綻するため、専用コマンドにしたほうが堅牢
- GitHub Actions / GitLab CI の例が早く出ると採用が加速する

最小実装:

```bash
$ archfit pr-check --base origin/main
... scanning base ref in worktree
... scanning HEAD
... computing diff

new findings:    3 (1 error, 2 warn)
fixed findings:  1
unchanged:      14

Score delta: 78.4 → 76.1 (-2.3)

DETAILS:
  + [error] P5.AGG.004 src/billing/ — no CODEOWNERS pattern
  + [warn]  P3.EXP.002 internal/x.go:42 — init() side-effect added
  ...
```

### 8.2 Monorepo モードの設計骨子

```yaml
# .archfit.yaml
version: 1
workspace:
  mode: monorepo
  tool: pnpm                          # auto / pnpm / yarn / npm / cargo / go-work / nx
  packages:
    - path: apps/api
      project_type: [web-saas]
      packs: [core, web-saas]
    - path: apps/mobile
      project_type: [mobile]
      packs: [core, mobile]
    - path: packages/auth
      project_type: [shared-lib]
      packs: [core]
      risk_tier: high                 # 横断重要パッケージ
```

`archfit scan` は各 package を独立にスキャンし、**workspace summary** + **per-package** で出力。

### 8.3 incremental scan(差分スキャン)

巨大 repo で**毎 PR フルスキャンは無駄**です。git の changed paths から「**この変更で再評価必要なルールだけ**」を導出:

```
$ archfit scan --since=origin/main
analyzing 42 changed files affecting:
  - 8 rules require re-evaluation
  - 39 rules unchanged (using baseline)
... 
```

これは:
- 各ルールの `applies_to.path_globs` を活用
- 「globally applicable」と「path-scoped」を区別
- baseline scan のキャッシュを `.archfit-cache/`(オプトイン)

### 8.4 `archfit fix` の安全弁強化

現状の fix engine は scan → plan → snapshot → apply → re-scan → rollback と既に堅牢ですが、追加すべき safety net:

- **PR-only mode**: `archfit fix --pr-mode` で git branch 作成 + PR 自動作成、main へ直接コミットしない
- **dependency rules**: ある fix が他 fix を前提とする場合の依存解決(例: `P1.LOC.001` を fix してから `P1.LOC.002`)
- **fix budget per rule**: `--max-fixes-per-rule=10` で暴走防止
- **interactive mode**: `archfit fix --interactive` で 1 件ずつ確認

---

## 9. メタ一貫性・自己評価の精緻化

### 9.1 「self-scan score 100」の脱・絶対化

現在の CLAUDE.md §19 は "score must not drop on any PR" を gate にしていますが、これは:
- ルール追加時にスコアが下がるのを抑制する圧力 ← **逆効果**
- 「下がらない」のではなく「**ルール拡張に対する透明性**」を価値に

提案する CI gate:

```
self-scan check:
  passes if any of:
    1. score >= score_on_main, OR
    2. score < score_on_main BUT new rules introduced (rule count increased)
       AND score_on_main_with_new_rules == current_score
       (新ルールが新たに発動した分のみ低下)
```

これは **「ルールを足すために score を下げない」と「リファクタで score を保つ」を両立**します。

### 9.2 Self-scan の depth=deep ルーチン化

現在の self-scan は標準 depth で素通りしています。`make self-scan-deep` で全コレクタを動かす CI ジョブを追加:

```yaml
# .github/workflows/self-scan-deep.yml
on:
  schedule:
    - cron: '0 3 * * *'   # nightly
jobs:
  deep-scan:
    runs-on: ubuntu-latest
    steps:
      - run: make self-scan-deep
      - run: archfit metric verification_latency_s
      - uses: actions/upload-artifact@v4
        with: { path: scan-deep.json }
```

これで `verification_latency_s` の trend が**自動で蓄積**され、archfit 自身が遅くなったら検知できます。

### 9.3 `IDEA.md` と運用の温度差

IDEA.md は「scan score 100」を強調していますが、**この発信は近視眼的**です。現実には:

- 17 ルールで 100 = サーフェス品質保証のみ
- ルール拡張で必ず 100 は崩れる
- それは進歩であって退化ではない

IDEA.md を改訂し、**「self-scan は絶対値ではなく成長軌跡を示す」**という stance に切り替えることを推奨。

---

## 10. 優先順位付きロードマップ(改訂提案)

PROJECT.md の Phase 0/1/2/3 を踏まえ、**Phase 1 〜 Phase 2 をより具体化**した提案:

### Phase 1.0(now → 4 週間)— 鋭さの底上げ

P0 として実施:

1. **AST collector の導入**(1 ADR + `internal/collector/ast/`)— 全 P3 ルール拡張の前提
2. **Severity class の正常化**(`error` を 4-6 ルール導入)
3. **Rule expansion: P3 を 1 → 6 ルールに**(P3.EXP.002〜007)
4. **Rule expansion: P5.AGG.004(CODEOWNERS strict、初の error)+ P5.AGG.003 risk-tier**
5. **Calibration corpus v0**(10 repos で start)
6. **PR mode 前倒し**(`archfit pr-check` 実装)

成果物:`v0.4.0`、ルール数 17 → 24、深さ大幅向上。

### Phase 1.5(4 〜 10 週間)— 網羅の拡大

7. **Rule expansion: P2 + P4 + P6 = +12 ルール**(計 36 ルール)
8. **`iac` pack first cut**(8 ルール、external pack 公開ワークフロー検証)
9. **Calibration corpus v1**(30 repos、precision/recall publish)
10. **Score model 改訂**(severity_pass_rate / metrics first-class)
11. **Output schema v1.1**(provenance / typed evidence)
12. **Skill scripts化**(triage / plan / verify_loop)

成果物:`v0.5.0`、ルール数 ~44、外部 pack 1 つ、出力 schema 1.1。

### Phase 2(10 〜 24 週間)— 規模対応

13. **Monorepo / workspace mode**
14. **Incremental scan**(差分のみ)
15. **`data-event` pack**
16. **Adaptive Rule Engine v0**(ローカル ignore/fix からの confidence 調整)
17. **Output schema v1.2**(meta-findings / cross-rule synthesis)

### Phase 3(toward 1.0)

18. **Rule ID freeze**(全ルール stable 化、ID 不変保証)
19. **JSON schema v1.0 freeze**
20. **Public stability statement**
21. **Calibration repository public listing**

---

## 11. 個別の中規模問題

### 11.1 `.archfit.yaml` の表現力不足

現行は YAML パースされるようになりましたが、`risk_tiers` が**ルールから参照されていない**(P5.AGG.001 が独自に security 関連ファイルを検出するなど)。提案:

```yaml
# .archfit.yaml の表現力拡張案
version: 2                              # ← 既存 v1 と互換、v2 でリッチに

risk_tiers:
  critical:
    paths: ["src/auth/**", "migrations/**"]
    require_codeowners: true
    require_intent_md: true
    require_runbook: true
  high:
    paths: ["src/billing/**", "infra/**"]
    require_codeowners: true
  medium:
    paths: ["src/features/**"]
  low:
    paths: ["docs/**", "tests/**"]

verification:                            # PR2 PROJECT.md §6.3 とも整合
  lint:        { command: "make lint",      timeout_s: 5,    layer: 1 }
  typecheck:   { command: "make typecheck", timeout_s: 10,   layer: 1 }
  unit:        { command: "make test",      timeout_s: 60,   layer: 2 }
  integration: { command: "make e2e",       timeout_s: 300,  layer: 3 }

agent_directives:                        # ← 追加提案
  forbidden_paths: ["secrets/**", "third_party/vendored/**"]
  caution_paths:   ["src/auth/**"]
  default_review_required: ["src/billing/**"]
```

`agent_directives` は archfit が**直接エージェントに渡す情報**で、CLAUDE.md / AGENTS.md にも反映されるべき。

### 11.2 `--with-llm` の限界と改善

現状 LLM には「rule metadata + evidence」のみ送り、ソースコードは送らない設計。これは安全だが**実質的価値が薄い**。改善案:

```
--with-llm-mode={off|metadata|file-snippet|full-context}
  off:           LLM 呼ばない
  metadata:      現状(evidence のみ)
  file-snippet:  違反箇所周辺 N 行(default 30 行)を送る ← 新提案
  full-context:  違反ファイル全体(opt-in、warn 表示)
```

`file-snippet` モードは、ユーザに**明示確認**(`Sending lines 80-110 of src/auth/handlers.go to Anthropic. Confirm? [y/N]`)を出してから送信。これで**ローカル静的解析 + LLM 文脈推論のハイブリッド**になり、archfit の差別化が効きます。

### 11.3 観測性

archfit 自身の観測性が薄い。提案:

```
$ archfit scan --metrics-otlp=http://localhost:4317 .
```

OpenTelemetry でスキャン全体の trace/span を出力。ルール実行ごとに span を切れば、**遅いルール / 失敗するルール**が即座に判別できます。これは archfit を**プラットフォームチームの計測対象**にできる(自分の P7 を強化する)。

### 11.4 セキュリティ:LLM プロンプトインジェクション耐性

`--with-llm` 時に、対象 repo の README やコメントにプロンプトインジェクションが仕込まれている可能性があります(将来的に file-snippet モードを入れた場合)。対策:

```
internal/adapter/llm/sanitizer.go
  - Strip HTML/Markdown comments before sending
  - Detect and label prompt-injection-like patterns:
    "ignore previous", "you are now", "system:", etc.
  - Warn when sample contains > 20% non-ASCII
```

これは**他社の coding agent ツールがほとんど対策していない**領域で、archfit の差別化要因になり得ます。

### 11.5 スキーマ駆動が部分的

PROJECT.md の Phase 0 で「YAML が source of truth」になりましたが、検査側は不完全:

- `schemas/output.schema.json` の strict validation は CI にあるが、エンドユーザがローカルで確認する手段が弱い
- `archfit validate-output <file.json>` を追加して、**任意の JSON 出力をスキーマ検証**できるようにする

---

## 12. リスクと未決事項

### 12.1 ルール拡張による偽陽性増加

47 ルールに拡張すると、初期は false-positive が増えます。緩和策:

- **新ルールは必ず experimental から**(既に CLAUDE.md §8 で規約化済み)
- **Calibration コーパスで precision >= 0.85 を達成するまで stable に昇格しない**
- ルール docs に **「known false-positive cases」** セクションを必須化

### 12.2 AST collector の保守コスト

tree-sitter は文法アップデートで挙動変化のリスク。緩和:

- 言語ごとに**最低限のパース回復力**を持たせる(エラー位置で fail-soft)
- バージョンを `go.sum` で固定
- 各言語のパースが失敗したら**スコアに影響しないが finding は出る**(ParseFailure)

### 12.3 設計者と評価対象の同一性

archfit の評価規範は archfit 自身が「正しい」と思っている形に偏る危険があります。緩和:

- **calibration コーパスを多様にする**(モバイル、ML、Rust、組み込み等)
- **community feedback ループ**を `archfit feedback` コマンドで明示的に開く
  ```
  $ archfit feedback P3.EXP.002 --suppress
  Why are you suppressing this rule? > [user input]
  Send to telemetry? [y/N]
  ```

### 12.4 バージョン凍結と進化のトレードオフ

ADR 0012 で全ルール `stability: stable` 凍結というのは**Phase 1 の柔軟性を奪う早すぎる決断**の可能性があります。提案:

- **核ルール(P1.LOC.001 など 5-7 個)のみ stable**
- 残りは **experimental → stable へ段階移行**
- ADR 0012 を修正する ADR を切ってもよい

---

## 13. 1 行サマリ

archfit の現状は**整合性のある基礎フレームワーク**として高品質ですが、評価ツールとしての**密度・深さ・実証性**が次の山です。具体的には、(1) **ルールを 17 → 47 に拡張**し原則間の偏在を是正、(2) **AST collector を導入**してファイル存在ベースから構造ベースへ深化、(3) **キャリブレーションコーパスで実証**、(4) **メトリクスを first-class 化してスコア依存を脱却**、(5) **PR mode と incremental scan を前倒し**して運用価値を引き上げる、の 5 点を Phase 1.0 〜 1.5 で実施することを推奨します。

特に重要なのは **「スコア 100 を達成可能なツールから、スコア 100 が到達困難で意味のあるツールへ」** の転換です。これにより、archfit は「形式的なチェッカー」から「**エージェント時代のアーキテクチャ品質を本当に測れる稀有な計器**」になります。