# Test-Scripting

Die MCP-Tester Scripting-Engine ermöglicht es Ihnen, komplexe, mehrstufige Test-Szenarien für Ihre MCP-Server zu definieren. Skripte verwenden die Dateiendung `.mcp` und folgen einer klaren, zeilenbasierten Grammatik.

## Grundlegende Syntax

- **Befehle**: Ein Befehl pro Zeile.
- **Variablen**: Referenzierung mit `$` (z.B. `$userId`).
- **Kommentare**: Zeilen, die mit `#` oder `//` beginnen, werden ignoriert.

## Kernbefehle

### call_tool
Ruft ein MCP-Tool namentlich auf.
```mcp
call_tool mein_tool param1:wert1 param2:wert2
```

### set_var
Extrahiert Daten aus der letzten Antwort mittels Punktnotation.
```mcp
set_var status $.profile.status
```

### Assertions
Validieren Sie das Verhalten Ihres Servers mit integrierten Prüfungen:
- `assert_contains`: Prüft, ob das Ergebnis einen bestimmten Text enthält.
- `assert_equals`: Prüft auf exakte Übereinstimmung.
- `assert_gt`: Prüft, ob eine Zahl größer als ein anderer Wert ist.
- `assert_error_code`: Verifiziert JSON-RPC Fehlercodes.

## Beispiel-Workflow

```mcp
# Benutzer erstellen und ID speichern
call_tool create_user name:"John"
set_var id $.id

# Prüfen, ob der Benutzer existiert
call_tool get_user userId:$id
assert_contains "John"
```
