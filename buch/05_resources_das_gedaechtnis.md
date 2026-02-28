# Kapitel 5: Resources – Das Gedächtnis des Modells

Während Tools für Aktionen gedacht sind, dienen **Resources** dazu, dem LLM Wissen oder Daten zur Verfügung zu stellen. In diesem Kapitel lernen wir den Unterschied zwischen statischen Ressourcen und dynamischen Templates kennen.

## Was ist eine MCP Resource?

Eine Resource ist vergleichbar mit einer Datei oder einem Datenbankeintrag, den der Client dem Modell "zeigen" kann. Jede Resource wird über eine eindeutige **URI** identifiziert (z. B. `file:///logs/today.txt` oder `db://users/123`).

Es gibt zwei Arten von Ressourcen:
1.  **Statische Ressourcen**: Feste Datenquellen, die beim Handshake aufgelistet werden.
2.  **Resource Templates**: Dynamische Pfade, die Variablen enthalten (z. B. `logs://{date}`).

## Implementierung in Go

### 1. Statische Resource hinzufügen
Hier nutzen wir `s.AddResource`. Der Server meldet dem Client sofort, dass diese Daten existieren.

```go
s.AddResource(mcp.NewResource("file:///config.json", "App Config",
    mcp.WithResourceMimeType("application/json"),
), func(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
    return &mcp.ReadResourceResult{
        Contents: []mcp.ResourceContents{
            mcp.TextResourceContents{
                URI:      "file:///config.json",
                MimeType: "application/json",
                Text:     `{"env": "production", "debug": false}`,
            },
        },
    }, nil
})
```

### 2. Dynamische Resource Templates
Templates erlauben es, auf unendlich viele Ressourcen zu reagieren, ohne sie alle einzeln aufzulisten.

```go
s.AddResourceTemplate(mcp.NewResourceTemplate("users://{id}/profile", "User Profile"),
    func(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
        // Die URI aus dem Request enthält die konkrete ID
        uri := request.Params.URI 
        return &mcp.ReadResourceResult{
            Contents: []mcp.ResourceContents{
                mcp.TextResourceContents{
                    URI:  uri,
                    Text: "Dies sind die Profil-Details für URI: " + uri,
                },
            },
        }, nil
    },
)
```

## Resources testen mit `mcp-tester`

Resources werden im MCP-Flow oft vom Modell "gelesen", bevor es eine Antwort generiert. Mit dem `mcp-tester` kannst du prüfen, ob der Server die richtigen Daten liefert.

### Ressourcen auflisten
```bash
./bin/mcp-tester resources list --profile local
```

### Ressource auslesen
Hier gibst du die volle URI an. Bei Templates setzt du einfach einen Wert für den Platzhalter ein:
```bash
./bin/mcp-tester resources read "users://42/profile" --profile local
```

## So sieht die Ressource für das LLM aus (JSON)

Im Gegensatz zu Tools, die das Modell *aktiv aufruft*, werden Ressourcen oft vom Client in den Kontext geladen und dem Modell präsentiert. Wenn ein Modell eine Ressource "liest", sieht der JSON-Datenstrom, den der Client an das LLM sendet, so aus:

```json
{
  "model": "gpt-4",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Analysiere bitte die folgende Konfigurationsdatei:"
        },
        {
          "type": "resource",
          "resource": {
            "uri": "file:///config.json",
            "mimeType": "application/json",
            "text": "{\"env\": \"production\", \"debug\": false}"
          }
        }
      ]
    }
  ]
}
```

Das Modell erhält also den Inhalt der Ressource direkt als Teil der Nachricht. Es kann nun Fragen dazu beantworten, ohne selbst eine Funktion ausführen zu müssen.

## Der Unterschied zu Tools
...

Ein häufiger Fehler ist die Verwechslung von Tools und Resources. Merke dir:
*   **Tools** sind Verben: "Sende eine E-Mail", "Addiere Zahlen", "Lösche Datei". Sie verändern oft den Zustand.
*   **Resources** sind Nomen: "Das Logbuch", "Das Benutzerprofil", "Die Konfiguration". Sie dienen primär dem Informationsfluss zum Modell.

Im nächsten Kapitel schauen wir uns an, wie wir mit **Prompts** die Interaktion zwischen Mensch und Maschine perfekt orchestrieren.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Prompts →](06_prompts_die_anweisungen.md)

---
*Copyright Michael Lechner - 2026-02-28*
