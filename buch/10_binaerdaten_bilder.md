# Kapitel 10: Binärdaten – Wenn die KI Bilder schickt

Ein modernes LLM kann nicht nur lesen, es kann auch "sehen". MCP ermöglicht es Servern, nicht nur Text, sondern auch Binärdaten wie Bilder direkt in den Kontext des Modells zu übertragen. In diesem Kapitel lernen wir, wie das technisch funktioniert.

## Das ImageContent Objekt

Wenn ein Tool oder eine Ressource ein Bild zurückgeben möchte, nutzt es das `ImageContent` Objekt. Im Gegensatz zu Text wird hier das Bild **Base64-kodiert** übertragen.

**Beispiel für eine Antwort mit Bild:**
```json
{
  "content": [
    {
      "type": "image",
      "data": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJ...",
      "mimeType": "image/png"
    }
  ]
}
```

## Warum Base64?

Da MCP auf JSON-RPC basiert und JSON ein Textformat ist, können wir keine "rohen" Binärdaten senden. Base64 wandelt die Bits und Bytes des Bildes in eine Zeichenfolge um, die sicher innerhalb eines JSON-Strings transportiert werden kann.

---

## Implementierung in Go

Das Go SDK nimmt uns die Kodierung teilweise ab. Du musst lediglich das Byte-Array deines Bildes bereitstellen:

```go
func handleGetImage(ctx context.Context, request *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
    // Bild laden oder generieren
    imgBytes, _ := os.ReadFile("chart.png")
    
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            &mcp.ImageContent{
                Data:     imgBytes,
                MIMEType: "image/png",
            },
        },
    }, nil, nil
}
```

---

## Binärdaten testen mit `mcp-tester`

Das Testen von Binärdaten ist eine Herausforderung, da man Bilder nicht einfach im Terminal "sehen" kann. Unser `mcp-tester` löst dies auf zwei Arten:

1.  **Direktaufruf (`call`)**: Der Tester erkennt die Binärdaten und gibt eine Zusammenfassung aus:
    `Content 0 (Image): image/png data, size 12345 bytes`.
2.  **Skript-Modus (`test`)**: Wenn ein Skript ein Tool aufruft, das ein Bild zurückgibt, speichert der `mcp-tester` das Bild **automatisch als Datei** im aktuellen Verzeichnis (z. B. `get_image_0.png`).

Dadurch kannst du den Test laufen lassen und danach einfach die Datei öffnen, um das Ergebnis zu prüfen.

## Ausblick: Andere Medientypen

Obwohl Bilder (PNG, JPEG) am häufigsten genutzt werden, erlaubt das Protokoll theoretisch auch andere Typen wie `audio`. Die Logik bleibt gleich: Die Daten werden als Base64-Blob im `content`-Array übertragen und vom Client dem multimodalen Modell präsentiert.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Echtzeit-Feedback →](11_echtzeit_feedback_und_audio.md)

---
*Copyright Michael Lechner - 2026-02-28*
