# Prompts im Model Context Protocol (MCP)

Dieses Dokument erklärt, was Prompts im Kontext von MCP sind, warum sie nützlich sind und wie sie mit dem `mcp-tester` validiert werden können.

## Was ist ein MCP Prompt?

Ein **Prompt** im Model Context Protocol ist ein serverseitig definiertes **Template für Anweisungen**, die an ein Large Language Model (LLM) gesendet werden. 

Im Gegensatz zu **Tools** (die das LLM *selbst* aufrufen kann, um Aktionen auszuführen) sind **Prompts** Vorlagen, die von der Client-Applikation (oder dem Benutzer) abgerufen werden, um dem LLM einen spezifischen Kontext oder eine bestimmte Aufgabe zuzuweisen.

### Warum braucht man das? (Der Nutzen)

1.  **Zentralisierung**: Anweisungen ("Du bist ein Experte für...") müssen nicht in jeder Client-App neu geschrieben werden, sondern liegen zentral auf dem MCP-Server.
2.  **Dynamik**: Prompts können Argumente (Variablen) enthalten, die zur Laufzeit gefüllt werden (z. B. ein zu prüfender Code-Schnipsel).
3.  **Struktur**: Ein Prompt kann aus mehreren Nachrichten bestehen (z. B. eine System-Nachricht, eine User-Nachricht und eine Beispiel-Antwort des Assistenten).

---

## Beispiel für einen Prompt

Stellen wir uns einen MCP-Server für "Code-Qualität" vor. Er könnte einen Prompt namens `review` anbieten:

*   **Name**: `review`
*   **Argumente**: `code` (Pflicht)
*   **Logik**: Der Server nimmt den Text aus `code` und bettet ihn in eine komplexe Anweisung ein.

**Die Ausgabe des Servers wäre dann:**
```json
{
  "messages": [
    {
      "role": "user",
      "content": {
        "type": "text",
        "text": "Bitte analysiere diesen Code auf Sicherheitslücken: [Hier steht der übergebene Code]"
      }
    }
  ]
}
```

---

## Prompts testen mit `mcp-tester`

Unser Tool bietet zwei Befehle, um die Prompt-Funktionalität eines Servers zu prüfen.

### 1. Verfügbare Prompts auflisten
Zuerst möchten wir wissen, welche Vorlagen der Server überhaupt anbietet und welche Argumente sie erwarten.

```bash
./bin/mcp-tester prompts list --profile local
```

**Was du siehst:**
*   Den Namen des Prompts (z. B. `review`).
*   Eine Beschreibung, was der Prompt tut.
*   Eine Liste der Argumente und ob diese erforderlich sind.

### 2. Einen Prompt generieren (Get Prompt)
Um zu prüfen, ob der Server die Variablen korrekt in das Template einsetzt, rufen wir den Prompt ab.

```bash
./bin/mcp-tester prompts get review --args '{"code": "func main() { panic(1) }"}' --profile local
```

**Was du prüfen solltest:**
*   Erscheint der übergebene Code an der richtigen Stelle in der Nachricht?
*   Sind die Rollen (`user`, `assistant`) korrekt zugewiesen?
*   Gibt der Server einen Fehler zurück, wenn ein Pflicht-Argument fehlt?

---

## Zusammenfassung für Entwickler

Wenn du einen MCP-Server entwickelst, helfen dir Prompts dabei, dem LLM zu helfen, deine Daten (Resources) oder Funktionen (Tools) besser zu nutzen. Mit dem `mcp-tester` stellst du sicher, dass deine Templates sauber gerendert werden und die Kommunikation zwischen Server und Modell reibungslos funktioniert.


---
*Copyright Michael Lechner - 2026-02-28*
