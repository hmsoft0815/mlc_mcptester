# Kapitel 2: Minimal MCP (Stdio)

Der einfachste Weg, einen MCP-Server zu betreiben, ist über **Stdio (Standard Input/Output)**. In diesem Kapitel lernen wir, wie diese Kommunikation funktioniert und wie man einen minimalen Server startet.

## Warum Stdio?

Man könnte denken, dass MCP immer über das Internet (HTTP/Websockets) laufen muss. Aber für lokale Anwendungen (z. B. eine IDE, die auf Dateien zugreift) ist Stdio viel effizienter:
*   **Keine Ports**: Man muss keine TCP-Ports verwalten oder Firewalls konfigurieren.
*   **Sicherheit**: Nur der Prozess, der den Server gestartet hat, kann mit ihm reden.
*   **Lebenszyklus**: Wenn die Client-App (z. B. der `mcp-tester`) beendet wird, stirbt auch der MCP-Server-Prozess automatisch.

## SSE (Server-Sent Events): Der Web-Standard

Für Anwendungen, die über das Netzwerk kommunizieren (z. B. ein Server in der Cloud), nutzt MCP meist **SSE**. Hierbei wird ein HTTP-Server gestartet:
*   **Remote-Zugriff**: Der Server kann von überall erreichbar sein.
*   **Persistenz**: Läuft unabhängig vom Client-Lebenszyklus.
*   **Komplexität**: Erfordert Port-Management und Authentifizierung.

## Das Prinzip: JSON-RPC über Pipes oder HTTP

Die Kommunikation findet über einen Datenstrom statt. Der Client sendet ein JSON-Objekt an `stdin` des Servers, und der Server antwortet über `stdout`.

Ein typischer Handshake sieht so aus:

1.  **Client -> Server (`initialize`)**: "Hallo, ich bin Client X und unterstütze Version Y."
2.  **Server -> Client**: "Hallo! Ich bin Server Z und habe folgende Fähigkeiten (Tools, Resources)."
3.  **Client -> Server (`initialized`)**: "Okay, fangen wir an!"

## Ein minimaler Server in Go

Hier ist das absolute Minimum, um einen MCP-Server zu starten:

```go
package main

import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/modelcontextprotocol/go-sdk/server"
)

func main() {
    // 1. Server-Instanz erstellen
    s := server.NewServer("mini-server", "1.0.0")

    // 2. Über Stdio bereitstellen
    if err := server.ServeStdio(s); err != nil {
        panic(err)
    }
}
```

## Den Server testen

Mit unserem `mcp-tester` können wir diesen Server sofort validieren:

```bash
# Falls die Binary 'mini-server' heißt:
./mcp-tester list --command "./mini-server"
```

Selbst wenn der Server noch keine Tools hat, wird der `mcp-tester` den Handshake erfolgreich durchführen und eine (leere) Liste zurückgeben. Damit hast du deinen ersten funktionierenden MCP-Kanal aufgebaut!

## Jenseits von Stdio: Remote-Server (SSE)

Stdio ist perfekt für lokale Workflows, hat aber Grenzen. Wenn dein MCP-Server im Rechenzentrum laufen soll oder von vielen verschiedenen Clients gleichzeitig genutzt wird, kommt der **SSE-Transport (Server-Sent Events)** ins Spiel.

### Warum ein externer Server?
1.  **Zentralisierung**: Ein einziger MCP-Server kann die gesamte Belegschaft mit Tools versorgen (z. B. Zugriff auf das interne Wiki).
2.  **Ressourcen**: Rechenintensive Tools (z. B. Video-Rendering oder große Datenbank-Abfragen) können auf starker Hardware laufen, während der Client (z. B. ein Laptop) schlank bleibt.
3.  **Cloud-Native**: MCP-Server können als Container (Docker) in Kubernetes oder als Serverless-Funktionen betrieben werden.

Der `mcp-tester` ist bereits auf diese Welt vorbereitet. Statt eines Kommandos übergibst du einfach eine URL:
```bash
./mcp-tester list --url "https://mcp.meine-firma.de/sse"
```


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Tools →](04_tools_die_haende_des_modells.md)

---
*Copyright Michael Lechner - 2026-02-28*
