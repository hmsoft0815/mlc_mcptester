# System Integration Tests mit mcp-tester

Dieses Dokument beschreibt, wie der `mcp-tester` für automatisierte System-Integrationstests (z. B. in CI/CD Pipelines) eingesetzt werden kann.

## Automatisierung & Exit-Codes

Der `mcp-tester` ist so konzipiert, dass er bei Fehlern in einem Skript sofort mit einem **Nicht-Null-Exit-Code** abbricht. Dies ermöglicht eine nahtlose Integration in GitHub Actions, GitLab CI oder Jenkins.

*   **Exit Code 0**: Alle Befehle und Assertions im Skript waren erfolgreich.
*   **Exit Code 1**: Ein Tool-Aufruf ist fehlgeschlagen oder eine Assertion (Zusicherung) war nicht korrekt.

### Beispiel für CI/CD (Bash)
```bash
./bin/mcp-tester test --profile prod --script tests/full_suite.mcp
if [ $? -eq 0 ]; then
  echo "Integrationstest erfolgreich!"
else
  echo "Integrationstest fehlgeschlagen!"
  exit 1
fi
```

---

## Test-Zusammenfassung

Am Ende eines Skript-Durchlaufs gibt der Tester eine Zusammenfassung aus:
```text
Test Summary: 12 commands executed, 12 passed, 0 failed
```
Sollte ein Fehler auftreten, bricht der Tester die Ausführung an der entsprechenden Zeile ab und gibt den Grund an (z. B. `assertion failed`).

---

## Best Practices für Integrationstests

1.  **Status-Prüfungen**: Nutze `call_tool health_check` (falls vorhanden) zu Beginn eines Tests.
2.  **Variablen nutzen**: Extrahiere IDs aus Erstellungs-Tools (`set_var`) und nutze sie in Lese- oder Lösch-Tools, um echte Workflows zu simulieren.
3.  **Mathematische Validierung**: Nutze `assert_gt` und `assert_number` für Performance-Metriken oder finanzielle Berechnungen.
4.  **Profile**: Nutze die `mcp-tester.yml`, um unterschiedliche Umgebungen (Staging, Production) sauber zu trennen.

---

## Beispiel: Komplexer Workflow-Test

```text
// 1. Neues Objekt erstellen
call_tool create_item "Test-Produkt"
set_var item_id structuredContent.id

// 2. Objekt lesen
call_tool get_item $item_id
assert_contains "Test-Produkt"

// 3. Status prüfen
assert_equals "active" structuredContent.status
```

Durch diese Struktur stellst du sicher, dass dein MCP-Server nicht nur einzelne Tools korrekt ausführt, sondern als Gesamtsystem stabil funktioniert.
