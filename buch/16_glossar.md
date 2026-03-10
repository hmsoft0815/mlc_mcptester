# Kapitel 16: Glossar – Die Sprache von MCP

Um in der Welt des Model Context Protocol sicher zu navigieren, hilft ein klares Verständnis der Fachbegriffe. Hier sind die wichtigsten Termini zusammengefasst:

### MCP-Host (Host)
Die Anwendung, die das LLM steuert und die Verbindung zu den MCP-Servern verwaltet. 
*Beispiele: SOTA-Desktop-Clients, IDEs (Cursor, VS Code), mcp-tester.*

### MCP-Client (Client)
Die Komponente innerhalb des Hosts, die die eigentliche Kommunikation mit dem Server übernimmt (via Stdio oder SSE). Oft werden die Begriffe Host und Client synonym verwendet.

### MCP-Server (Server)
Ein eigenständiger Prozess oder Webservice, der Tools, Resources und Prompts bereitstellt. Er implementiert das MCP-Protokoll.

### Transport (Transport Layer)
Die technische Schicht, über die Client und Server Daten austauschen.
*   **Stdio**: Kommunikation über Standard-Input/Output (lokale Pipes).
*   **SSE (Server-Sent Events)**: Einseitig gerichtetes Streaming vom Server zum Client über HTTP, ergänzt durch HTTP POST für die Rückrichtung.

### Tool (Werkzeug)
Eine ausführbare Funktion des Servers mit einem definierten Eingabe-Schema. Tools erlauben dem LLM, Aktionen in der Außenwelt auszuführen.

### Resource (Ressource)
Datenquellen (Texte, Bilder, Dateien, Blobs), die der Server dem Modell zur Verfügung stellt. Sie werden über URIs identifiziert.

### Prompt (Anweisung / Vorlage)
Vordefinierte Textbausteine oder System-Anweisungen, die vom Server geliefert werden, um das Verhalten des Modells zu steuern.

### Elicitation (Abfrage)
Ein Mechanismus, bei dem der Server den Benutzer (via Client) um Informationen bittet. Es gibt den **Form-Modus** (für strukturierte Daten) und den **URL-Modus** (für OAuth/Logins im Browser).

### Sampling (Generierung)
Ein Verfahren, bei dem der Server den Client auffordert, eine Antwort vom LLM generieren zu lassen. Dies ermöglicht "agentische" Server, die die KI für ihre eigenen Workflows nutzen.

### Task (Aufgabe)
Eine asynchrone, lang laufende Operation. Tasks verwandeln synchrone Tool-Aufrufe in Zustandsmaschinen (working, completed, failed), um Timeouts zu vermeiden.

### Cancellation (Abbruch)
Die Möglichkeit für Client oder Server, eine laufende Anfrage vorzeitig zu beenden (`notifications/cancelled`), um Ressourcen zu sparen.

### Ping
Ein einfacher Health-Check Request, um zu prüfen, ob die Verbindung zwischen Client und Server noch aktiv und reaktionsfähig ist.

### Icon
Visuelle Metadaten (URLs oder Base64-Daten), die es Clients ermöglichen, Tools, Ressourcen und Prompts mit Grafiken darzustellen.

### Logging Level
Stufen der Protokollierung basierend auf RFC 5424 (debug, info, warning, error, etc.). Clients können die Verbose-Stufe über `logging/setLevel` steuern.

### URI (Uniform Resource Identifier)
Eine eindeutige Kennung für Ressourcen (z. B. `file:///logs/today.txt`).

### Context Window (Kontext-Fenster)
Die maximale Menge an Informationen (Token), die ein LLM gleichzeitig verarbeiten kann. MCP-Informationen belegen Platz in diesem Fenster.

### RPC (Remote Procedure Call)
Ein technisches Konzept, bei dem eine Funktion in einem anderen Prozess aufgerufen wird. MCP nutzt JSON-RPC 2.0 für diesen Austausch.

### Handshake (Initialisierung)
Der erste Datenaustausch, bei dem Versionen und Fähigkeiten (Capabilities) ausgehandelt werden.


[← Zurück zum Inhaltsverzeichnis](README.md)

---
*Copyright Michael Lechner - 2026-03-09*
