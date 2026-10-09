// Minimal JSON-RPC connections for protocol 2026-07-28, over stdio or
// Streamable HTTP. No SDK client: each request is framed here, so what the
// server sees comes from @modelcontextprotocol/ext-tasks and this file only.

import { spawn } from "node:child_process";
import { createInterface } from "node:readline";

export const PROTOCOL = "2026-07-28";
const CLIENT_INFO = { name: "mcp-tester-interop", version: "1" };
const CLIENT_CAPS = { elicitation: { form: {} } };
const CAPS_KEY = "io.modelcontextprotocol/clientCapabilities";

/** Adds the per-request _meta of protocol 2026-07-28, keeping what the caller set. */
function frame(params = {}) {
  const meta = params._meta ?? {};
  return {
    ...params,
    _meta: {
      ...meta,
      "io.modelcontextprotocol/protocolVersion": PROTOCOL,
      "io.modelcontextprotocol/clientInfo": CLIENT_INFO,
      [CAPS_KEY]: { ...CLIENT_CAPS, ...(meta[CAPS_KEY] ?? {}) },
    },
  };
}

function toResponse(msg) {
  return msg.error ? { kind: "error", error: msg.error } : { kind: "result", result: msg.result };
}

/** Listener sets shared by both transports. */
function listeners() {
  const notification = new Set();
  const invalidation = new Set();
  const state = { invalidated: false };
  return {
    state,
    notify: (msg) => notification.forEach((l) => l(msg)),
    invalidate: (reason) => {
      state.invalidated = true;
      invalidation.forEach((l) => l(reason));
    },
    port: {
      onNotification(l) {
        notification.add(l);
        return () => notification.delete(l);
      },
      onInvalidated(l) {
        invalidation.add(l);
        return () => invalidation.delete(l);
      },
      get invalidated() {
        return state.invalidated;
      },
    },
  };
}

/** A connection to a server process over stdio (newline-delimited JSON). */
export function connectStdio(command) {
  const child = spawn(command, [], { stdio: ["pipe", "pipe", "inherit"], shell: true });
  const ls = listeners();
  const pending = new Map();
  let nextId = 1;
  const send = (msg) => child.stdin.write(JSON.stringify(msg) + "\n");

  createInterface({ input: child.stdout }).on("line", (line) => {
    const msg = JSON.parse(line);
    if (msg.id !== undefined && !msg.method && pending.has(msg.id)) {
      pending.get(msg.id)(toResponse(msg));
      pending.delete(msg.id);
    } else if (msg.method && msg.id === undefined) {
      ls.notify(msg);
    } else if (msg.method) {
      // Server-to-client requests are not used by these tools
      send({ jsonrpc: "2.0", id: msg.id, error: { code: -32601, message: "not supported" } });
    }
  });
  child.on("exit", () => ls.invalidate(new Error("server exited")));

  return {
    ...ls.port,
    dispatch(request, options = {}) {
      const id = nextId++;
      return new Promise((resolve, reject) => {
        pending.set(id, resolve);
        options.signal?.addEventListener("abort", () => {
          pending.delete(id);
          reject(options.signal.reason);
        });
        send({ jsonrpc: "2.0", id, method: request.method, params: frame(request.params) });
      });
    },
    close() {
      child.stdin.end();
      child.kill();
    },
  };
}

// Methods whose Mcp-Name header mirrors a body field (2026-07-28 standard
// headers; tasks/* per the Tasks extension's Streamable HTTP binding).
const NAME_SOURCE = {
  "tools/call": "name",
  "prompts/get": "name",
  "resources/read": "uri",
  "tasks/get": "taskId",
  "tasks/update": "taskId",
  "tasks/cancel": "taskId",
};

/** Reads a Streamable HTTP response: plain JSON, or SSE events up to the answer to id. */
async function readResponse(res, id, notify) {
  const type = res.headers.get("content-type") ?? "";
  if (type.startsWith("application/json")) return toResponse(await res.json());
  if (!type.startsWith("text/event-stream")) throw new Error(`HTTP ${res.status} with content type ${type || "none"}`);
  const decoder = new TextDecoder();
  let buffer = "";
  for await (const chunk of res.body) {
    buffer += decoder.decode(chunk, { stream: true });
    let end;
    while ((end = buffer.indexOf("\n\n")) >= 0) {
      const event = buffer.slice(0, end);
      buffer = buffer.slice(end + 2);
      const data = event
        .split("\n")
        .filter((l) => l.startsWith("data:"))
        .map((l) => l.slice(5).trimStart())
        .join("\n");
      if (!data) continue;
      const msg = JSON.parse(data);
      if (msg.id === id && !msg.method) {
        await res.body.cancel().catch(() => {});
        return toResponse(msg);
      }
      if (msg.method && msg.id === undefined) notify(msg);
    }
  }
  throw new Error(`the SSE stream ended without an answer to request ${id}`);
}

/** A connection to a Streamable HTTP endpoint; headers are added to every request (e.g. Authorization). */
export function connectHTTP(url, headers = {}) {
  const ls = listeners();
  let nextId = 1;
  return {
    ...ls.port,
    async dispatch(request, options = {}) {
      const id = nextId++;
      const params = frame(request.params);
      const h = {
        ...headers,
        "Content-Type": "application/json",
        Accept: "application/json, text/event-stream",
        "MCP-Protocol-Version": PROTOCOL,
        "Mcp-Method": request.method,
        ...(options.context?.headers ?? {}),
      };
      const field = NAME_SOURCE[request.method];
      if (field && typeof params[field] === "string") h["Mcp-Name"] = params[field];
      const res = await fetch(url, {
        method: "POST",
        headers: h,
        body: JSON.stringify({ jsonrpc: "2.0", id, method: request.method, params }),
        signal: options.signal,
      });
      if (!res.ok && !(res.headers.get("content-type") ?? "").startsWith("application/json")) {
        throw new Error(`HTTP ${res.status}: ${(await res.text()).trim()}`);
      }
      return readResponse(res, id, ls.notify);
    },
    close() {},
  };
}
