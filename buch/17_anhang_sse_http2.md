# Anhang A: Technische Deep-Dive – SSE und HTTP/2

Während die lokale Kommunikation über Stdio sehr simpel ist, erfordert die Anbindung von Remote-Servern ein tieferes Verständnis der Web-Technologien. In diesem Anhang erklären wir die Funktionsweise des SSE-Transports und die Rolle von HTTP/2.

## Was ist SSE (Server-Sent Events)?

**Server-Sent Events (SSE)** ist ein Standard, der es einem Webserver ermöglicht, Daten in Echtzeit an einen Browser (oder Client) zu "pushen", ohne dass dieser ständig nachfragen muss (Polling).

### Die Funktionsweise in MCP:
MCP nutzt SSE für die Richtung **Server → Client**. 
1.  Der Client öffnet eine persistente HTTP-Verbindung zum Endpunkt `/sse`.
2.  Der Server hält diese Verbindung offen und sendet Nachrichten als Stream (Event-Stream).
3.  Für die Gegenrichtung (**Client → Server**) nutzt MCP klassische **HTTP POST** Anfragen.

**Vorteil**: SSE ist wesentlich leichtgewichtiger als WebSockets, da es auf einfachem HTTP basiert und keine komplexen Handshakes für die bidirektionale Verbindung benötigt.

---

## Die Bedeutung von HTTP/2

Wenn MCP über das Netzwerk (SSE) betrieben wird, ist die Nutzung von **HTTP/2** (oder höher) dringend empfohlen.

### Warum nicht HTTP/1.1?
Das alte HTTP/1.1 Protokoll hat eine strikte Grenze für gleichzeitige Verbindungen zum selben Host (meist nur 6-8 Verbindungen). Da jeder MCP-Server eine dauerhafte SSE-Leitung benötigt, wäre diese Grenze bei einer Multi-Server-Architektur (siehe Kapitel 7) sehr schnell erreicht. Neue Verbindungen würden blockieren (Head-of-Line Blocking).

### Die Vorteile von HTTP/2 für MCP:
1.  **Multiplexing**: Über eine einzige TCP-Verbindung können beliebig viele SSE-Streams gleichzeitig laufen. Du kannst also 50 MCP-Server über einen einzigen Proxy anbinden, ohne dass diese sich gegenseitig blockieren.
2.  **Server Push & Header Compression**: Reduziert die Latenz und den Overhead bei den vielen kleinen JSON-RPC Nachrichten.
3.  **Stabilität**: HTTP/2 ist besser darin, Verbindungen über längere Zeit offen zu halten, was für den "Always-On" Charakter von MCP-Tools ideal ist.

---

## Fazit für Entwickler

Wenn du einen MCP-Server für den Remote-Einsatz planst:
*   Nutze **SSE** für die Benachrichtigungen und Tool-Antworten.
*   Stelle sicher, dass dein Load-Balancer oder Reverse-Proxy (z. B. Nginx, Caddy) **HTTP/2** zwingend unterstützt.
*   Achte darauf, dass Timeouts für HTTP-Verbindungen hoch genug eingestellt sind, damit die SSE-Leitung nicht mitten im Chat unterbrochen wird.
