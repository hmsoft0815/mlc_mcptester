// Interop test: the official TypeScript Tasks client
// (@modelcontextprotocol/ext-tasks) drives tools of the mcp-tester reference
// server (pkg/mcptasks) over stdio. It checks pkg/mcptasks against an
// independent reading of the spec, and with it the assumptions mcp-tester
// makes. No SDK client sits in between: transport.mjs frames each request
// for protocol 2026-07-28 itself.
//
// Usage: node interop.mjs --stdio <server command>
//        node interop.mjs --http <url> [--bearer <token>]

import { parseArgs } from "node:util";
import {
  createApplicationInputHandler,
  resultFromTaskOutcome,
  withTasks,
} from "@modelcontextprotocol/ext-tasks/client";
import { connectHTTP, connectStdio } from "./transport.mjs";

const TASKS = "io.modelcontextprotocol/tasks";

const { values: args } = parseArgs({
  options: { stdio: { type: "string" }, http: { type: "string" }, bearer: { type: "string" } },
});
if (!args.stdio === !args.http) {
  console.error("usage: node interop.mjs --stdio <server command> | --http <url> [--bearer <token>]");
  process.exit(2);
}

const failures = [];
async function check(name, fn) {
  try {
    await fn();
    console.log(`[PASS] ${name}`);
  } catch (err) {
    failures.push(name);
    console.log(`[FAIL] ${name}: ${err?.message ?? err}`);
  }
}
function expect(cond, message) {
  if (!cond) throw new Error(message);
}
const text = (result) => result?.content?.[0]?.text;

const conn = args.stdio
  ? connectStdio(args.stdio)
  : connectHTTP(args.http, args.bearer ? { Authorization: `Bearer ${args.bearer}` } : {});
console.log(`--- ${args.stdio ? `stdio: ${args.stdio}` : `Streamable HTTP: ${args.http}`}`);
const discover = await conn.dispatch({ method: "server/discover", params: {} });
if (discover.kind !== "result" || !discover.result.capabilities?.extensions?.[TASKS]) {
  console.log("[FAIL] server/discover: the server does not declare the Tasks extension", JSON.stringify(discover));
  conn.close();
  process.exit(1);
}

const port = {
  endpointId: "mcp-tester-reference",
  taskCapabilities: { generation: "v2", capabilities: discover.result.capabilities.extensions[TASKS] },
  dispatch: conn.dispatch,
  onServerRequest: () => () => {},
  onNotification: (l) => conn.onNotification(l),
  onInvalidated: (l) => conn.onInvalidated(l),
  get invalidated() {
    return conn.invalidated;
  },
};
const inputErrors = [];
const session = withTasks(port, {
  onInputRequest: createApplicationInputHandler({
    elicitation: async () => ({ action: "accept", content: { name: "Ada" } }),
  }),
  onError: (err) => inputErrors.push(err),
});

try {
  await check("long_job runs as a task and completes", async () => {
    const execution = await session.callTool("long_job", { seconds: 0.3 });
    expect(execution.kind === "task", `expected a task, got ${execution.kind}`);
    const { outcome } = await execution.settle();
    expect(outcome.status === "completed", `status ${outcome.status}`);
    expect(text(resultFromTaskOutcome(outcome)) === "Job finished after 300ms", `result ${JSON.stringify(outcome.result)}`);
  });

  await check("ask_name: input_required answered via tasks/update", async () => {
    const execution = await session.callTool("ask_name", {});
    expect(execution.kind === "task", `expected a task, got ${execution.kind}`);
    const statuses = [];
    const { outcome } = await execution.settle({ onEvent: (e) => e.type === "task" && statuses.push(e.task.status) });
    expect(statuses.includes("input_required"), `never input_required: ${statuses.join(" → ")}`);
    expect(outcome.status === "completed", `status ${outcome.status}`);
    expect(text(outcome.result) === "Hello, Ada!", `result ${JSON.stringify(outcome.result)}`);
    expect(inputErrors.length === 0, `input errors: ${inputErrors.map(String).join("; ")}`);
  });

  await check("failing_job ends failed with the JSON-RPC error", async () => {
    const execution = await session.callTool("failing_job", {});
    const { outcome } = await execution.settle();
    expect(outcome.status === "failed", `status ${outcome.status}`);
    expect(outcome.error?.code === -32603, `error ${JSON.stringify(outcome.error)}`);
  });

  await check("tool_error_job ends completed with isError", async () => {
    const execution = await session.callTool("tool_error_job", {});
    const { outcome } = await execution.settle();
    expect(outcome.status === "completed", `status ${outcome.status}`);
    expect(outcome.result?.isError === true, `result ${JSON.stringify(outcome.result)}`);
  });

  await check("cancel reaches the server", async () => {
    const execution = await session.callTool("long_job", { seconds: 30 });
    expect(execution.kind === "task", `expected a task, got ${execution.kind}`);
    const taskId = execution.handle.taskId;
    await execution.cancel();
    // cancel() releases local ownership; ask the server directly
    const res = await conn.dispatch({
      method: "tasks/get",
      params: { taskId, _meta: { "io.modelcontextprotocol/clientCapabilities": { extensions: { [TASKS]: {} } } } },
    });
    expect(res.kind === "result" && res.result.status === "cancelled", `server state ${JSON.stringify(res)}`);
  });

  await check("a tool that is not task-capable answers immediately", async () => {
    const execution = await session.callTool("echo", { message: "hi" });
    expect(execution.kind !== "task", `expected an immediate result, got ${execution.kind}`);
    const { outcome } = await execution.settle();
    expect(outcome.status === "completed", `status ${outcome.status}`);
  });
} finally {
  await session.close();
  conn.close();
}

console.log(failures.length ? `\n${failures.length} check(s) failed` : "\nall checks passed");
process.exit(failures.length ? 1 : 0);
