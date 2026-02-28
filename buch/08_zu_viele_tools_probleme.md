# Kapitel 8: Die Tool-Falle – Probleme bei zu vielen Werkzeugen

In den vorangegangenen Kapiteln haben wir gelernt, wie einfach es ist, MCP-Server und Tools zu erstellen. Doch Vorsicht: Mehr ist nicht immer besser. In diesem Kapitel besprechen wir das Phänomen der "Tool-Fatigue" bei LLMs und wie man damit umgeht.

## Das Problem: Wenn das Modell den Wald vor lauter Bäumen nicht sieht

Stell dir vor, du verbindest 20 MCP-Server mit jeweils 50 Tools. Das LLM hat nun 1.000 Werkzeuge zur Auswahl. Dies führt zu drei kritischen Problemen:

1.  **Context Window Inflation**: Jede Tool-Definition (Name, Beschreibung, Parameter) verbraucht Token. Bei 1.000 Tools ist ein großer Teil des Gedächtnisses (Context Window) des Modells bereits belegt, bevor der Benutzer überhaupt die erste Frage gestellt hat.
2.  **Modell-Konfusion (Halluzinationen)**: Je mehr Tools mit ähnlichen Namen oder Beschreibungen existieren, desto eher wählt das Modell das falsche Werkzeug aus. Es beginnt zu "raten" oder vermischt Parameter.
3.  **Latenz**: Das Modell benötigt mehr Zeit, um die riesige Liste der Tools zu verarbeiten, bevor es mit der Generierung beginnt. Die Antwortzeiten steigen spürbar.

---

## Strategien zur Lösung

Wie gehen wir mit dieser Komplexität um? Es gibt drei bewährte Ansätze im MCP-Ökosystem:

### 1. Relevanz-Filterung (Client-seitig)
Ein intelligenter MCP-Client injiziert nicht *alle* Tools auf einmal. Stattdessen nutzt er ein kleineres, schnelles Modell (oder eine Vektor-Suche), um basierend auf der User-Anfrage nur die 5-10 relevantesten Tools auszuwählen und an das Haupt-LLM weiterzugeben.

### 2. Hierarchische Tools (Routing)
Anstatt 100 spezialisierte Tools anzubieten, bietet der Server ein einziges "Router-Tool" an.
*   **Schlecht**: `get_sales_2023`, `get_sales_2024`, `get_sales_prognosis`...
*   **Gut**: Ein Tool `query_data(category, year)`, das intern auf dem Server entscheidet, welche Logik aufgerufen wird.

### 3. Klare Abgrenzung durch Prompts
Wie in Kapitel 6 gelernt, können **Prompts** dem Modell helfen, sich zu fokussieren. Ein Prompt kann dem Modell sagen: *"Du bist heute nur für die Buchhaltung zuständig. Ignoriere alle Tools, die nichts mit Finanzen zu tun haben."*

---

## Testing der Tool-Überlastung mit `mcp-tester`

Mit unserem Tool kannst du genau prüfen, wie dein Server reagiert, wenn er unter Last steht oder sehr viele Definitionen liefert.

### Tool-Discovery Benchmarking
Prüfe, wie lange der Handshake dauert, wenn dein Server viele Tools zurückgibt:
```bash
time ./bin/mcp-tester list --profile heavy-server
```

### Ambiguitäts-Test
Erstelle ein Test-Skript, das versucht, das Modell zu verwirren. Bietet dein Server zwei sehr ähnliche Tools an? Teste mit dem `mcp-tester`, ob das Modell (oder dein Skript-Ablauf) zuverlässig das richtige Tool trifft.

## Fazit

Ein guter MCP-Server zeichnet sich nicht durch die *Anzahl* der Tools aus, sondern durch deren **Qualität und Eindeutigkeit**. Jedes Tool sollte einen klaren, einzigartigen Zweck erfüllen. Wenn du merkst, dass dein Modell Fehler macht, ist es oft an der Zeit, die Tool-Liste zu entschlacken oder die Beschreibungen präziser zu formulieren.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Rückgabewerte →](09_rueckgabewerte_text_vs_structured.md)

---
*Copyright Michael Lechner - 2026-02-28*
