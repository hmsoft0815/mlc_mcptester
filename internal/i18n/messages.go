package i18n

import "fmt"

var Lang = "en"

type MessageKey string

const (
	MsgInspectionTitle MessageKey = "inspection_title"
	MsgServerInfo      MessageKey = "server_info"
	MsgProtocolVersion MessageKey = "protocol_version"
	MsgCapabilities    MessageKey = "capabilities"
	MsgTools           MessageKey = "tools"
	MsgPrompts         MessageKey = "prompts"
	MsgResources       MessageKey = "resources"
	MsgLogging         MessageKey = "logging"
	MsgProgress        MessageKey = "progress"
	MsgCancel          MessageKey = "cancel"
	MsgSubscription    MessageKey = "subscription_capable"
	MsgSupported       MessageKey = "supported"
	MsgFound           MessageKey = "found"
	MsgScore           MessageKey = "score"
	MsgPerfect         MessageKey = "perfect"
	MsgNoDescription   MessageKey = "no_description"
	MsgNoInputSchema   MessageKey = "no_input_schema"
	MsgNoOutputSchema  MessageKey = "no_output_schema"
	MsgNoPrompts       MessageKey = "no_prompts"
	MsgNoLogging       MessageKey = "no_logging"
	MsgTestSummary     MessageKey = "test_summary"
	MsgExecuting       MessageKey = "executing"
	MsgVariableSet     MessageKey = "variable_set"
	MsgExpectedError   MessageKey = "expected_error"
	MsgAssertionPassed MessageKey = "assertion_passed"
	MsgResource        MessageKey = "resource"
	MsgTemplate        MessageKey = "template"
	MsgPrompt          MessageKey = "prompt"
	MsgTool            MessageKey = "tool"
	MsgIcons           MessageKey = "icons"
	MsgDescription     MessageKey = "description"
	MsgSchema          MessageKey = "schema"
)

var messages = map[string]map[MessageKey]string{
	"en": {
		MsgInspectionTitle: "=== MCP Server Inspection: %s ===\n",
		MsgServerInfo:      "[✓] Server: %s (%s)\n",
		MsgProtocolVersion: "[✓] Protocol Version: %s\n",
		MsgCapabilities:    "[i] Capabilities & Features:",
		MsgTools:           "    - Tools:     %v",
		MsgPrompts:         "    - Prompts:   %v",
		MsgResources:       "    - Resources: %v",
		MsgLogging:         "    - Logging:   %v\n",
		MsgProgress:        "    - Progress:  Supported (Protocol Standard)\n",
		MsgCancel:          "    - Cancel:    Supported (Protocol Standard)\n",
		MsgSubscription:    " (Subscription capable)",
		MsgSupported:       "Supported",
		MsgFound:           "[✓] %d %s found.\n",
		MsgScore:           "\n--- Quality Report (Score: %d/100) ---",
		MsgPerfect:         "Perfect! The server follows all best practices.",
		MsgNoDescription:   "WARNING: Tool '%s' has no description. The LLM needs this to understand the tool's purpose.",
		MsgNoInputSchema:   "ERROR: Tool '%s' has no input schema.",
		MsgNoOutputSchema:  "HINT: Tool '%s' has no output schema. Structured returns help the LLM process results precisely.",
		MsgNoPrompts:       "WARNING: No prompts defined. Prompts are highly recommended to set the system context and persona.",
		MsgNoLogging:       "HINT: The server does not support logging. Server-side logs via MCP help debugging.",
		MsgTestSummary:     "\nTest Summary: %d commands executed, %d passed, %d failed\n",
		MsgExecuting:       "Executing: %s %v\n",
		MsgVariableSet:     "Variable set: %s = %s\n",
		MsgExpectedError:   "Expected error caught: %v (code: %d)\n",
		MsgAssertionPassed: "Assertion passed: %s\n",
		MsgResource:        "Resource: %s\n",
		MsgTemplate:        "Template: %s\n",
		MsgPrompt:          "Prompt: %s\n",
		MsgTool:            "Tool: %s\n",
		MsgIcons:           "Icons:",
		MsgDescription:     "Description: %s\n",
		MsgSchema:          "Schema: %+v\n",
	},
	"de": {
		MsgInspectionTitle: "=== MCP Server Inspektion: %s ===\n",
		MsgServerInfo:      "[✓] Server: %s (%s)\n",
		MsgProtocolVersion: "[✓] Protokoll-Version: %s\n",
		MsgCapabilities:    "[i] Fähigkeiten & Features:",
		MsgTools:           "    - Tools:     %v",
		MsgPrompts:         "    - Prompts:   %v",
		MsgResources:       "    - Ressourcen: %v",
		MsgLogging:         "    - Logging:   %v\n",
		MsgProgress:        "    - Progress:  Unterstützt (Protokoll-Standard)\n",
		MsgCancel:          "    - Cancel:    Unterstützt (Protokoll-Standard)\n",
		MsgSubscription:    " (Subscription-fähig)",
		MsgSupported:       "Unterstützt",
		MsgFound:           "[✓] %d %s gefunden.\n",
		MsgScore:           "\n--- Qualitätsbericht (Score: %d/100) ---",
		MsgPerfect:         "Perfekt! Der Server folgt allen Best Practices.",
		MsgNoDescription:   "WARNUNG: Tool '%s' hat keine Beschreibung. Das LLM benötigt diese, um den Zweck zu verstehen.",
		MsgNoInputSchema:   "FEHLER: Tool '%s' hat kein Input-Schema.",
		MsgNoOutputSchema:  "HINT: Tool '%s' hat kein Output-Schema. Strukturierte Rückgaben helfen dem LLM.",
		MsgNoPrompts:       "WARNUNG: Keine Prompts definiert. Prompts werden dringend empfohlen.",
		MsgNoLogging:       "HINT: Der Server unterstützt kein Logging. Server-seitige Logs helfen bei der Fehlersuche.",
		MsgTestSummary:     "\nTest-Zusammenfassung: %d Befehle ausgeführt, %d bestanden, %d fehlgeschlagen\n",
		MsgExecuting:       "Ausführung: %s %v\n",
		MsgVariableSet:     "Variable gesetzt: %s = %s\n",
		MsgExpectedError:   "Erwarteter Fehler abgefangen: %v (Code: %d)\n",
		MsgAssertionPassed: "Zusicherung bestanden: %s\n",
		MsgResource:        "Ressource: %s\n",
		MsgTemplate:        "Vorlage: %s\n",
		MsgPrompt:          "Prompt: %s\n",
		MsgTool:            "Tool: %s\n",
		MsgIcons:           "Icons:",
		MsgDescription:     "Beschreibung: %s\n",
		MsgSchema:          "Schema: %+v\n",
	},
}

func T(key MessageKey, a ...any) string {
	m, ok := messages[Lang]
	if !ok {
		m = messages["en"]
	}
	fmtStr, ok := m[key]
	if !ok {
		return string(key)
	}
	return fmt.Sprintf(fmtStr, a...)
}
