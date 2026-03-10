# Kapitel 5: Resources – Das Gedächtnis des Modells

Während Tools für Aktionen gedacht sind, dienen **Resources** dazu, dem LLM Wissen oder Daten zur Verfügung zu stellen. In der MCP-Spezifikation (Stand 2025-11-25) sind Resources ein zentraler Bestandteil, um externe Datenquellen sicher einzubinden.

## Was ist eine MCP Resource?

Eine Resource ist vergleichbar mit einer Datei oder einem Datenbankeintrag. Jede Resource wird über eine eindeutige **URI** identifiziert (z. B. `file:///logs/today.txt`).

### Wichtige Merkmale:
*   **Identifikation**: Über URIs (Uniform Resource Identifiers).
*   **Inhalt**: Kann Text (`Text`) oder Binärdaten (`Blob`) sein.
*   **Metadaten**: Beinhaltet Name, Beschreibung und den MIME-Typ (`MIMEType`).
*   **Icons (Neu 2025-11)**: Jede Ressource und jedes Resource-Template kann nun Icons für die visuelle Darstellung in Clients enthalten (SEP-973).

## Implementierung in Go

### 1. Statische Resource
Hier nutzen wir `s.AddResource`. Das Go-SDK (v1.4+) erlaubt es, Metadaten-Icons über das Interface hinzuzufügen.

```go
s.AddResource(&mcp.Resource{
    Name:        "App Config",
    URI:         "file:///config.json",
    Description: "Die aktuelle Systemkonfiguration",
    MIMEType:    "application/json",
}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
    return &mcp.ReadResourceResult{
        Contents: []*mcp.ResourceContents{
            {
                URI:      "file:///config.json",
                MIMEType: "application/json",
                Text:     `{"env": "production", "debug": false}`,
            },
        },
    }, nil
})
```

### 2. Dynamische Resource Templates
Templates erlauben es, auf unendlich viele Ressourcen zu reagieren, ohne sie alle einzeln aufzulisten.

```go
s.AddResourceTemplate(&mcp.ResourceTemplate{
    Name:        "User Profile",
    URITemplate: "users://{id}/profile",
    Description: "Zugriff auf Benutzerprofile über ID",
}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
    // Die konkrete URI aus dem Request auslesen
    uri := request.Params.URI
    return &mcp.ReadResourceResult{
        Contents: []*mcp.ResourceContents{
            {
                URI:  uri,
                Text: "Dies sind die Details für " + uri,
            },
        },
    }, nil
})
```

## Echtzeit-Updates: Subscriptions

Neu in der aktuellen Spezifikation ist die Möglichkeit, Ressourcen zu abonnieren (`resources/subscribe`). Wenn Sie in den `ServerOptions` die Capabilities `Subscribe: true` setzen, kann der Server den Client über Änderungen informieren.

## Resources testen mit `mcp-tester`

Mit dem `mcp-tester` können Sie prüfen, ob Ihr Server die Ressourcen korrekt ausliefert:

```bash
# Ressourcen auflisten (inkl. Pagination Support mit -C)
./bin/mcp-tester resources list --profile local

# Templates auflisten
./bin/mcp-tester resources templates --profile local

# Ressource auslesen
./bin/mcp-tester resources read "file:///config.json" --profile local
```

Im nächsten Kapitel schauen wir uns an, wie wir mit **Prompts** die Interaktion zwischen Mensch und Maschine perfekt orchestrieren.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Prompts →](06_prompts_die_anweisungen.md)

---
*Copyright Michael Lechner - 2026-03-09*
