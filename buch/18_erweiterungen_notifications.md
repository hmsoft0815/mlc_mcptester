# Kapitel 18: Erweiterungen – Notifications (Push-Nachrichten)

Während das Standard-MCP auf einem Request-Response-Modell basiert, erlauben **Notifications**, dass eine Seite (Server oder Client) Informationen an die Gegenseite sendet, ohne auf eine Antwort zu warten. Dies ist das "Push"-Prinzip von MCP.

## 1. Was sind Notifications?

In MCP sind Notifications asynchrone Nachrichten. Sie haben keine `request_id`, da keine Antwort erwartet oder möglich ist. Sie dienen dazu, den Zustand zu synchronisieren oder über Ereignisse zu informieren.

### Standard-Notification-Typen (Server -> Client)

-   `notifications/message`: Sendet Log-Daten an den Client.
-   `notifications/progress`: Informiert über den Fortschritt einer laufenden Operation.
-   `notifications/resources/list_changed`: Signalisiert, dass sich die Liste der verfügbaren Ressourcen geändert hat.
-   `notifications/prompts/list_changed`: Signalisiert Änderungen an den Prompts.

---

## 2. Implementierung im Server (Go)

Ein Server sendet Notifications über das `Session`-Objekt, das im Tool-Request enthalten ist.

### Beispiel: Fortschritt melden
```go
func handleLongRun(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
    token := req.Params.GetProgressToken()
    if token != nil {
        req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
            Progress:      25.0,
            Total:         100.0,
            Message:       "Viertel geschafft...",
            ProgressToken: token,
        })
    }
    return mcp.NewToolResultText("Fertig!"), nil, nil
}
```

---

## 3. Implementierung im Client (unser mcp-tester)

Damit ein Client Notifications verarbeiten kann, muss er beim Start der Session entsprechende **Handler** registrieren. Unser `mcp-tester` nutzt dafür die `ClientOptions`.

Hier ein Auszug aus `transport.go`:

```go
func getClient(verbose bool) *mcp.Client {
    opts := &mcp.ClientOptions{
        // Handler für Log-Nachrichten
        LoggingMessageHandler: func(ctx context.Context, req *mcp.LoggingMessageRequest) {
            fmt.Printf("[SERVER LOG] [%s] %s: %v\n", 
                req.Params.Level, req.Params.Logger, req.Params.Data)
        },
        // Handler für Fortschritts-Updates
        ProgressNotificationHandler: func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
            fmt.Printf("[PROGRESS] Token: %v, %.2f%%: %s\n", 
                req.Params.ProgressToken, (req.Params.Progress/req.Params.Total)*100, req.Params.Message)
        },
    }
    // ... Client erstellen ...
}
```

### Warum ist das wichtig?
Ohne diese Handler würde der Client die eingehenden JSON-RPC Notifications einfach ignorieren. Durch die Handler können wir dem Benutzer Echtzeit-Feedback geben, während er auf das Ergebnis eines langwierigen Tool-Aufrufs wartet.

---

## 4. Erkennung der Unterstützung

Ob ein Server oder Client Notifications unterstützt, wird während des `initialize`-Schritts über die **Capabilities** ausgehandelt:

-   **Server-Capability**: `logging: {}` bedeutet, der Server wird Logs senden.
-   **Client-Capability**: `roots: { listChanged: true }` bedeutet, der Client möchte über Änderungen an den Workspace-Roots informiert werden.

Im `mcp-tester` kannst du diese Aushandlung mit dem `inspect` Befehl und dem Verbose-Flag (`-v`) beobachten.

---

## 5. Request Cancellation: Aufgaben abbrechen

Ein besonders mächtiger Aspekt von Notifications ist die **Request Cancellation**. Sie erlaubt es dem Client, eine bereits gesendete, aber noch nicht abgeschlossene Anfrage (z.B. einen langen Tool-Aufruf) abzubrechen.

### Der Mechanismus: `$/cancelRequest`

Wenn ein Client eine Anfrage abbrechen möchte, sendet er eine Notification vom Typ `$/cancelRequest` mit der ursprünglichen `requestId`. 

### Implementierung in Go (Server-Seite)

Unsere Go-Bibliothek macht die Handhabung von Abbrüchen extrem einfach, da sie direkt auf dem Standard-Go-Pattern `context.Context` basiert. Wenn ein Client einen Abbruch sendet, wird der `ctx`, der an den Tool-Handler übergeben wurde, automatisch "cancelled".

**Beispiel für sauberes Stornieren:**
```go
func handleCalculations(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
    for i := 0; i < 100; i++ {
        // WICHTIG: Prüfen, ob der Chat/Agent abgebrochen hat
        select {
        case <-ctx.Done():
            // Cleanup und schnelles Beenden
            return nil, nil, ctx.Err()
        default:
            // Weiterarbeiten...
            doHeavyWork()
        }
    }
    return mcp.NewToolResultText("Fertig"), nil, nil
}
```

### Warum ist das wichtig?
Ohne Cancellation würden lang laufende Tools wertvolle Server-Ressourcen (CPU, Memory) verbrauchen, selbst wenn der Agent die Antwort gar nicht mehr benötigt (z.B. weil der User den Chat geschlossen oder eine andere Frage gestellt hat). In einer skalierten Umgebung ist dies essentiell für die Performance und Kosteneffizienz.

---

## Fazit

Notifications machen MCP lebendig. Vom einfachen Logging über Fortschrittsbalken bis hin zur harten Stornierung von Aufgaben (`cancelRequest`) – sie verwandeln eine statische API in eine interaktive Schnittstelle, die dem Benutzer (und dem Modell) zeigt, was "hinter den Kulissen" passiert.

[← Anhang A: SSE und HTTP/2](17_anhang_sse_http2.md) | [Inhaltsverzeichnis](README.md)

---
*Copyright Michael Lechner - 2026-03-01*
