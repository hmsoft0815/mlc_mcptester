# Kapitel 11: Echtzeit-Feedback & Abbruch – Interaktive Tools

Bisher haben wir MCP als ein einfaches Anfrage-Antwort-System betrachtet. Doch für produktive Anwendungen, die länger dauern, bietet MCP mächtige **Utilities**: Echtzeit-Feedback (Logging, Progress) und die Möglichkeit, laufende Aufgaben abzubrechen (**Cancellation**).

## 1. Fortschritt & Feedback (Progress & Logging)

Wenn ein Tool eine längere Berechnung durchführt (z. B. ein Datenexport oder eine Video-Konvertierung), sollte der Server dem Client Status-Updates senden.

### Logging (Strukturiertes Protokollieren)
Der Server kann jederzeit Log-Nachrichten an den Client senden. MCP nutzt hierfür ein standardisiertes System basierend auf RFC 5424.

*   **Technisch**: Eine `notifications/message` mit Level, optionalem Logger-Namen und Daten.
*   **Log-Levels**: MCP definiert acht Stufen: `debug`, `info`, `notice`, `warning`, `error`, `critical`, `alert`, `emergency`.
*   **Steuerung durch den Client**: Ein Client kann dem Server via `logging/setLevel` mitteilen, ab welcher Stufe er Logs empfangen möchte. Dies schont die Bandbreite bei SSE-Verbindungen.

### Progress (Fortschrittsanzeige)

Für lang laufende Tools kann der Server einen Fortschrittsbalken füttern.
*   **Voraussetzung**: Der Client sendet ein `progressToken` beim Aufruf mit.
*   **Update**: Der Server sendet regelmäßig `progress` (aktueller Wert) und `total` (Zielwert).

---

## 2. Cancellation: Den Stecker ziehen (Neu 2025-11)

Was passiert, wenn der Nutzer eine laufende Tool-Anfrage abbricht? In der MCP-Spezifikation (Stand 2025-11-25) gibt es hierfür den Mechanismus **Cancellation**.

### Das Prinzip
Sowohl der Client als auch der Server können eine laufende Anfrage stornieren. Dies geschieht über die Benachrichtigung `notifications/cancelled`.
*   **ID-basiert**: Die Benachrichtigung enthält die `requestId` der ursprünglichen Anfrage.
*   **Grund**: Optional kann ein Grund (z. B. "User clicked cancel") mitgegeben werden.
*   **Fire-and-Forget**: Die Stornierung erwartet keine Antwort. Der Empfänger sollte die Verarbeitung so schnell wie möglich einstellen.

### Warum ist das wichtig?
Ohne Cancellation würden lang laufende Prozesse auf dem Server wertvolle Ressourcen (CPU, DB-Verbindungen) verbrauchen, obwohl das Ergebnis vom Client gar nicht mehr erwartet wird.

---

## Implementierung in Go

Das Go-SDK macht uns die Stornierung sehr einfach: Es nutzt den Standard-Go-**`context.Context`**. Wenn ein Client eine Anfrage abbricht, wird der `ctx` des Tool-Handlers automatisch geschlossen (`ctx.Done()`).

```go
func handleLongTask(ctx context.Context, request *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
    // 1. Fortschritt melden (falls Token vorhanden)
    if token := request.Params.GetProgressToken(); token != nil {
        request.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
            Progress: 10, Total: 100, ProgressToken: token,
        })
    }

    // 2. Lang laufende Schleife mit Abbruch-Check
    for i := 0; i < 100; i++ {
        select {
        case <-ctx.Done():
            // WICHTIG: Hier Ressourcen aufräumen!
            log.Println("Anfrage vom Client abgebrochen.")
            return nil, nil, ctx.Err()
        default:
            // Simuliere Arbeit
            time.Sleep(100 * time.Millisecond)
        }
    }
    
    return &mcp.CallToolResult{
        Content: []mcp.Content{&mcp.TextContent{Text: "Fertig!"}},
    }, nil, nil
}
```

## Validierung mit `mcp-tester`

Der `mcp-tester` kann Fortschritt und Logs im Verbose-Modus (`-v`) anzeigen. In Test-Skripten können Sie zudem mit dem Befehl **`timeout`** eine Stornierung auf Client-Seite erzwingen:

```text
# Bricht den Tool-Call nach 500ms ab
timeout 500 call_tool progressTest 10
```

Dies ist ideal, um das robuste Verhalten Ihres Servers bei einem Abbruch durch den Benutzer zu validieren.

## Fazit


## Fazit

Mit Progress, Logging und **Cancellation** wird MCP zu einem robusten Protokoll für professionelle Anwendungen. Es stellt sicher, dass Serverressourcen effizient genutzt werden und der Benutzer stets über den Zustand seiner Anfragen informiert bleibt.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Grenzen & Limits →](12_grenzen_und_limitierungen.md)

---
*Copyright Michael Lechner - 2026-03-09*
