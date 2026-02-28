# Kapitel 3: Tools – Die Hände des Modells

Während Resources (Daten) dem Modell "Wissen" geben, sind **Tools** die Werkzeuge, mit denen das Modell aktiv werden kann. In diesem Kapitel lernen wir, wie man Tools definiert, implementiert und testet.

## Was ist ein MCP Tool?

Ein Tool ist eine ausführbare Funktion, die der Server dem Client (und damit dem LLM) zur Verfügung stellt. Jedes Tool besteht aus drei Teilen:
1.  **Name**: Eine eindeutige ID (z. B. `calculate_sum`).
2.  **Beschreibung**: Ein Text, der dem LLM erklärt, *wann* und *warum* es dieses Tool nutzen sollte.
3.  **Input Schema**: Ein JSON-Schema, das definiert, welche Argumente das Tool erwartet.

## Implementierung in Go

Das offizielle SDK macht es uns sehr einfach. Wir nutzen die Funktion `s.AddTool`. Hier ist ein Beispiel für ein Tool, das zwei Zahlen addiert:

```go
s.AddTool(mcp.NewTool("add",
    mcp.WithDescription("Addiert zwei ganze Zahlen"),
    mcp.WithArgument("a", mcp.ArgumentDescription("Erste Zahl"), mcp.RequiredArgument()),
    mcp.WithArgument("b", mcp.ArgumentDescription("Zweite Zahl"), mcp.RequiredArgument()),
), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Argumente sicher auslesen
    a, ok1 := request.Params.Arguments["a"].(float64)
    b, ok2 := request.Params.Arguments["b"].(float64)
    
    if !ok1 || !ok2 {
        return mcp.NewToolResultError("Ungültige Argumente. 'a' und 'b' müssen Zahlen sein."), nil
    }
    
    sum := int(a) + int(b)
    return mcp.NewToolResultText(fmt.Sprintf("Ergebnis: %d", sum)), nil
})
```

> [!TIP]
> Die Nutzung von `mcp.NewToolResultText` ist der empfohlene Weg, um einfache Textantworten zurückzugeben.

## Fehlerbehandlung: Best Practices

Ein robuster MCP-Server sollte niemals abstürzen. Wenn etwas schiefgeht, nutze `mcp.NewToolResultError`:

1.  **Validierung**: Prüfe alle Eingaben des LLMs gründlich.
2.  **Sprechende Fehler**: Sag dem Modell genau, *was* fehlt oder falsch ist. Das Modell kann dann oft den Fehler korrigieren und es erneut versuchen.
3.  **IsError Flag**: Setze das `IsError` Flag im Result (wird von `NewToolResultError` automatisch gemacht), damit das Modell weiß, dass die Aktion fehlgeschlagen ist.

```go
if id == "" {
    return mcp.NewToolResultError("Die 'id' darf nicht leer sein."), nil
}
```

## Tools testen mit `mcp-tester`

Ein großer Teil der MCP-Entwicklung besteht darin, sicherzustellen, dass die Tools robust auf verschiedene Eingaben reagieren.

### Manueller Aufruf
Du kannst ein Tool direkt über die Kommandozeile testen:
```bash
./bin/mcp-tester call add --profile local --args '{"a": 10, "b": 5}'
```

### Automatisierte Tests (Scripting)
Viel mächtiger ist die Skript-Engine unseres Testers. In einer Datei `test.mcp` kannst du Abläufe definieren:

```text
# Teste die Addition
call_tool add 10 20
# Überprüfe das Ergebnis
assert_contains "Ergebnis: 30"
```

Der `mcp-tester` übernimmt hierbei die **Typ-Konvertierung**. Obwohl im Skript nur Text steht, erkennt der Tester anhand des Server-Schemas, dass `10` und `20` als Zahlen gesendet werden müssen.

## Best Practices für Tools

1.  **Gute Beschreibungen**: Das LLM "liest" deine Beschreibung, um zu entscheiden, ob es das Tool aufruft. Sei präzise!
2.  **Fehlerbehandlung**: Wenn eine Eingabe ungültig ist, sende eine verständliche Fehlermeldung im `Content` zurück (mit `IsError: true`), anstatt das ganze Programm abstürzen zu lassen.
3.  **Sicherheit**: Vertraue niemals den Eingaben des Modells. Validiere Pfade, Berechtigungen und Wertebereiche gründlich.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Resources →](05_resources_das_gedaechtnis.md)

---
*Copyright Michael Lechner - 2026-02-28*
