# Kapitel 1: Einleitung

In diesem Kapitel legen wir das Fundament für das Verständnis von MCP (Model Context Protocol). Warum brauchen wir es überhaupt und wie haben LLMs gelernt, mit der Außenwelt zu kommunizieren?

## Was ist ein LLM?

Large Language Models (LLMs) wie GPT-4, Claude oder Llama sind im Kern statistische Modelle, die das nächste Wort in einer Sequenz vorhersagen können. Sie haben ein beeindruckendes Wissen über die Welt (aus ihren Trainingsdaten), sind aber von Natur aus "isoliert". Ein reines LLM kann keine E-Mails senden, keine aktuellen Aktienkurse abrufen und keine Dateien auf deiner Festplatte lesen.

## Die Evolution der Tool-Calls

### Die "Frühe Zeit": Manuelles Copy-Paste
Anfangs mussten Benutzer Ergebnisse von API-Abfragen manuell in den Chat kopieren. Das Modell gab Anweisungen ("Bitte suche nach X"), der Mensch führte sie aus und kopierte das Ergebnis zurück.

### Die Ära der Plugins & Function Calling
OpenAI führte 2023 "Function Calling" ein. Entwickler konnten dem Modell beschreiben, welche Funktionen (APIs) verfügbar sind. Das Modell antwortete nicht mehr mit Fließtext, sondern mit einem strukturierten JSON-Objekt: *"Ich möchte die Funktion `get_weather` mit dem Parameter `city: Berlin` aufrufen."*

Der Nachteil: Jeder Anbieter kochte sein eigenes Süppchen. Ein Plugin für ein System funktionierte nicht in einem anderen.

## Was ist MCP?

Das **Model Context Protocol (MCP)** ist der Versuch, einen universellen Standard für diese Verbindung zu schaffen. Anstatt für jedes Modell eine neue Schnittstelle zu bauen, schreiben wir einen **MCP-Server**. 

Jeder kompatible Client (wie Claude Desktop, IDEs oder unser `mcp-tester`) kann sich mit diesem Server verbinden und sofort dessen Tools, Daten (Resources) und Vorlagen (Prompts) nutzen. MCP ist für LLMs das, was USB für Computer oder HTTP für das Web ist: Ein einheitlicher Stecker für alles.

## Ziele dieses Handbuchs

Dieses Buch bietet eine praxisorientierte Einführung in die Welt von MCP. Es ist wichtig zu verstehen, dass MCP ein **stark expandierender Bereich** ist. Die Spezifikation entwickelt sich rasant weiter (wie die Neuerungen Ende 2025 mit Tasks und Sampling zeigen).

Wir beleuchten in diesem Werk die **häufigsten und wichtigsten Aspekte** eines MCP-Servers, um Ihnen einen soliden Start zu ermöglichen. Aufgrund der Dynamik des Protokolls wird dieses Buch nicht jede Nische abdecken können, aber es vermittelt die fundamentalen Muster.

Ein zentraler Punkt bei der Entwicklung ist die **strikte Einhaltung der offiziellen Spezifikation**. Da viele verschiedene Clients (KIs) auf Ihren Server zugreifen werden, führen kleinste Abweichungen im JSON-Format oder im Handshake zu Fehlern. Genau aus diesem Grund haben wir den **`mcp-tester`** entwickelt: Er hilft Ihnen dabei, Ihren Server gegen die Spec zu validieren und sicherzustellen, dass er in der "echten Welt" stabil funktioniert.


[← Inhaltsverzeichnis](README.md) | [Nächstes Kapitel: Wie LLMs kommunizieren →](02_wie_llms_kommunizieren.md)

---
*Copyright Michael Lechner - 2026-02-28*
