# MCP Scripting Engine

Die Scripting Engine des `mcp-tester` ermöglicht automatisierte Testabläufe für MCP-Server. Skripte werden in Dateien mit der Endung `.mcp` gespeichert.

## Allgemeine Syntax

- **Befehle**: Ein Befehl pro Zeile.
- **Kommentare**: Zeilen, die mit `#` oder `//` beginnen, werden ignoriert. Trailing-Kommentare sind ebenfalls erlaubt.
- **Variablen**: Werden mit dem Präfix `$` angesprochen (z.B. `$name`).
- **Strings**: Können in Anführungszeichen gesetzt werden, wenn sie Leerzeichen enthalten.

---

## Befehlsübersicht

### 1. `call_tool`
Ruft ein MCP-Tool auf.
```mcp
call_tool <tool_name> [arg1] [arg2] ...
```
- **Argumente**: Werden positional übergeben und automatisch anhand des JSON-Schemas des Tools in den richtigen Typ (integer, boolean, etc.) konvertiert. Die Reihenfolge entspricht der **alphabetischen Sortierung** der Property-Namen im Schema.

### 2. `set_var`
Extrahiert einen Wert aus der letzten Tool-Antwort und speichert ihn in einer Variable.
```mcp
set_var <variable_name> <pfad>
```
- **Pfade**:
    - `rawResponse`: Speichert die komplette JSON-Antwort des Servers.
    - `structuredContent.<pfad>`: Navigiert durch die JSON-Struktur (Punkt-Notation).
    - `$.<pfad>`: Kurzform für `structuredContent`.

### `call_tool <tool_name> <arg1> <arg2> ...`
Ruft ein Tool auf dem MCP-Server mit den angegebenen positionalen Argumenten auf. Argumente mit Leerzeichen müssen in Anführungszeichen gesetzt werden.

**Heredoc Unterstützung:**
Für mehrzeilige Argumente (z.B. JSON oder Code-Blöcke) kann die Heredoc-Syntax verwendet werden:
```
call_tool execute_script <<EOF
console.log("Hallo vom Heredoc!");
console.log(1 + 2);
EOF
```

---

### `assert_contains <expected>` oder `assert_contains <value> <expected>`
Prüft, ob die letzte Antwort (Text oder JSON) oder ein spezifischer Wert den erwarteten String enthält.
- `assert_contains "Execution finished"` (prüft letzte Antwort)
- `assert_contains $var "expected"` (prüft Variableninhalt)

---

### `assert_equals <expected>` oder `assert_equals <value> <expected>`
Prüft auf eine exakte Übereinstimmung mit der letzten Antwort oder zwischen zwei Werten.
- `assert_equals "30"` (prüft letzte Antwort)
- `assert_equals $var "true"` (prüft Variableninhalt)

---
```mcp
assert_equals "Exakter Text"
```

### 3. `input_var`
Fragt den Benutzer während des Tests nach einer Eingabe.
```mcp
input_var <variable_name> ["Interaktiver Prompt"]
```

### 4. `assert_contains`
Prüft, ob die letzte Antwort einen bestimmten Text enthält.
```mcp
assert_contains "Erwarteter Text"
```

### 5. `assert_equals`
Prüft auf exakte Übereinstimmung der letzten Antwort.
```mcp
assert_equals "Exakter Text"
```

### 6. `assert_number`
Prüft, ob ein Wert (oder eine Variable) eine gültige Zahl ist.
```mcp
assert_number $variable
```

### 7. `assert_gt`
Prüft, ob der erste Wert größer als der zweite ist.
```mcp
assert_gt $wert1 $wert2
```
### 8. `assert_string_length`
Prüft, ob die Länge eines Strings (oder einer Variable) in einem bestimmten Bereich liegt.
```mcp
assert_string_length $variable <min> <max>
```
- `assert_string_length $var 5 10`

---

## Beispiel-Skript

```mcp
# 1. Tool aufrufen und ID speichern
call_tool create_user "Max Mustermann"
set_var user_id $.id

# 2. Variable in nächstem Aufruf nutzen
call_tool get_user $user_id
assert_contains "Mustermann"

# 3. Mathematische Prüfung
set_var score $.profile.score
assert_gt $score 0
```

---

## Ausführung
Ein Skript wird über den Menüpunkt `test` oder direkt per CLI gestartet:
```bash
mcp-tester test --script my_test.mcp --profile my_server
```
