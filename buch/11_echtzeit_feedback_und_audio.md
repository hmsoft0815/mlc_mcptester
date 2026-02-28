# Kapitel 11: Echtzeit-Feedback & Audio – Mehr als nur Text

Bisher haben wir MCP als ein Anfrage-Antwort-System betrachtet. Doch MCP kann mehr: Es ermöglicht dem Server, dem Client während einer laufenden Aufgabe Feedback zu geben (Push-Prinzip) und unterstützt neben Bildern auch Audio-Inhalte.

## 1. Echtzeit-Feedback vom Server

Wenn ein Tool eine längere Berechnung durchführt, möchte der Benutzer nicht vor einem leeren Bildschirm warten. MCP bietet hierfür zwei Mechanismen:

### Logging (Benachrichtigungen)
Der Server kann jederzeit Log-Nachrichten an den Client senden. Dies ist ideal für Debugging-Informationen oder Status-Updates.
*   **Technisch**: Der Server sendet eine `notifications/message`.
*   **Im Tester**: Sichtbar als `[SERVER LOG] [info] ...`.

### Progress (Fortschrittsanzeige)
Für lang laufende Tools kann der Server einen Fortschrittsbalken füttern.
*   **Voraussetzung**: Der Client sendet ein `progressToken` beim Aufruf mit.
*   **Technisch**: Der Server sendet regelmäßig Updates mit `progress` (aktueller Wert) und `total` (Zielwert).

---

## 2. Audio in MCP

Neben Text und Bildern unterstützt die MCP-Spezifikation auch **AudioContent**. Dies ist besonders wichtig für die nächste Generation von KI-Modellen, die direkt mit Stimme interagieren können.

### Das AudioContent Objekt
Ähnlich wie Bilder werden Audio-Daten als Base64-String übertragen.

**Beispiel für eine Antwort mit Audio:**
```json
{
  "content": [
    {
      "type": "audio",
      "data": "UklGRiYAAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQAAAAA=",
      "mimeType": "audio/wav"
    }
  ]
}
```

### Warum Audio?
1.  **Sprachausgabe**: Ein Tool könnte Text-to-Speech (TTS) auf dem Server ausführen und das Ergebnis direkt als Audio-Datei an das LLM (oder den Client) zurückgeben.
2.  **Sound-Analyse**: Das Modell kann Audio-Ressourcen "hören" und analysieren (z. B. Geräusche in einer Maschinenhalle zur Fehlerdiagnose).

---

## Implementierung in Go

So sendest du Log-Nachrichten und Progress-Updates in einem Tool-Handler:

```go
func handleLongTask(ctx context.Context, request *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
    // 1. Log senden
    request.Session.Log(ctx, &mcp.LoggingMessageParams{
        Level: "info",
        Data:  "Starte Prozess...",
    })

    // 2. Fortschritt melden
    if request.Params.GetProgressToken() != nil {
        request.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
            Progress: 50,
            Total:    100,
            Message:  "Halbzeit!",
            ProgressToken: request.Params.GetProgressToken(),
        })
    }
    
    return mcp.NewToolResultText("Fertig"), nil, nil
}
```

## Validierung mit `mcp-tester`

Unser `mcp-tester` unterstützt diese Funktionen im Verbose-Modus (`-v`):

```bash
./bin/mcp-tester call long_task --profile local -v
```

Du siehst dann live im Terminal, wie der Server seine Logs und Fortschrittsmeldungen "pusht", noch bevor das eigentliche Tool-Ergebnis eintrifft.

## Fazit

Mit Logging, Progress, Bildern und Audio wird MCP zu einer **vollständigen Multimedia-Schnittstelle**. Es erlaubt eine nahtlose Integration von KI in komplexe, interaktive Systeme, die weit über einfaches Chatten hinausgehen.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Grenzen & Limits →](12_grenzen_und_limitierungen.md)

---
*Copyright Michael Lechner - 2026-02-28*
