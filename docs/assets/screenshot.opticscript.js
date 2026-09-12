//!OUTPUT: OUTPUT
//
// Zeichnet ein Terminalfenster mit echter mcp-tester-Ausgabe.
// Der Text unten stammt aus zwei echten Laeufen gegen den
// OpticScript-MCP-Server. Gekuerzt ist nur, was nichts trug: ein
// leeres Namensfeld in der Kopfzeile, die zweite gleichlautende
// HINT-Zeile, und von den 34 Schritten des Testlaufs stehen fuenf
// stellvertretend da. Erfunden ist nichts.

const W = 1280, H = 800;
const BG = "#080b11", WIN = "#11161f", EDGE = "#243040", BAR = "#1a2230";
const FG = "#d7dee8", DIM = "#8b949e", GREEN = "#3fb950", BLUE = "#58a6ff",
      YELLOW = "#e3b341", CYAN = "#56d4dd";

const MONO = "JetBrains Mono";
const SIZE = 15, CH = SIZE * 0.6, LH = 24;

// ── Fenster ────────────────────────────────────────────────────────
const img = Engine.createImage(W, H, BG);
const cv = Engine.createCanvas(W, H);

function rounded(cvs, x, y, w, h, r, color) {
  cvs.fill(color);
  cvs.drawPath(Engine.createPath().rect(x + r, y, w - 2 * r, h));
  cvs.drawPath(Engine.createPath().rect(x, y + r, w, h - 2 * r));
  cvs.drawPath(Engine.createPath().circle(x + r, y + r, r));
  cvs.drawPath(Engine.createPath().circle(x + w - r, y + r, r));
  cvs.drawPath(Engine.createPath().circle(x + r, y + h - r, r));
  cvs.drawPath(Engine.createPath().circle(x + w - r, y + h - r, r));
}

const X = 48, Y = 48, WW = W - 2 * X, WH = H - 2 * Y;
rounded(cv, X - 1, Y - 1, WW + 2, WH + 2, 15, EDGE);
rounded(cv, X, Y, WW, WH, 14, WIN);
rounded(cv, X, Y, WW, 44, 14, BAR);
cv.fill(WIN);
cv.drawPath(Engine.createPath().rect(X, Y + 36, WW, 8));

const dots = ["#ff5f56", "#ffbd2e", "#27c93f"];
for (let i = 0; i < 3; i++) {
  cv.fill(dots[i]);
  cv.drawPath(Engine.createPath().circle(X + 24 + i * 22, Y + 22, 6));
}

const shot = cv.toImage();
img.blendAt(shot, px(0, 0), 1.0, Blend.Over);
shot.free();
cv.free();

img.drawText("mcp-tester — mlc OpticScript MCP server", W / 2, Y + 27,
  { font: MONO, size: 13, color: DIM, anchor: "middle" });

// ── Text ───────────────────────────────────────────────────────────
const TX = X + 28;
let ty = Y + 76;

function line(segments) {
  let col = 0;
  for (const seg of segments) {
    img.drawText(seg[0], TX + col * CH, ty, { font: MONO, size: SIZE, color: seg[1] });
    col += seg[0].length;
  }
  ty += LH;
}

function prompt(cmd) {
  line([["$ ", GREEN], [cmd, FG]]);
}

// Echte Ausgabe von `mcp-tester inspect`.
prompt("mcp-tester inspect -c \"bin/mlcos-mcp\"");
line([["=== MCP Server Inspection ===", CYAN]]);
line([["[✓] ", GREEN], ["Server: mlc-opticscript (0.2.0)", FG]]);
line([["[✓] ", GREEN], ["Protocol Version: 2026-07-28", FG]]);
line([["[i] ", BLUE], ["Capabilities & Features:", FG]]);
line([["    - Tools:     true (Subscription capable)", DIM]]);
line([["    - Prompts:   true (Subscription capable)", DIM]]);
line([["    - Resources: true", DIM]]);
line([["    - Logging:   true", DIM]]);
line([["    - Progress:  Supported (Protocol Standard)", DIM]]);
line([["    - Cancel:    Supported (Protocol Standard)", DIM]]);
line([["[✓] ", GREEN], ["2 prompts found.", FG]]);
line([["[✓] ", GREEN], ["10 tools found.", FG]]);
line([["[✓] ", GREEN], ["1 resources found.", FG]]);
ty += 6;
line([["--- Quality Report (Score: ", FG], ["100/100", GREEN], [") ---", FG]]);
line([["- HINT: Tool 'get_api_reference' has no output schema.", YELLOW]]);
line([["  Structured returns help the LLM process results precisely.", DIM]]);
ty += 12;

// Echte Ausgabe von `mcp-tester test`.
prompt("mcp-tester test -s test/mcp/04_analyze.mcp -c \"bin/mlcos-mcp\"");
line([["--- analyze_image: measure, sharpness, transparency ---", CYAN]]);
line([["Assertion passed: ", DIM], ["\"Canon\" == \"Canon\"", FG]]);
line([["Assertion passed: ", DIM], ["\"256\" == \"256\"", FG]]);
line([["Assertion passed: ", DIM], ["0.140564 > 0.000000", FG]]);
line([["Expected error caught: ", DIM], ["Tool error: cannot read \"gibt_es_nicht.png\"", FG]]);
line([["Assertion passed: ", DIM], ["tool error (isError: true)", FG]]);
ty += 6;
line([["Test Summary: 34 commands executed, ", FG], ["34 passed, 0 failed", GREEN]]);

img.save(OUTPUT, { format: "png" });
