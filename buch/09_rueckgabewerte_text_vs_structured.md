# Kapitel 9: Rückgabewerte – Von Text zu strukturierten Daten

In der Anfangszeit von MCP lieferten Tools fast ausschließlich Text zurück. Doch das Protokoll hat sich weiterentwickelt. Heute können MCP-Server Daten so liefern, dass das LLM sie direkt als "Datenobjekt" versteht. In diesem Kapitel schauen wir uns diesen Unterschied genauer an.

## Die klassische Methode: TextContent

Früher (und in einfachen Servern auch heute noch) war der Rückgabewert eines Tools ein einfacher String. Das LLM musste diesen Text lesen und die Informationen selbst wieder extrahieren.

**Beispiel für eine klassische Antwort:**
```json
{
  "content": [
    {
      "type": "text",
      "text": "Benutzer Max (ID: 42) hat ein Guthaben von 150,50 Euro."
    }
  ]
}
```
*Nachteil:* Das Modell muss den Text "parsen". Es besteht die Gefahr, dass es die ID oder den Betrag falsch extrahiert, wenn der Text kompliziert formuliert ist.

---

## Die moderne Methode: StructuredContent

Moderne MCP-Server nutzen das Feld `structuredContent`. Hier werden die Daten als echtes JSON-Objekt übertragen. Das LLM erhält die Informationen "rein" und ohne störenden Fließtext drumherum.

**Beispiel für eine strukturierte Antwort:**
```json
{
  "content": [
    { "type": "text", "text": "Hier sind die Benutzerdaten:" }
  ],
  "structuredContent": {
    "user_id": 42,
    "name": "Max",
    "balance": 150.50,
    "currency": "EUR"
  }
}
```
*Vorteil:* Das Modell greift direkt auf die Felder zu. Die Fehlerquote bei der Weiterverarbeitung der Daten (z. B. für eine Berechnung im nächsten Schritt) sinkt gegen Null.

---

## Implementierung im Go SDK

Das offizielle Go SDK unterstützt beide Welten. Wenn du einen `ToolHandlerFor` verwendest und ein Go-Struct zurückgibst, füllt das SDK automatisch den `structuredContent` für dich aus.

```go
type UserResponse struct {
    ID      int     `json:"user_id"`
    Name    string  `json:"name"`
    Balance float64 `json:"balance"`
}

// Im Handler:
return nil, UserResponse{ID: 42, Name: "Max", Balance: 150.50}, nil
```

## Validierung mit `mcp-tester`

Unser `mcp-tester` wurde speziell dafür gebaut, beide Formate anzuzeigen. Beim Aufruf eines Tools im `call`-Befehl siehst du eine klare Trennung:

```bash
./bin/mcp-tester call get_user --profile local
```

**Die Ausgabe zeigt dir:**
*   `Content 0 (Text)`: Der für Menschen lesbare Teil.
*   `StructuredContent`: Das für die KI gedachte Datenobjekt.

Durch diese Unterscheidung kannst du als Entwickler sicherstellen, dass dein Server nicht nur "schönen Text" für den Chat liefert, sondern auch "saubere Daten" für die Logik des Modells.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Binärdaten →](10_binaerdaten_bilder.md)

---
*Copyright Michael Lechner - 2026-02-28*
