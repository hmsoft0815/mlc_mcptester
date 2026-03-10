# Kapitel 18: Tasks – Lang laufende Aufgaben & Asynchronität (Experimental)

In den vorherigen Kapiteln haben wir gelernt, wie Tools direkt auf Anfragen antworten. Doch was passiert, wenn eine Aufgabe Minuten oder Stunden dauert (z. B. das Training eines Modells oder eine komplexe Datenanalyse)? In der MCP-Spezifikation vom 25.11.2025 wurde hierfür der **Task-Mechanismus** (SEP-1686) eingeführt.

## 1. Das Problem mit synchronen Aufrufen

Standard-Tool-Aufrufe sind synchron: Der Client wartet, bis der Server fertig ist. Bei Netzwerkverbindungen (SSE) führt dies oft zu Timeouts. Zudem blockiert ein wartendes LLM den gesamten Chat-Fluss.

## 2. Das Prinzip der Tasks (Aufgaben)

Tasks verwandeln einen synchronen Aufruf in eine **asynchrone Zustandsmaschine**.

1.  **Anfrage**: Der Client ruft ein Tool auf.
2.  **Sofortige Antwort**: Der Server antwortet nicht mit dem Ergebnis, sondern mit einer `taskId`.
3.  **Hintergrund**: Der Server arbeitet im Hintergrund weiter.
4.  **Polling/Status**: Der Client fragt regelmäßig den Status ab (`tasks/get`) oder wartet auf eine Benachrichtigung (`notifications/tasks/status_changed`).
5.  **Ergebnis**: Wenn die Aufgabe den Status `completed` erreicht, holt der Client das finale Ergebnis ab (`tasks/result`).

### Der Lebenszyklus einer Task
*   **`working`**: Die Aufgabe wird bearbeitet.
*   **`input_required`**: Der Server braucht weitere Informationen vom Nutzer (Elicitation).
*   **`completed`**: Erfolgreich beendet. Das Ergebnis liegt bereit.
*   **`failed`**: Ein Fehler ist aufgetreten.
*   **`cancelled`**: Die Aufgabe wurde abgebrochen.

## 3. Vorteile für die KI-Interaktion

Ein besonderes Feature von Tasks ist die **Immediate Response**. Während die eigentliche Aufgabe (z. B. "Analysiere 10.000 PDF-Dateien") im Hintergrund läuft, kann der Server dem LLM sofort einen Text zurückgeben: *"Ich habe die Analyse gestartet. Das wird etwa 5 Minuten dauern. Möchtest du in der Zwischenzeit etwas anderes tun?"*

Das LLM erhält also eine "Platzhalter-Antwort" und bleibt für den Nutzer ansprechbar, während die schwere Arbeit asynchron erledigt wird.

## 4. Implementierung (Konzept)

Da Tasks noch experimentell sind, werden sie oft über spezielle Metadaten (`_meta`) in den Tool-Aufrufen signalisiert.

### Polling-Beispiel (Client-Sicht)
1.  **Call Tool**: `tools/call { name: "heavy_job" }`
2.  **Server Result**: `{ _meta: { taskId: "job_42" }, content: [{ type: "text", text: "Job gestartet..." }] }`
3.  **Poll Status**: `tasks/get { taskId: "job_42" }` -> Antwort: `{ status: "working", progress: 45 }`
4.  **Get Result**: `tasks/result { taskId: "job_42" }` -> Antwort: `{ content: [...] }`

## 5. Cancellation & TTL

Da Tasks Ressourcen auf dem Server verbrauchen, haben sie eine **TTL (Time To Live)**. Nach Ablauf dieser Zeit (z. B. 24 Stunden) löscht der Server die Task und das Ergebnis automatisch. Zudem kann eine Task jederzeit via `tasks/cancel` abgebrochen werden.

## Fazit

Tasks machen MCP "Cloud-Ready". Sie ermöglichen komplexe Workflows, Batch-Processing und eine flüssige Benutzererfahrung, selbst wenn die zugrunde liegenden Prozesse viel Zeit in Anspruch nehmen.

[← Inhaltsverzeichnis](README.md)

---
*Copyright Michael Lechner - 2026-03-09*
