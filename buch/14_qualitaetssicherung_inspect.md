# Kapitel 14: Qualitätssicherung – Der Server-Inspector

Einen MCP-Server zu schreiben ist einfach, aber einen *guten* MCP-Server zu schreiben, ist eine Kunst. In diesem Kapitel lernen wir, wie wir mit dem `inspect` Kommando die Qualität unserer Implementierung messen und verbessern können.

## Warum ist Qualität bei MCP so wichtig?

Ein LLM ist auf klare Anweisungen angewiesen. Wenn dein Server unvollständige Metadaten liefert, wird die KI Fehler machen:
*   **Fehlende Beschreibungen**: Das Modell weiß nicht, ob es das Tool `fetch_data` oder `get_info` aufrufen soll.
*   **Fehlende Output-Schemata**: Das Modell bekommt nur einen Text-Klumpen zurück und muss die Daten mühsam raten (siehe Kapitel 9).
*   **Fehlende Prompts**: Das Modell hat keine Identität (Persona) und kennt keine Geschäftsregeln für deine Tools (siehe Kapitel 6).

---

## Den Server begutachten

Mit dem Befehl `inspect` führt der `mcp-tester` eine Tiefenanalyse des Servers durch:

```bash
./bin/mcp-tester inspect --profile my-server
```

### Die Bewertungskriterien (Quality Score)

Der Tester vergibt einen Score von 0 bis 100 basierend auf folgenden Punkten:

1.  **Vollständigkeit der Features (-20 Pkt bei fehlenden Prompts)**: Ein professioneller Server sollte Prompts anbieten, um dem LLM zu erklären, wie die Tools im Verbund genutzt werden sollen.
2.  **Dokumentation (-5 Pkt pro fehlende Tool-Beschreibung)**: Jedes Tool muss dem Modell seinen Zweck erklären.
3.  **Strukturierte Daten (-2 Pkt pro fehlendes Output-Schema)**: Tools sollten nicht nur Text, sondern strukturierte JSON-Objekte zurückgeben. Dies erhöht die Zuverlässigkeit bei Folge-Aktionen (Turn-based Loops).

---

## Der Weg zum "Perfekten Server" (100/100)

Um die volle Punktzahl zu erreichen und die bestmögliche KI-Integration zu garantieren, folge dieser Checkliste:

*   [ ] **Handshake prüfen**: Unterstützt der Server die aktuellste Protokoll-Version?
*   [ ] **Tools beschreiben**: Ist jede Funktion so erklärt, dass auch ein fachfremdes LLM sie versteht?
*   [ ] **Input & Output Typisieren**: Sind alle Parameter und Rückgabewerte über JSON-Schemata definiert?
*   [ ] **Personas definieren**: Bietet der Server mindestens einen Prompt an, der den System-Kontext für das LLM setzt?
*   [ ] **Logging aktivieren**: Sendet der Server Status-Updates während langer Aufgaben?

---

## Unit-Testing von Tool-Handlern

Neben dem Blick von außen (Integrationstests) sollten Tool-Handler auch intern durch klassische Go-Unit-Tests abgesichert werden. Da Tool-Handler in MCP einfache Funktionen sind, lassen sie sich hervorragend isoliert testen.

### Beispiel: Unit-Test für das `add` Tool

Angenommen, wir haben einen Handler `handleAdd`. So sieht der dazugehörige Test aus:

```go
func TestHandleAdd(t *testing.T) {
    ctx := context.Background()
    
    // Test-Daten (Eingabe-Argumente)
    args := map[string]any{
        "a": 10.0, // JSON-Zahlen kommen oft als float64 an
        "b": 20.0,
    }

    // Handler direkt aufrufen (ohne MCP-Server-Infrastruktur)
    result, structured, err := handleAdd(ctx, nil, args)

    // Validierung
    if err != nil {
        t.Errorf("Unerwarteter Fehler: %v", err)
    }

    // Text-Antwort prüfen
    expectedText := "Result: 30"
    if result.Content[0].(*mcp.TextContent).Text != expectedText {
        t.Errorf("Erwartet %q, got %q", expectedText, result.Content[0])
    }

    // Strukturierte Antwort prüfen
    resMap := structured.(map[string]any)
    if resMap["sum"] != 30 {
        t.Errorf("Mathematisch falsches Ergebnis im structuredContent: %v", resMap["sum"])
    }
}
```

### Warum Unit-Tests?
1.  **Geschwindigkeit**: Sie laufen in Millisekunden, ohne dass ein Server-Prozess gestartet werden muss.
2.  **Edge-Cases**: Du kannst extrem einfach Fehlerzustände (z. B. Division durch Null, ungültige JSON-Typen) provozieren.
3.  **Refactoring**: Wenn du den internen Code deines Tools änderst, garantieren Unit-Tests, dass die Logik konsistent bleibt.

**Best Practice**: Nutze Unit-Tests für die **Logik** und den `mcp-tester` für das **Verhalten** und die **Schnittstellen-Qualität**.

## Fazit
...

Qualitätssicherung bei MCP bedeutet, die Reibungsverluste zwischen Server und KI zu minimieren. Ein Server mit 100/100 Punkten im `inspect`-Check wird in der Praxis deutlich seltener zu Fehlinterpretationen oder Fehlern im Modell-Verlauf führen. Nutze dieses Tool regelmäßig während der Entwicklung, um deine Implementierung zu perfektionieren.

[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Glossar →](15_glossar.md)

---
*Copyright Michael Lechner - 2026-02-28*
