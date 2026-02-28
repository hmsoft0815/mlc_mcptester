# Kapitel 6: Prompts – Die Anweisungen und der System-Kontext

*Hinweis zur Terminologie: Der englische Begriff **Prompt** wird im Deutschen oft treffend mit **Anweisung** oder **Aufforderung** übersetzt.*

Ein **Prompt** im MCP ist weit mehr als nur ein kurzer Textbaustein. Er ist die primäre Methode, um dem LLM eine **Identität**, Regeln und den notwendigen **Kontext** zu geben. Technisch gesehen dient ein Prompt oft dazu, den **System-Prompt** des Modells zu definieren oder zu erweitern.

## Strategische Prompts: Die "Bedienungsanleitung" für Tools

Das LLM kennt zwar die technischen Definitionen deiner Tools (Name, Parameter), aber es kennt nicht deine **Geschäftsregeln**. Hier kommen strategische Prompts ins Spiel. Sie geben dem Modell Anweisungen, *wann* und *wie* ein Tool genutzt werden muss.

**Beispiele für solche Anweisungen in einem MCP-Prompt:**
*   *"Nutze das Tool `delete_file` NIEMALS, ohne vorher den User um Bestätigung zu bitten."*
*   *"Bevor du das Tool `send_email` aufrufst, nutze IMMER erst `check_spam_score`, um den Text zu prüfen."*
*   *"Falls das Tool `get_weather` einen Fehler liefert, versuche es automatisch mit dem Tool `get_forecast` erneut."*

## Beispiel: Der "Sicherheits-Check" Prompt

Stellen wir uns einen Server vor, der Zugriff auf ein Bankkonto hat. Ein strategischer Prompt namens `secure_transfer` könnte so aussehen:

### 1. Implementierung im MCP-Server (Go)
```go
s.AddPrompt(mcp.NewPrompt("secure_transfer",
    mcp.WithPromptDescription("Anweisungen für sichere Überweisungen"),
), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    return &mcp.GetPromptResult{
        Messages: []mcp.PromptMessage{
            {
                Role: mcp.RoleSystem,
                Content: mcp.TextContent{
                    Text: `Du bist ein Sicherheits-Assistent für Finanzen.
                           WICHTIGE REGELN:
                           1. Nutze das Tool 'execute_transfer' NUR, wenn der Betrag unter 100€ liegt.
                           2. Für Beträge ÜBER 100€ musst du IMMER erst das Tool 'request_2fa' aufrufen.
                           3. Antworte dem User erst, wenn die Transaktion bestätigt wurde.`,
                },
            },
        },
    }, nil
})
```

### 2. Was das LLM im JSON-Flow sieht
Wenn der Client diesen Prompt lädt, wird der System-Prompt des LLM wie folgt injiziert:

```json
{
  "model": "gpt-4",
  "messages": [
    {
      "role": "system",
      "content": "Du bist ein Sicherheits-Assistent für Finanzen. WICHTIGE REGELN: 1. Nutze das Tool 'execute_transfer' NUR... [gekürzt]"
    },
    {
      "role": "user",
      "content": "Überweise 150€ an Max."
    }
  ],
  "tools": [
    { "name": "execute_transfer", "parameters": { ... } },
    { "name": "request_2fa", "parameters": { ... } }
  ]
}
```

**Ergebnis:** Das LLM wird nun aufgrund der Anweisung im Prompt erkennen: *"Halt, 150€ ist über 100€, ich muss erst `request_2fa` aufrufen!"* – obwohl der User das nicht explizit gesagt hat.

## Woher kommen diese Anweisungen? (Vom Server!)

Diese strategischen Anweisungen werden **nicht im Client programmiert**. Sie werden vom **MCP-Server** definiert. Der Server liefert also das **Werkzeug (Tool)** und die **Fachkunde (Prompt)** aus einer Hand.

Das macht das System extrem wartungsfreundlich: Wenn sich die Sicherheitsregeln ändern (z. B. 2FA erst ab 500€), änderst du nur den Prompt auf dem Server. Alle angeschlossenen KIs verhalten sich sofort korrekt.

## Prompts testen mit `mcp-tester`

Nutze den Tester, um sicherzustellen, dass deine "Guiderails" (Leitplanken) im Prompt klar und unmissverständlich formuliert sind:

```bash
./bin/mcp-tester prompts get secure_transfer --profile local
```

Prüfe in der Ausgabe, ob die Regeln vollständig und korrekt als `system` oder `user` Nachricht erscheinen. Nur wenn der Prompt sitzt, wird das LLM deine Tools sicher und effizient orchestrieren.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Multi-Server Orchestrierung →](07_die_macht_der_kombination.md)

---
*Copyright Michael Lechner - 2026-02-28*
