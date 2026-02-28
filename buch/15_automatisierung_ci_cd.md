# Kapitel 15: Automatisierung & CI/CD

Der `mcp-tester` ist so konzipiert, dass er nicht nur für manuelles Debugging, sondern als integraler Bestandteil deines Software-Lebenszyklus funktioniert. In diesem Kapitel lernen wir, wie wir Tests automatisieren.

## Exit-Codes: Die Sprache der Pipelines

Ein Test-Tool in einer CI/CD-Umgebung (wie GitHub Actions oder GitLab CI) muss über den Erfolg oder Misserfolg informieren. Der `mcp-tester` nutzt hierfür Standard-Exit-Codes:

*   **Exit Code 0**: Erfolg. Alle Schritte im Skript wurden fehlerfrei ausgeführt.
*   **Exit Code 1**: Fehler. Ein Tool-Aufruf schlug fehl oder eine Assertion wurde verletzt.

### Beispiel für ein Shell-Skript:
```bash
./bin/mcp-tester test --profile staging --script tests/smoke_test.mcp
if [ $? -eq 0 ]; then
  echo "Deployment validiert!"
else
  echo "Kritischer Fehler im MCP-Server!"
  exit 1
fi
```

---

## Test-Zusammenfassungen

Am Ende jedes Skript-Durchlaufs gibt der Tester eine kompakte Statistik aus:
`Test Summary: 12 commands executed, 12 passed, 0 failed`

Dies ermöglicht es Entwicklern, auf einen Blick zu sehen, wie umfangreich die Testabdeckung ist.

---

## Best Practices für stabile Pipelines

1.  **Isolierte Umgebungen**: Nutze separate Profile in der `mcp-tester.yml` für `dev`, `staging` und `production`.
2.  **Variablen für Dynamik**: Nutze `set_var`, um IDs aus Erstellungs-Tools zu extrahieren und in Folge-Tests zu verwenden. Dies vermeidet hartcodierte Testdaten.
3.  **Leitplanken-Tests**: Erstelle Skripte, die gezielt die strategischen Prompts (Kapitel 6) prüfen. Reagiert der Server korrekt auf unzulässige Anfragen?
4.  **Raw-Mode für Logs**: Wenn eine Pipeline fehlschlägt, kann der `--raw` Modus in den CI-Logs helfen, die exakte JSON-Antwort des Servers zu sehen.

## Fazit

Durch die Automatisierung deiner MCP-Tests mit dem `mcp-tester` schaffst du Vertrauen in deine KI-Infrastruktur. Du stellst sicher, dass Änderungen am Code deines Servers nicht zu unvorhersehbarem Verhalten im LLM-Chat führen.
