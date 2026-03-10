# Kapitel 6: Prompts – Die Anweisungen und Vorlagen

Ein **Prompt** im MCP (Model Context Protocol) ist weit mehr als nur ein Textbaustein. Er ist eine serverseitig definierte Vorlage für Nachrichten, die dem LLM eine Identität, Regeln oder spezifischen Kontext geben können. In der aktuellen Spezifikation (Stand 2025-11-25) sind Prompts ein mächtiges Werkzeug zur Steuerung der Interaktion.

## Was ist ein MCP Prompt?

Prompts werden in der Regel durch explizite Benutzeraktionen ausgelöst (z. B. durch "Slash-Commands" wie `/debug` in einem Client). Sie bestehen aus:
*   **Name & Beschreibung**: Zur Identifikation und Erklärung für den Nutzer.
*   **Argumente**: Dynamische Platzhalter. Neu seit November 2025: Unterstützung für Default-Werte (SEP-1034).
*   **Icons (Neu 2025-11)**: Optionale visuelle Metadaten (SEP-973).
*   **Nachrichten**: Eine Liste von Nachrichten, die an das Modell gesendet werden sollen.

## Implementierung in Go

Das Go-SDK stellt die Methode `s.AddPrompt` zur Verfügung.

```go
s.AddPrompt(&mcp.Prompt{
    Name:        "code_review",
    Description: "Analysiert den Code in einer bestimmten Datei",
    Arguments: []*mcp.PromptArgument{
        {Name: "file_path", Description: "Pfad zur Datei", Required: true},
    },
}, func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    path := request.Params.Arguments["file_path"]
    
    return &mcp.GetPromptResult{
        Description: "Code-Review Anweisungen",
        Messages: []*mcp.PromptMessage{
            {
                Role: "user", // "user" oder "assistant" gemäß Spec
                Content: &mcp.TextContent{
                    Text: fmt.Sprintf("Bitte analysiere den Code in: %s", path),
                },
            },
        },
    }, nil
})
```

## Neue Elicitation-Möglichkeiten (SEP-1036)

Seit Ende 2025 unterstützt MCP erweiterte Abfragen (Elicitations). Ein Server kann nun spezifisch nach **URLs** fragen oder komplexe Enums mit Standardwerten nutzen. Dies ermöglicht es, Prompts noch interaktiver zu gestalten, indem der Client gezielt Informationen vom Nutzer oder aus dessen Umgebung (z. B. eine Browser-URL) einfordert.

## Rollen und Inhalte (Spezifikation 2025-11-25)

In der aktuellen Revision sind die Rollen innerhalb eines Prompts streng definiert:
*   **`user`**: Nachrichten, die den Benutzer oder Anweisungen repräsentieren.
*   **`assistant`**: Nachrichten, die eine Antwort der KI simulieren.

**Wichtig**: Der klassische `system`-Prompt wird in dieser Version nicht als Nachrichtentyp in der Liste geführt. Anweisungen mit System-Charakter werden stattdessen als `user`-Nachrichten mit klaren Instruktionen formuliert.

### Reiche Inhalte: Bilder und Ressourcen
Ein Prompt kann nicht nur Text enthalten. Seit dem 26.03.2025 (und bestätigt im Nov 2025) können Prompts direkt auf **Resources** verweisen:

```go
&mcp.PromptMessage{
    Role: "user",
    Content: &mcp.EmbeddedResource{
        Resource: &mcp.ResourceContents{
            URI:  "file:///logs/error.log",
            Text: "Log-Inhalt hier...",
        },
    },
}
```

## Prompts testen mit `mcp-tester`

Nutzen Sie den Tester, um sicherzustellen, dass Ihre Vorlagen korrekt mit Argumenten arbeiten:

```bash
# Verfügbare Prompts auflisten (inkl. Pagination Support mit -C)
./bin/mcp-tester prompts list --profile local

# Prompt abrufen (mit Argumenten)
./bin/mcp-tester prompts get code_review --args '{"file_path": "main.go"}' --profile local
```

[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Die Macht der Kombination →](07_die_macht_der_kombination.md)

---
*Copyright Michael Lechner - 2026-03-09*
