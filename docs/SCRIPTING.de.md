# MCP Scripting Engine

Die Scripting Engine des `mcp-tester` ermöglicht automatisierte Testabläufe für MCP-Server. Skripte werden in Dateien mit der Endung `.mcp` gespeichert.

## Allgemeine Syntax

- **Befehle**: Ein Befehl pro Zeile.
- **Kommentare**: Zeilen, die mit `#` oder `//` beginnen, werden ignoriert. Trailing-Kommentare sind ebenfalls erlaubt.
- **Variablen**: Werden mit dem Präfix `$` angesprochen (z.B. `$name`). Die Ersetzung erfolgt per Regex (`\$([A-Za-z_][A-Za-z0-9_]*)`). Die Verwendung unbekannter Variablen bricht die Ausführung mit einem klaren Fehler und Zeilenangabe ab.
- **Strings**: Können in Anführungszeichen (`"..."` oder `'...'`) gesetzt werden, wenn sie Leerzeichen oder Sonderzeichen enthalten. In doppelten Anführungszeichen sind `\"` und `\\` Escapes (`"missing: [\"code\"]"`); jeder andere Backslash bleibt wörtlich. Einfache Anführungszeichen werden unverändert übernommen. Ein nicht geschlossenes Anführungszeichen lässt die Zeile fehlschlagen.
- **Listen / Objekte**: JSON-Arrays (`'["a.png", "b.png"]'`) oder JSON-Objekte (`'{"key": "value"}'`) können direkt als Argument übergeben werden.

---

## Befehlsübersicht

### 1. `echo`
Gibt eine Nachricht in der Konsole aus. Nützlich zur Strukturierung von Testausgaben und für Abschnittsüberschriften in längeren Skripten.
```mcp
echo "--- Testing user creation workflow ---"
echo "Aktuelle ID: $user_id"
```

### 2. `call_tool`
Ruft ein MCP-Tool auf.
```mcp
call_tool <tool_name> [arg1] [arg2] ...
```
- **Argumente**: Können positional oder als benannte Argumente (`key:value`) übergeben werden.
    - **Positional**: Werden basierend auf dem JSON-Schema des Tools automatisch in den richtigen Typ (Integer, Boolean, Array, Object etc.) konvertiert. Die Reihenfolge entspricht der **alphabetischen Sortierung** der Property-Namen im Schema.
    - **Benannt**: Folgen der Syntax `key:value` (z.B. `paths:'["a.png", "b.png"]'`). Dies wird empfohlen, um Verwechslungen durch die alphabetische Sortierung zu vermeiden.
    - **Arrays und Objekte**: Unterstützt Schemata mit `type: "array"`, `type: "object"` sowie Nullable-Definitionen (`type: ["null", "array"]`).
    - **Gemischt**: Es können beide Arten gemischt werden; positionale Argumente füllen die verbleibenden Properties in alphabetischer Reihenfolge auf.
- **Ergebnisprüfung**: Gibt das Tool ein `outputSchema` an, schlägt der Aufruf fehl, wenn das Ergebnis kein dazu passendes `structuredContent` enthält — dieselbe Prüfung, die strikte Clients machen (das offizielle TypeScript-SDK, das OpenCode nutzt, lehnt so einen Aufruf mit `-32600` ab). Ergebnisse mit `isError: true` sind ausgenommen. Das Go-SDK, auf dem der Tester aufbaut, prüft das selbst nicht; deshalb tut es der Tester.

**Heredoc-Unterstützung:**
Für mehrzeilige Argumente (z.B. JSON oder Code-Blöcke) kann die Heredoc-Syntax verwendet werden:
```mcp
call_tool execute_script <<EOF
console.log("Hallo vom Heredoc!");
console.log(1 + 2);
EOF
```

### 3. `expect_error`
Wird vor einen Befehl gestellt, wenn ein Fehler (Werkzeugfehler oder Protokollfehler) erwartet wird.
```mcp
expect_error call_tool add a:"keine_zahl"
assert_tool_error
assert_contains "type"

expect_error call_tool unknown_tool
assert_error_code -32602
```

### 4. `assert_tool_error` und `assert_error_code`
Die MCP-Spezifikation unterscheidet strikt zwischen Werkzeugfehlern (`isError: true` im Ergebnis) und JSON-RPC Protokollfehlern (wie ungültiger Request oder unbekanntes Tool).

- **`assert_tool_error`**: Prüft, dass der Aufruf ein Tool-Ergebnis mit `isError: true` geliefert hat (alternativ auch `assert_error_code tool`).
- **`assert_error_code <code>`**: Prüft den numerischen JSON-RPC Protokollfehler-Code des letzten fehlgeschlagenen Requests. Ein Tool-Fehler (`isError: true`) schlägt bei numerischen Codes fehl.

```mcp
# Tool-Ausführungsfehler
expect_error call_tool validate_script script:"ungültig"
assert_tool_error

# Protokollfehler (-32602 = Invalid params / unknown tool)
expect_error call_tool kein_werkzeug
assert_error_code -32602
```

#### Gängige JSON-RPC Fehler-Codes
- `-32700`: Parse error (ungültiges JSON)
- `-32600`: Invalid Request (ungültiger Request)
- `-32601`: Method not found (Methode nicht gefunden)
- `-32602`: Invalid params (Schema-Validierung auf RPC-Ebene)
- `-32603`: Internal error (Interner Fehler)

### 5. `set_var`
Extrahiert einen Wert aus der letzten Tool-Antwort und speichert ihn in einer Variable.
```mcp
set_var <variable_name> <pfad>
```
- **Pfade**:
    - `rawResponse`: Speichert die komplette JSON-Antwort des Servers.
    - `structuredContent.<pfad>`: Navigiert durch die JSON-Struktur (Punkt-Notation).
    - `$.<pfad>`: Kurzform für `structuredContent.<pfad>`; fehlt das Feld dort, wird die oberste Ebene des Ergebnisses versucht.

### 6. `input_var`
Fragt den Benutzer während des Tests nach einer Eingabe.
```mcp
input_var <variable_name> ["Interaktiver Prompt"]
```

### 7. `assert_contains`
Prüft, ob die letzte Antwort (Text oder JSON) oder ein spezifischer Wert den erwarteten String enthält.
```mcp
assert_contains "Execution finished"
assert_contains $var "expected"
```

### 8. `assert_equals`
Prüft auf eine exakte Übereinstimmung mit der letzten Antwort oder zwischen zwei Werten.
```mcp
assert_equals "Result: 30"
assert_equals $var "123"
```

### 9. `assert_number`
Prüft, ob ein Wert (oder eine Variable) eine gültige Zahl ist.
```mcp
assert_number $variable
```

### 10. `assert_gt`
Prüft, ob der erste Wert größer als der zweite ist.
```mcp
assert_gt $wert1 $wert2
```

### 11. `assert_string_length`
Prüft, ob die Länge eines Strings (oder einer Variable) in einem bestimmten Bereich liegt.
```mcp
assert_string_length $variable <min> <max>
```

---

### 12. `complete`
Fragt den Server nach Vervollständigungen für ein Argument (`completion/complete`).
```mcp
complete <prompt:name|resource:uri> <argument> [wert]
```
Das Ergebnis wird als `{values, total, hasMore}` abgelegt: `assert_contains` sieht die Werte zeilenweise, `set_var` adressiert `values.0` oder `total`.
```mcp
complete prompt:persona_developer language g
assert_contains "golang"
set_var first values.0
```

---

### 13. Eingabeanfragen: `elicit_response`, `sample_response`, `add_root`
Seit Spec 2026-07-28 fragt ein Server über *Multi Round-Trip Requests* nach Eingaben: `tools/call`, `prompts/get` oder `resources/read` liefert `inputRequests`, der Client antwortet und wiederholt den Aufruf. Der Tester erledigt das automatisch; das Skript legt die Antworten **vor** dem Aufruf fest. Eine Anfrage ohne vorbereitete Antwort lässt den Aufruf scheitern.
```mcp
elicit_response accept '{"confirm": true}'   # auch: decline, cancel
sample_response "Antwort des Modells"        # Antwort auf sampling/createMessage
add_root file:///home/user/project project   # angeboten über roots/list
```
Antworten werden der Reihe nach verbraucht (Warteschlange). `assert_elicited <text>` prüft die Nachricht der letzten Elicitation, `assert_sampled <text>` den Prompt der letzten Sampling-Anfrage.
```mcp
elicit_response accept '{"confirm": true}'
call_tool confirm_delete item:"report.pdf"
assert_elicited "report.pdf"
```
Auf der Kommandozeile gibt es dieselben Antworten mit `--elicit 'accept:{"confirm":true}'`, `--sample "text"` und `--root file:///pfad` (jeweils wiederholbar); dort wird eine unbeantwortete Elicitation abgelehnt.

---

### 14. Notifications: `subscribe`, `wait_notification`
Der Tester öffnet einen `subscriptions/listen`-Stream für jede Liste, die der Server als `listChanged` kennzeichnet, und zeichnet `notifications/tools|prompts|resources/list_changed` und `notifications/resources/updated` auf.
```mcp
subscribe mcp://time                             # Resource-Updates für diese URI
wait_notification tools/list_changed             # wartet bis zu 5s (Default)
wait_notification resources/updated mcp://time 2s
```
`wait_notification` ist erfüllt durch eine Notification seit dem letzten Warten, auch eine, die vor dem Befehl eintraf, und verbraucht sie. Mit `expect_error` prüft es, dass nichts ankommt.

Auf der Kommandozeile zeigt `mcp-tester listen [--subscribe <uri>] [--duration 30s]` eintreffende Notifications an.

---

### 15. Tasks: `call_task`, `start_task`, `wait_task`, `get_task`, `cancel_task`
Mit der Tasks-Extension (`io.modelcontextprotocol/tasks`) darf ein Server einen Tool-Aufruf mit einem Task-Handle beantworten und im Hintergrund ausführen. Der Tester kündigt die Extension nur für diese Befehle an; `call_tool` bleibt synchron.
```mcp
call_task long_job seconds:2          # starten, bis zum Ende pollen, Ergebnis wie bei call_tool
assert_task_status completed

start_task long_job seconds:30        # nur das Handle
set_var id taskId
get_task $id                          # ein tasks/get
cancel_task $id
wait_task $id 5s                      # pollen bis completed, failed oder cancelled
assert_task_status cancelled
```
Braucht ein Task Eingaben (Status `input_required`), gehen die mit `elicit_response` / `sample_response` / `add_root` vorbereiteten Antworten per `tasks/update` an den Server. Ein fehlgeschlagener Task (`failed`) ist für `expect_error` / `assert_error_code` ein RPC-Fehler. Auf der Kommandozeile: `mcp-tester call <tool> --task`.

---

### 16. Skills: `list_skills`, `verify_skills`
Für Server, die Agent Skills über die Skills-Extension (`io.modelcontextprotocol/skills`) veröffentlichen.
```mcp
list_skills                 # abgelegt als {skills: [...], count}; Text: ein Name pro Zeile
assert_contains "image-prompting"
set_var n count

verify_skills               # alle Prüfungen inkl. Lesen jeder Datei; scheitert bei jedem Verstoß
```
Auf der Kommandozeile zeigt `mcp-tester skills [--verify]` Name, Beschreibung, Lizenz, `allowed-tools` und Dateien jedes Skills und die Prüfergebnisse.

---

## Beispiel-Skript

```mcp
echo "--- Benutzer-Workflow starten ---"

# 1. Tool aufrufen und ID speichern
call_tool create_user "Max Mustermann"
set_var user_id $.id

# 2. Variable in nächstem Aufruf nutzen
call_tool get_user $user_id
assert_contains "Mustermann"

# 3. Listen-Argument übergeben
call_tool assign_roles user_id:$user_id roles:'["admin", "tester"]'
assert_contains "Roles assigned"

# 4. Mathematische Prüfung
set_var score $.profile.score
assert_gt $score 0
```

---

## Ausführung
Ein Skript wird über den Menüpunkt `test` oder direkt per CLI gestartet:
```bash
mcp-tester test --script my_test.mcp --profile my_server
```

Der Befehl endet mit Status 1, sobald ein Skriptbefehl fehlschlägt, und taugt damit als Gate für CI oder Taskfile. Mit `--format json` steht auf stdout nur die Zusammenfassung (inklusive `failures`-Liste mit Zeile und Fehler); die Ausgabe der einzelnen Befehle geht nach stderr.
