# Kapitel 15: Glossar – Die Sprache von MCP

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
Datenquellen (Texte, Bilder, Dateien), die der Server dem Modell zur Verfügung stellt. Sie werden über URIs identifiziert.

### Prompt (Anweisung / Vorlage)
Vordefinierte Textbausteine oder System-Anweisungen, die vom Server geliefert werden, um das Verhalten des Modells zu steuern oder Workflows zu bootstrappen.

### Context Window (Kontext-Fenster)
Die maximale Menge an Informationen (Token), die ein LLM gleichzeitig verarbeiten kann. MCP-Informationen (Server-Definitionen und Tool-Antworten) belegen Platz in diesem Fenster.

### RPC (Remote Procedure Call)
Ein technisches Konzept, bei dem ein Computerprogramm eine Funktion in einem anderen Adressraum (häufig auf einem anderen Computer oder in einem anderen Prozess) aufruft, als wäre es ein lokaler Funktionsaufruf. MCP nutzt dieses Prinzip, um Tools auf dem Server für das LLM verfügbar zu machen.

### JSON-RPC
Das zugrunde liegende Nachrichtenformat von MCP. Es ist eine spezifische Implementierung von RPC, die JSON zur Kodierung der Daten nutzt. MCP verwendet JSON-RPC 2.0 für den Austausch von Anfragen, Antworten und Benachrichtigungen.

### Handshake (Initialisierung)
Der erste Datenaustausch zwischen Client und Server, bei dem Versionen und Fähigkeiten (Capabilities) ausgehandelt werden.


[← Zurück zum Inhaltsverzeichnis](README.md)

---
*Copyright Michael Lechner - 2026-02-28*
