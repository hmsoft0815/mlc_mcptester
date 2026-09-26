# MCP-Tester

<img src="docs/assets/hero.jpg" alt="MCP-Tester — Illustration" width="820">


> **[mlcgo.eu](https://mlcgo.eu)** — tools, libraries and manuals · [Product page](https://mlcgo.eu/products/mlc-tester/)


Ein Command-Line Tool zum Testen, Debuggen und Validieren von Model Context Protocol (MCP) Servern nach der aktuellen Spezifikation vom 2026-07-28.

[English Version](README.en.md)

## Warum MCP-Tester?

Fast täglich erscheinen neue Hosts, Harnesses und Modelle, und die MCP-Spezifikation entwickelt sich laufend weiter. Ein Server, der die Spezifikation nicht genau einhält, läuft dann oft „plötzlich“ nicht mehr sauber: Aufrufe scheitern bei strikteren Clients, Fehlermeldungen helfen dem Modell nicht weiter, oder das Modell verbrennt unnötig Tokens mit Wiederholungen. `mcp-tester` hilft, schnell und mit wenig Aufwand auf dem Stand der Technik zu bleiben.

### Tests schreiben statt klicken

Klassische Unit-Tests (`go test`, `pytest`, `npm test`) rufen die Handler-Funktion direkt auf. Sie prüfen die Geschäftslogik, **nicht den MCP-Protokollvertrag**: Handshake, Schema-Validierung, Fehlercodes, strukturierte Ergebnisse, Fortschritt, Abbruch. Genau dort entstehen die Fehler, die erst auffallen, wenn ein echtes LLM das Tool aufruft.

Mit den deklarativen **`.mcp`-Testskripten** testet ihr den Server so, wie ein Client ihn sieht: als Blackbox über den echten Transport (`stdio`, SSE, Streamable HTTP), mit Variablen, Typumwandlung und Assertions, ohne Boilerplate und ohne SDK-Mocks. Die Skripte laufen als Gate in CI und Taskfile (Exit-Code, `--format json`).

```mcp
call_tool generate_image prompt:"a red flower"
set_var size $.size
assert_equals $size "512x512"

expect_error call_tool generate_image prompt:""
assert_tool_error
```

### Früh wissen, was die nächste Spec-Revision verlangt

`mcp-tester` wird regelmäßig an die neueste MCP-Spezifikation angepasst; was er zu welchem Datum prüft, steht in der [Spec-Abdeckung](docs/SPEC_COVERAGE.de.md). Wer seine Server damit testet, sieht notwendige Anpassungen, bevor Nutzer oder Hosts darüber stolpern, und erkennt auch **veraltete Features**, etwa eine alte Protokollversion, entfernte Methoden wie `ping` oder stillgelegte Fehlercodes.

Wir schreiben selbst viele MCP-Server für die unterschiedlichsten Anwendungen. Gerade dort spart `mcp-tester` wertvolle Zeit: Ein Lauf der Testskripte zeigt nach einem Spec- oder SDK-Update sofort, welche Server angepasst werden müssen und wo es klemmt.

Besonders bei neuen Features lohnt frühes Testen. Fehler in der **Autorisierung** (OAuth, Token, Header) sind nicht nur lästig, sondern können gefährlich und teuer werden. `mcp-tester` spielt den vollständigen OAuth-2.1-Ablauf durch und prüft mit `http-check` die Transportregeln von Streamable HTTP.

### Fremde und Closed-Source-Server vor der Freigabe prüfen

Auch bei MCP-Servern, deren Quellcode ihr nicht kennt, ist eine Analyse sinnvoll, **bevor** ihr sie einem „echten“ LLM zur Verfügung stellt. `inspect` zeigt Protokollversion, Fähigkeiten, Tools mit Schemas und Annotations (z. B. ob ein Tool nur liest), Icons und Auffälligkeiten; `http-check` zeigt, wie sauber der Transport umgesetzt ist; `skills --verify` zeigt, welche Anweisungen (Skills) der Server dem Modell mitgibt, welche Tools sie vorab freigeben wollen (`allowed-tools`) und ob der Inhalt zu den veröffentlichten Digests passt.

Unser eigener, interner Harness zeigt genau solche Details an, bevor ein MCP-Server freigeschaltet wird. Für Anwender ist das eine sehr hilfreiche Entscheidungshilfe: verwenden, ja oder nein?

### Findet auch Fehler im SDK

Weil `mcp-tester` den Server von außen gegen die Spezifikation prüft, findet er nicht nur Fehler im eigenen Code. Bei der Anpassung an Spec 2026-07-28 sind uns so auch Fehler und Lücken im verwendeten SDK aufgefallen, zum Beispiel ein Base64-kodierter `Mcp-Name`-Header, den das offizielle Go-SDK v1.8.0 fälschlich ablehnt.

## Kern-Features

- **Echte Client-Perspektive** über `stdio`, SSE und **Streamable HTTP**, mit Profilen in `mcp-tester.yml`.
- **Scripting Engine** (`.mcp`): Tools, Tasks, Completion, Elicitation, Sampling, Roots und Notifications skriptbar, mit Assertions und Exit-Code für CI.
- **MCP Apps**: interaktive Oberflächen eines Servers auflisten und prüfen (`apps`): Tool-Verknüpfung, `ui://`-Resources, externe Domains (CSP) und angeforderte Berechtigungen wie Kamera oder Mikrofon – wichtig vor der Freigabe.
- **Auth-Extensions**: Client Credentials (Maschine-zu-Maschine) und Enterprise-Login per ID-JAG; `auth-check` zeigt, welche Flows ein Server anbietet.
- **Skills-Extension**: Skills eines Servers auflisten und verifizieren (`skills --verify`): Manifest, Digests, Frontmatter, Namensregeln. Das Go-Paket [`pkg/mcpskills`](pkg/mcpskills) stellt Skills aus einem Verzeichnis bereit.
- **Tasks-Extension**: lang laufende Tool-Aufrufe als Task starten, pollen, abbrechen, Eingaben nachreichen. Für Server-Autoren gibt es das Go-Paket [`pkg/mcptasks`](pkg/mcptasks), das die Extension auf Servern mit dem offiziellen go-sdk nachrüstet.
- **Server Inspector** (`inspect`): Spec-Abgleich, Best Practices und Quality Score; `--min-score` als CI-Gate.
- **HTTP-Konformität** (`http-check`): Pflicht-Header, Fehlercodes, Origin, Sessions nach Spec 2026-07-28.
- **Autorisierung**: Bearer-Token, eigene Header und der OAuth-2.1-Ablauf (PKCE, Protected Resource Metadata, `iss`-Prüfung).
- **Strikte Ergebnisprüfung**: Jedes Ergebnis wird gegen das `outputSchema` geprüft, so streng wie das offizielle TypeScript-SDK.
- **Aktuelle Spezifikation**: Protokoll 2026-07-28 mit Rückfall auf ältere Revisionen; Abdeckung mit Datum dokumentiert.
- **Raw Mode** für tiefgreifendes Debugging nicht konformer Server.

## Häufige Fragen

**`mcp-tester` meldet Fehler – was tun?**
Unsere Empfehlung ist klar: gegen die aktuelle Spezifikation testen und die Meldungen ernst nehmen. Die Ausgabe nennt Methode, Feld, Fehlercode und die verletzte Regel. Moderne LLMs wie Claude oder Gemini machen daraus in der Regel schnell konkrete Vorschläge zur Behebung – gebt ihnen die Ausgabe (gern mit `--format json`) einfach mit.

**Muss ich den Quality Score auf 100/100 bringen?**
Nein. Der Score beruht auf unseren eigenen Erfahrungen und Bewertungen; ein Wert unter 100 muss nicht automatisch eine Änderung am MCP-Server auslösen. Wir halten die Zahl trotzdem für eine sehr hilfreiche Information, deshalb gibt es sie. Anders sind **Protokollfehler** (`inspect`) und **FAIL** (`http-check`): Das sind Verstöße gegen MUST-Regeln der Spezifikation, an denen echte Clients scheitern.

---

## Der "Everything" Test-Server

Im Projekt ist ein Referenz-Server (`cmd/test-server`) enthalten, der die Möglichkeiten des MCP-Protokolls vorführt: Tools mit Output-Schemata, Resources und Templates, Prompts, Completion, Logging, Progress, Elicitation, Sampling, Roots, Notifications über `subscriptions/listen` und `x-mcp-header`. Mit `-addr :8080` läuft er über HTTP, mit `-auth` zusätzlich OAuth-geschützt mit eingebautem Test-Autorisierungsserver.

---

## Benutzung

### 1. Installation

**Über Go (Direkt von GitHub):**
```bash
go install github.com/hmsoft0815/mlc_mcptester/cmd/mcp-tester@latest
```
*Hinweis: Stellt sicher, dass `$GOPATH/bin` (meist `~/go/bin`) in eurem `PATH` liegt.*

**Erste Schritte:**
Nach der Installation könnt ihr euren ersten Server hinzufügen und sofort testen:
```bash
# Server-Profil hinzufügen
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"

# Verfügbare Tools auflisten
mcp-tester list -p my-server
```

**Über Curl (Linux/macOS):**
```bash
curl -sSL https://raw.githubusercontent.com/hmsoft0815/mlc_mcptester/main/scripts/install.sh | bash
```

**Manueller Build:**
```bash
git clone https://github.com/hmsoft0815/mlc_mcptester.git
cd mlc_mcptester
task all            # Baut den Tester und Referenz-Server in den bin/ Ordner
```

> **Ohne `--recursive` klonen.** Das Repository verweist auf zwei
> Submodule (`.mlcai`, `mlcprodweb`), die auf einem internen Server
> liegen — interne Dokumentation und die Produktseite. Für den Bau
> werden sie nicht gebraucht; `git clone --recursive` bricht ohne
> Zugang zu diesem Server ab.

<img src="docs/assets/inspect-and-test.png" alt="mcp-tester inspect und test gegen einen MCP-Server" width="820">

*Echte Ausgabe: der Inspektor und ein Testlauf gegen den MCP-Server von
[mlc OpticScript](https://mlcgo.eu/products/mlc-opticscript/).*

### 2. Kommandos (Auszug)

#### Profilverwaltung
Verwaltet verschiedene Server-Konfigurationen direkt über die CLI:
```bash
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"
mcp-tester profile list
mcp-tester profile disable my-server
mcp-tester profile delete my-server
```

#### Geschützte Server (Autorisierung)
Für HTTP-Transporte. Statische Zugangsdaten per Flag oder Profil, sonst der OAuth-2.1-Ablauf der Spec (PKCE, Protected Resource Metadata, `iss`-Prüfung; Client-Registrierung per Client ID Metadata Document, vorregistriertem Client oder dynamisch):
```bash
# Bearer-Token oder beliebige Header
mcp-tester list -u https://example.com/mcp --bearer "$TOKEN"
mcp-tester list -u https://example.com/mcp -H "X-Api-Key: $KEY"

# OAuth: URL wird ausgegeben (oder mit --oauth-browser geöffnet)
mcp-tester inspect -u https://example.com/mcp --oauth
mcp-tester inspect -u https://example.com/mcp --oauth --oauth-client-id my-client

# Ohne Browser, für Autorisierungsserver ohne Nutzerinteraktion (CI, test-server -auth)
mcp-tester inspect -u http://127.0.0.1:8080/mcp --oauth --oauth-auto

# Auth-Extensions: Maschine-zu-Maschine und Enterprise-Login (ID-Token → ID-JAG)
mcp-tester list -u https://example.com/mcp --oauth-client-credentials --oauth-client-id svc --oauth-client-secret "$SECRET"
mcp-tester list -u https://example.com/mcp --oauth-enterprise --idp-issuer https://idp.example --idp-client-id app \
  --id-token "$ID_TOKEN" --oauth-client-id app

# Welche Flows bietet der Server an, und stimmen die Metadaten?
mcp-tester auth-check -u https://example.com/mcp
```
Im Profil: `headers:` und `bearer:`, `${VAR}` wird aus der Umgebung ersetzt. `./bin/test-server -addr :8080 -auth` startet einen geschützten Test-Server mit eingebautem Autorisierungsserver (statisches Token `test-token`).

#### HTTP-Konformität (`http-check`)
Prüft einen Streamable-HTTP-Endpunkt mit gezielt gebauten Requests gegen die Transportregeln von Spec 2026-07-28: Pflicht-Header (`MCP-Protocol-Version`, `Mcp-Method`, `Mcp-Name`, Base64-Werte, `Mcp-Param-*` aus `x-mcp-header`), Fehlercodes mit HTTP-Status (`-32020`, `-32022`, 404/`-32601`), Origin-Prüfung (403), 405 für GET/DELETE und keine Sessions. MUST-Verstöße: FAIL und Exit 1, SHOULD-Verstöße: WARN.
```bash
mcp-tester http-check -u https://example.com/mcp --bearer "$TOKEN"
mcp-tester http-check -u http://127.0.0.1:8080/mcp --format json
```

#### Server Inspektion
Analysiere einen Server auf Qualität (Metadaten, Prompts, Struktur):
```bash
# Mit Profil aus mcp-tester.yml
mcp-tester inspect --profile local

# Direktaufruf ohne Konfigurationsdatei
mcp-tester inspect -c "npx -y @modelcontextprotocol/server-everything"

# Als CI-Gate: Exit 1 bei Protokollfehlern oder Score unter der Schwelle
mcp-tester inspect -p local --min-score 90
```

#### Tools, Resources & Prompts
```bash
mcp-tester list -p local
mcp-tester resources list --cursor "NEXT_TOKEN" -p local
mcp-tester prompts get code_review --args '{"file_path": "main.go"}' -p local
mcp-tester complete prompt:code_review file_path ma -p local
```

#### Test-Skripte (Automatisierung)
Führe komplexe Test-Szenarien aus:
```bash
./bin/mcp-tester test --script tests/03_variables_and_math.mcp --profile local
```

## Dokumentation

- [Das MCP-Handbuch (Online)](https://mlcgo.eu/books/mcp-handbuch/) — Die umfassende Einführung und Referenz in das Model Context Protocol (Deutsch).
- [Scripting Referenz (DE)](docs/SCRIPTING.de.md) — Detaillierte Dokumentation der Test-Grammatik.
- [Scripting Reference (EN)](docs/SCRIPTING.md) — Detailed documentation of the test grammar.
- [Changelog](CHANGELOG.md) — Änderungen je Version (Added/Changed/Fixed).
- [Spec-Abdeckung (DE)](docs/SPEC_COVERAGE.de.md) — Welche Features der aktuellen MCP-Spezifikation geprüft werden, mit Datum.

---

## Lizenz

Dieses Projekt steht unter der [MIT Lizenz](LICENSE).

---
*Copyright Michael Lechner - 2026-03-09*

<!-- mlcai-private -->
## Projektdokumentation (`.mlcai/`)

`.mlcai/` ist ein **privates Git-Submodul**: interne Planung, Backlog und Arbeitsnotizen, gepflegt mit dem MLC Doc Hub. Es ist nicht öffentlich zugänglich — **ohne** `--recurse-submodules` klonen; für den Build wird es nicht gebraucht. Links nach `.mlcai/` funktionieren nur mit Zugriff (`git submodule update --init .mlcai`).

## Wer ist „Claude“ in den Commits?

Einige Commits in diesem Repository sind zusammen mit Claude entstanden, dem
KI-Modell von Anthropic. Es schreibt Code mit, hält Dokumentation und Backlog
aktuell und sucht Fehler in der Build-Pipeline – jede Änderung wird geprüft,
bevor sie übernommen wird. Wir verstecken das nicht:
[wie wir mit Claude arbeiten](https://mlcgo.eu/ai/).
