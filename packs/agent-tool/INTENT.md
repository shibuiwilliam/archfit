# Pack: agent-tool — intent

Rules for repositories whose primary consumer is a coding agent: CLIs, MCP servers, code-review bots, and other tools meant to be driven by machines rather than humans.

These rules encode a narrower contract than `core`:

- The tool advertises a **versioned JSON output schema** (P2.SPC.010).
- The tool ships a **machine-readable change log** (P7.MRD.002).
- The tool records its **irreversible design decisions** in an ADR directory (P7.MRD.003).

Each is `strong`-evidence, `experimental` stability. A repo that is not an agent-tool (e.g. a plain web app) would see these rules as irrelevant; therefore the pack is **opt-in** via `.archfit.yaml` — it does not run by default on repos that did not ask for it.

archfit itself enables this pack, because it *is* an agent-tool.
