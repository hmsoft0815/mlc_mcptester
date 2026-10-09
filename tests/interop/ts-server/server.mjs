// Reference server on the TypeScript SDK v2 (@modelcontextprotocol/server),
// a second, independent implementation of protocol 2026-07-28 next to the
// go-sdk test server. mcp-tester runs against both: findings here that the
// go server does not show point at the tester, the SDK, or this server.
//
// Usage: node server.mjs                     (stdio)
//        node server.mjs --http 127.0.0.1:18090   (Streamable HTTP at /mcp)

import { createServer } from "node:http";
import { parseArgs } from "node:util";
import { localhostHostValidation, localhostOriginValidation, toNodeHandler } from "@modelcontextprotocol/node";
import { completable, createMcpHandler, McpServer, ResourceTemplate } from "@modelcontextprotocol/server";
import { serveStdio } from "@modelcontextprotocol/server/stdio";
import { z } from "zod";

const ICON = {
  src: "data:image/svg+xml;base64," + Buffer.from('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><circle cx="8" cy="8" r="7" fill="#3178c6"/></svg>').toString("base64"),
  mimeType: "image/svg+xml",
};

/** Builds one server instance; the serving entries call it per request (HTTP) or connection (stdio). */
function build() {
  const server = new McpServer(
    { name: "mcp-tester-ts-reference", version: "1.0.0", title: "mcp-tester TS reference", icons: [ICON] },
    { instructions: "Reference server for mcp-tester on the TypeScript SDK v2." },
  );

  server.registerTool(
    "echo",
    {
      title: "Echo",
      description: "Echoes the input back to the user",
      icons: [ICON],
      annotations: { readOnlyHint: true },
      inputSchema: z.object({ message: z.string().describe("The text to echo") }),
      outputSchema: z.object({ echo: z.string().describe("The echoed text") }),
    },
    async ({ message }) => ({
      content: [{ type: "text", text: `Echo: ${message}` }],
      structuredContent: { echo: message },
    }),
  );

  server.registerTool(
    "add",
    {
      title: "Add Numbers",
      description: "Adds two numbers together",
      icons: [ICON],
      annotations: { readOnlyHint: true },
      inputSchema: z.object({
        a: z.number().int().describe("The first number"),
        b: z.number().int().describe("The second number"),
      }),
      outputSchema: z.object({
        sum: z.number().int().describe("The sum of a and b"),
        a: z.number().int().describe("The first number"),
        b: z.number().int().describe("The second number"),
      }),
    },
    async ({ a, b }) => ({
      content: [{ type: "text", text: `${a} + ${b} = ${a + b}` }],
      structuredContent: { sum: a + b, a, b },
    }),
  );

  server.registerResource(
    "server-info",
    "info://server",
    { title: "Server Info", description: "Name and SDK of this server", mimeType: "text/plain", icons: [ICON] },
    async (uri) => ({ contents: [{ uri: uri.href, mimeType: "text/plain", text: "mcp-tester TS reference on @modelcontextprotocol/server 2.3.1" }] }),
  );

  server.registerResource(
    "greeting",
    new ResourceTemplate("greeting://{name}", { list: undefined }),
    { title: "Greeting", description: "A greeting for the given name", mimeType: "text/plain", icons: [ICON] },
    async (uri, { name }) => ({ contents: [{ uri: uri.href, mimeType: "text/plain", text: `Hello, ${name}!` }] }),
  );

  server.registerPrompt(
    "code_review",
    {
      title: "Code Review",
      description: "Asks for a review of code in the given language",
      icons: [ICON],
      argsSchema: z.object({
        language: completable(z.string().describe("Programming language"), (value) =>
          ["typescript", "javascript", "go", "python", "rust"].filter((l) => l.startsWith(value)),
        ),
      }),
    },
    ({ language }) => ({
      messages: [{ role: "user", content: { type: "text", text: `Please review this ${language} code.` } }],
    }),
  );

  return server;
}

const { values: args } = parseArgs({ options: { http: { type: "string" } } });

if (args.http) {
  const [host, port] = args.http.split(":");
  const handle = toNodeHandler(createMcpHandler(build));
  const validateHost = localhostHostValidation();
  const validateOrigin = localhostOriginValidation();
  createServer((req, res) => {
    if (!validateHost(req, res) || !validateOrigin(req, res)) return;
    if (new URL(req.url, "http://localhost").pathname !== "/mcp") {
      res.writeHead(404).end();
      return;
    }
    handle(req, res);
  }).listen(Number(port), host, () => console.error(`TS reference server on http://${host}:${port}/mcp`));
} else {
  serveStdio(build);
}
