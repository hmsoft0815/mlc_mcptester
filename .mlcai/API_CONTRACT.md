# 🔌 API Contract: MCP-Tester

## 1. Bereitgestellte Endpunkte (Exposed)

### Model Context Protocol (Client)
Dieses Projekt agiert primär als Client und "spricht" MCP zu externen Servern über:
- **stdio**: Startet lokale Prozesse und kommuniziert über stdin/stdout.
- **SSE (HTTP)**: Verbindet sich mit Remote-Endpunkten (z.B. `http://localhost:8080/sse`).

### Referenz-Server (test-server)
Beinhaltet einen Test-Server zur Validierung von MCP-Clients:
- **Basis-URL:** `http://localhost:8081`
- **Endpoint:** `/sse` (Server-Sent Events für MCP-Transport)
- **Features:** Exponiert Tools (`echo`, `add`, `long_running`), Resources und Prompts.

### Scripting DSL (.mcp)
Ermöglicht die Definition von Test-Szenarien:
- **Dateiendung:** `.mcp`
- **Commands:** `call_tool`, `set_var`, `assert_contains`, `assert_equals`, `assert_gt`, `expect_error`, etc.

---

## 2. Konsumierte APIs (Consumed)

### Extern: MCP Server
- **Zweck:** Testobjekt für die CLI.
- **Schnittstelle:** Beliebige MCP-kompatible Server (Node.js, Python, Go, etc.).

---

## 3. Globale Konventionen
- **Protokoll-Version:** MCP Spec 2025-03-26.
- **JSON-RPC 2.0:** Alle MCP-Nachrichten folgen dem JSON-RPC 2.0 Standard.
- **Paginierung:** Nutzt `cursor` basierte Paginierung für `list` Befehle (Tools, Resources, Prompts).

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI (v0.2.3)
- **Status:** Aktuell
