# Kapitel 21: Benutzerabfragen & Elicitation – Human-in-the-Loop

In den vorherigen Kapiteln haben wir gelernt, wie Server Daten liefern oder die KI (via Sampling) um Hilfe bitten. Doch manchmal braucht ein Server Informationen direkt vom **menschlichen Benutzer** – zum Beispiel ein Passwort, eine Auswahl aus einer Liste oder eine OAuth-Bestätigung. In der MCP-Spezifikation vom 25.11.2025 wurde hierfür der **Elicitation-Mechanismus** (Abfrage-System) standardisiert.

## 1. Was ist Elicitation?

Elicitation (deutsch: "Hervorlocken" oder "Abfrage") erlaubt es dem Server, den Client anzuweisen, ein Eingabefeld oder einen Dialog für den Nutzer anzuzeigen.

### Warum nicht einfach eine Textnachricht?
Früher mussten Server dem Modell sagen: "Bitte frag den Nutzer nach X". Das war unzuverlässig und unsicher. Elicitation bietet:
*   **Struktur**: Der Server definiert genau, welche Daten er braucht (Zahlen, Texte, Auswahl).
*   **Sicherheit**: Sensible Daten (wie Passwörter) können über gesicherte Kanäle abgefragt werden, ohne dass die KI sie "mitliest".
*   **Benutzererfahrung**: Der Client kann native UI-Elemente (Dropdowns, Checkboxen) statt einfachem Text nutzen.

## 2. Die zwei Modi der Abfrage

### Form Mode (`form`)
Der Server sendet ein **JSON-Schema** für ein Formular. Der Client zeigt dieses Formular direkt in seiner Oberfläche an.
*   **Einsatz**: Konfigurationsparameter, Auswahl von Filtern, einfache Texteingaben.
*   **Datentypen**: String, Number, Boolean, Enum (Dropdown), Array (Mehrfachauswahl).

### URL Mode (`url`)
Der Server sendet eine URL, die der Nutzer im Browser öffnen soll.
*   **Einsatz**: OAuth-Logins, Bezahlvorgänge, Akzeptieren von AGBs auf einer Webseite.
*   **Sicherheit**: Die sensiblen Daten (z. B. Zugangsdaten) werden im Browser eingegeben und fließen nie durch den MCP-Client oder das LLM.

## 3. Die Methode `elicitation/create`

Ein Server startet die Abfrage mit dem Request `elicitation/create`.

### Beispiel: Abfrage einer Auswahl (Pseudo-Code)
```go
// Der Server bittet den Client um eine Benutzereingabe
result, err := request.Session.CreateElicitation(ctx, &mcp.CreateElicitationRequest{
    Message: "Welches Datenbank-Schema möchtest du nutzen?",
    Mode:    "form",
    RequestedSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "schema": map[string]any{
                "type": "string",
                "enum": []string{"production", "staging", "development"},
                "default": "development",
            },
        },
    },
})

if result.Action == "accept" {
    selected := result.Content["schema"]
    // Weiterverarbeitung...
}
```

## 4. Sonderfall: `URLElicitationRequiredError` (-32042)

Wenn ein Server ein Tool nicht ausführen kann, weil der Nutzer nicht eingeloggt ist, sendet er diesen speziellen Fehlercode zurück. Der Fehler enthält die URL für den Login. Der Client erkennt dies automatisch, leitet den Nutzer zum Browser und erlaubt danach die Wiederholung des Tool-Aufrufs.

## 5. Elicitation testen mit `mcp-tester`

In unserem Test-Tool nutzen wir den Befehl **`input_var`**, um Elicitation für Skripte zu simulieren. In einer echten Umgebung (wie Claude Desktop) würde hier ein grafischer Dialog erscheinen.

```text
# Simuliert eine Benutzerabfrage im Skript
input_var db_schema "Welches Schema?"
call_tool query_data $db_schema
```

## Fazit

Elicitation schließt die Lücke zwischen Server, KI und Mensch. Es macht MCP zu einer interaktiven Plattform, die auch komplexe Workflows mit Benutzerbeteiligung und sicheren Authentifizierungen (OAuth) souverän beherrscht.

[← Inhaltsverzeichnis](README.md)

---
*Copyright Michael Lechner - 2026-03-09*
