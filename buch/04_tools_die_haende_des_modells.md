# Kapitel 4: Tools – Die Hände des Modells

Während Resources (Daten) dem Modell "Wissen" geben, sind **Tools** die Werkzeuge, mit denen das Modell aktiv werden kann. In diesem Kapitel lernen wir, wie man Tools definiert, implementiert und testet – basierend auf der MCP-Spezifikation vom 2025-11-25 (inkl. SEP-1303 und SEP-973).

## Was ist ein MCP Tool?

Ein Tool ist eine ausführbare Funktion, die der Server dem Client zur Verfügung stellt. Jedes Tool besteht aus:
1.  **Name**: Eine eindeutige ID (z. B. `calculate_sum`).
2.  **Beschreibung**: Ein Text, der dem LLM erklärt, *wann* und *warum* es dieses Tool nutzen sollte.
3.  **Input Schema**: Ein JSON-Schema (Standard: JSON Schema 2020-12), das die Argumente definiert.
4.  **Icons (Neu 2025-11)**: Optionale visuelle Metadaten für Clients.

## Metadaten und Icons (SEP-973)

Seit November 2025 können Tools (sowie Ressourcen und Prompts) visuelle Metadaten enthalten. Dies ermöglicht es Clients, eine ansprechende Benutzeroberfläche mit Icons zu gestalten.

Ein Icon besteht aus:
*   **src**: Eine URL (HTTP/HTTPS) oder ein Data-URI (Base64).
*   **mimeType**: Optionaler Medientyp (z. B. `image/png`).
*   **sizes**: Optionale Angabe der Dimensionen (z. B. `48x48`).

## Implementierung in Go

```go
mcp.AddTool(s, &mcp.Tool{
    Name:        "add",
    Description: "Addiert zwei ganze Zahlen",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "a": map[string]any{"type": "integer"},
            "b": map[string]any{"type": "integer"},
        },
        "required": []string{"a", "b"},
    },
    // Icons können nun als Metadaten hinterlegt werden (SEP-973)
}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
    a, _ := args["a"].(float64)
    b, _ := args["b"].(float64)
    
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            &mcp.TextContent{Text: fmt.Sprintf("Ergebnis: %d", int(a)+int(b))},
        },
    }, nil, nil
})
```

## Strategische Fehlerbehandlung (SEP-1303)

Eine wichtige Änderung in der Spezifikation vom November 2025 betrifft die Validierung. Wenn ein Modell ungültige Argumente sendet, sollte der Server **keinen JSON-RPC Fehler** (Protokoll-Ebene) zurückgeben.

Stattdessen sollte der Fehler als reguläres **Tool-Ergebnis** mit `IsError: true` gesendet werden.
**Warum?** Nur wenn der Fehler im Inhalts-Array landet, kann das LLM die Fehlermeldung "lesen", den Fehler verstehen und einen korrigierten Aufruf starten (Self-Correction).

```go
if b == 0 {
    return &mcp.CallToolResult{
        Content: []mcp.Content{&mcp.TextContent{Text: "Fehler: Division durch Null ist nicht erlaubt."}},
        IsError: true,
    }, nil, nil
}
```

## Tools testen mit `mcp-tester`

Der `mcp-tester` unterstützt die neuesten Standards und erlaubt es, Tool-Aufrufe präzise zu simulieren:

```bash
./bin/mcp-tester tools call add --args '{"a": 10, "b": 5}' --profile local
```

Prüfen Sie in der Ausgabe besonders das `IsError` Flag, um sicherzustellen, dass Ihr Server die neue Fehlerstrategie (SEP-1303) korrekt umsetzt.

[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Resources →](05_resources_das_gedaechtnis.md)

---
*Copyright Michael Lechner - 2026-03-09*
