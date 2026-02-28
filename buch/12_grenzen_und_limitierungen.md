# Kapitel 12: Die Grenzen der KI – Modell-Limits und Kontext-Sprengung

MCP gibt uns die Macht, riesige Mengen an Daten und mächtige Tools an ein LLM zu koppeln. Doch am Ende des Tages ist das LLM immer noch der Flaschenhals. In diesem Kapitel besprechen wir die physischen und kognitiven Grenzen der Modelle.

## 1. Das Problem der Multimodalität (Modality Mismatch)

Nicht jedes Modell kann "sehen" oder "hören". 
*   **Szenario**: Ein MCP-Server sendet ein Bild via `ImageContent`.
*   **Modell-Limit**: Wenn du ein rein textbasiertes Modell (wie viele kleinere Open-Source Modelle oder ältere GPT-Versionen) nutzt, kann dieses mit dem Bild nichts anfangen.
*   **Folge**: Der Client muss das Bild entweder herausfiltern, in eine Textbeschreibung umwandeln oder die Anfrage wird mit einem Fehler abgelehnt.

**Entwickler-Tipp**: Dein Server sollte idealerweise alternative Text-Repräsentationen für Binärdaten anbieten, falls das Modell nicht multimodal ist.

---

## 2. Die Kontext-Sprengung (Context Window Overflow)

Jedes Modell hat ein "Kontext-Fenster" (z. B. 128k Token bei GPT-4 oder 200k bei Claude). Jede Information, die über MCP reinkommt, verbraucht Platz in diesem Fenster.

### Die Gefahr großer Ressourcen
Wenn ein Tool oder eine Ressource eine 5 MB große Logdatei oder ein riesiges JSON-Objekt zurückgibt, passiert Folgendes:
1.  **Token-Inflation**: 5 MB Text entsprechen Millionen von Token.
2.  **Überlauf**: Das Kontext-Fenster ist sofort voll.
3.  **Gedächtnisverlust**: Das Modell beginnt, den Anfang der Konversation zu "vergessen". Oft betrifft das die **System-Anweisungen** (Prompts) aus Kapitel 6. Das Modell verliert seine Persona oder ignoriert Sicherheitsregeln.

---

## 3. Kosten und Latenz

Vergiss nicht: Bei Cloud-Modellen zahlst du pro Token. 
*   Ein hochauflösendes Bild oder eine massive Datenbank-Antwort über MCP zu schicken, kann eine einzelne Chat-Anfrage sehr teuer machen.
*   Zudem steigt die Latenz (Time-to-first-token) massiv an, wenn das Modell erst Hunderttausende Token verarbeiten muss, bevor es antworten kann.

---

## 4. Strategien für robuste MCP-Systeme

Um diese Limits zu umgehen, sollten Server-Entwickler folgende Techniken nutzen:

*   **Pagination**: Schicke niemals alle Daten auf einmal. Nutze Listen mit `nextCursor`.
*   **Summarization**: Anstatt 1.000 Zeilen Log zu schicken, lass den Server die Logs voranalysieren und nur eine Zusammenfassung senden.
*   **Sampling-Präferenz**: Wähle das Modell passend zum Server. Ein Server, der viele Bilder generiert, braucht zwingend ein multimodales Frontend.

## Validierung mit `mcp-tester`

Nutze den `mcp-tester`, um die Größe deiner Antworten zu überwachen. Wenn du im Skript-Modus Bilder speicherst, achte auf deren Dateigröße. Wenn eine Ressourcen-Abfrage im Tester mehrere Sekunden dauert, wird sie im echten LLM-Chat wahrscheinlich zu einem Timeout oder zu massiver Latenz führen.

## Fazit

Ein MCP-Entwickler muss immer die **Token-Ökonomie** im Blick behalten. Nur weil du eine ganze Festplatte als Ressource freigeben *kannst*, heißt das nicht, dass das LLM sie lesen *sollte*. Weniger ist oft mehr – Präzision schlägt Masse.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Artifact-Pattern →](13_das_artifact_pattern.md)

---
*Copyright Michael Lechner - 2026-02-28*
