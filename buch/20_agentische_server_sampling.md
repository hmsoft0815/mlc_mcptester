# Kapitel 19: Agentische Server & Sampling – Wenn der Server die KI fragt

Bisher haben wir MCP so kennengelernt: Der Client (die KI) fragt den Server nach Tools oder Daten. Mit **Sampling** (SEP-1577, Stand 2025-11-25) wird dieses Modell erweitert: Der Server kann nun selbst das Modell (über den Client) zur Generierung einer Nachricht auffordern.

## 1. Was ist Sampling?

Sampling erlaubt es einem MCP-Server, eine "Probe" (ein Sample) der Modell-Intelligenz zu nehmen. Der Server sendet eine Liste von Nachrichten und Instruktionen an den Client, und der Client liefert eine vom LLM generierte Antwort zurück.

### Warum ist das revolutionär?
*   **Agentische Server**: Ein Server kann nun eigene Workflows steuern. Beispiel: Ein "Recherche-Server" erhält eine Anfrage, nutzt Sampling, um Unterfragen zu generieren, ruft seine eigenen Tools auf und nutzt erneut Sampling, um das Ergebnis zusammenzufassen.
*   **Abstraktion der KI**: Der Server muss keine eigenen API-Keys für OpenAI oder Anthropic besitzen. Er nutzt einfach die KI-Anbindung, die der Client bereits hat.
*   **Datenschutz**: Der Client behält die Kontrolle. Er kann Sampling-Anfragen des Servers blockieren, filtern oder dem Nutzer zur Freigabe vorlegen.

## 2. Die Methode `sampling/createMessage`

Dies ist der technische Kern. Der Server sendet einen Request an den Client mit folgenden Parametern:

*   **`messages`**: Der bisherige Chat-Verlauf (Rollen: `user`, `assistant`).
*   **`systemPrompt`**: Spezielle Anweisungen für diese Generierung (z. B. "Du bist ein Daten-Analyst").
*   **`modelPreferences`**: Hier sagt der Server, was ihm wichtig ist:
    *   `intelligencePriority`: Brauchen wir ein "schlaues" Modell (z. B. Claude 3.5 Opus)?
    *   `speedPriority`: Soll es besonders schnell gehen (z. B. Haiku)?
    *   `costPriority`: Darf es fast nichts kosten?
*   **`tools`**: Der Server kann dem Modell sogar Tools anbieten, die es innerhalb dieses Sampling-Schritts nutzen soll.

## 3. Der "Human-in-the-Loop"

Die MCP-Spezifikation legt großen Wert darauf, dass Sampling-Anfragen nicht unbemerkt im Hintergrund ablaufen. Ein guter MCP-Client sollte:
1.  Den Nutzer informieren: "Server X möchte das Modell nutzen, um eine Nachricht zu generieren."
2.  Dem Nutzer erlauben, den System-Prompt oder die Nachrichten zu bearbeiten.
3.  Die Kostenkontrolle beim Nutzer belassen.

## 4. Anwendungsbeispiel: Datenschutz & Lokale LLMs

Ein besonders spannendes Einsatzgebiet für Sampling ist der **Datenschutz**. Da der Client (und nicht der Server) entscheidet, welches Modell für die Sampling-Anfrage genutzt wird, ergeben sich neue Möglichkeiten für den Umgang mit sensitiven Daten.

### Das Szenario: "Privacy-First" Analyse
Stellen wir uns einen MCP-Server vor, der medizinische Berichte oder interne Finanzdaten analysiert. Anstatt diese Daten direkt an eine Cloud-KI zu senden, nutzt der Server Sampling.

1.  **Server**: Erkennt sensitive Daten und sendet eine `sampling/createMessage` Anfrage an den Client. Er gibt dabei als Hint an, dass er ein Modell mit hoher Privatsphäre bevorzugt (z. B. `modelPreferences: { hints: [{ name: "local" }] }`).
2.  **Client**: Der Nutzer hat seinen MCP-Client so konfiguriert, dass "lokale" Anfragen an eine Instanz von **Ollama** oder **LM Studio** auf dem eigenen Rechner geleitet werden.
3.  **Lokal**: Die Analyse findet komplett offline auf der Hardware des Nutzers statt.
4.  **Ergebnis**: Der Server erhält die Zusammenfassung zurück, ohne dass die Rohdaten jemals das lokale Netzwerk verlassen haben.

### Vorteile dieses Patterns:
*   **Compliance**: Unternehmen können MCP-Server nutzen, die komplexe Logik enthalten, während die eigentliche KI-Verarbeitung lokal (z. B. via Llama 3 oder Mistral) erfolgt.
*   **Kosten**: Für einfache Sampling-Aufgaben (z. B. Format-Konvertierung) können lokale Modelle genutzt werden, was API-Gebühren spart.
*   **Souveränität**: Der Nutzer entscheidet pro Client, welcher KI er für welche Aufgabe vertraut.

## 5. Implementierung (Konzept)

Im Go-SDK wird Sampling über den `ClientSession`-Handler auf Client-Seite und über `RequestSampling` auf Server-Seite abgewickelt.

### Beispiel: Server fragt Modell (Pseudo-Code)
```go
// Im Tool-Handler des Servers:
result, err := request.Session.RequestSampling(ctx, &mcp.CreateMessageRequest{
    Messages: []mcp.SamplingMessage{
        { Role: "user", Content: mcp.TextContent{ Text: "Fasse diese Daten zusammen: ..." } },
    },
    SystemPrompt: "Sei präzise und nutze Markdown-Tabellen.",
    ModelPreferences: mcp.ModelPreferences{ IntelligencePriority: 0.8 },
})

// Das 'result' enthält nun die Antwort des LLM, die der Server weiterverarbeiten kann.
```

## Fazit

Sampling macht MCP-Server "intelligent". Sie sind nicht mehr nur passive Werkzeugkästen, sondern können aktiv an der Problemlösung teilnehmen, indem sie die Rechenpower und das Verständnis des Modells gezielt für ihre Fachaufgaben einsetzen.

[← Inhaltsverzeichnis](README.md)

---
*Copyright Michael Lechner - 2026-03-09*
