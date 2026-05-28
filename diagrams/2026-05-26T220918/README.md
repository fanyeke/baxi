# Baxi Architecture Diagrams

Generated: 2026-05-26T22:09:18 Asia/Shanghai

Files:

- `baxi-system-architecture.mmd`: Mermaid source for the current system architecture.
- `baxi-decision-loop.mmd`: Mermaid sequence diagram for the intended governed decision loop.
- `baxi-aip-alignment.svg`: Human-facing SVG diagram embedded by the research document.

Rendering recommendation:

- Use Mermaid first for source-controlled docs because it is plain text, reviewable, and usually rendered by Markdown tooling.
- Use SVG for polished documents that must render consistently without a Mermaid runtime.
- Use a whiteboard export when the target medium is Feishu/Lark whiteboard; the local `@larksuite/whiteboard-cli` render path could not be verified in this environment because the npm download hung after registry access.
