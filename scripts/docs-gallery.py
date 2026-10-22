"""Render the start of an mcp-tester docs reference as a highlighted editor view (HTML).

Used for the gallery image mlcprodweb/assets/docs-reference.png (task gallery-docs):
python3 scripts/docs-gallery.py <reference.md> <out.html>
"""
import html
import re
import sys

src, out = sys.argv[1], sys.argv[2]
lines = open(src, encoding="utf-8").read().split("\n")
# Head, overview table and the first tool's heading: 28 lines fill the 800 px frame.
picked = [(n + 1, lines[n]) for n in range(0, 28)]


def count(section, prefix):
    """Lines starting with prefix inside the "## section" block."""
    inside, n = False, 0
    for s in lines:
        if s.startswith("## "):
            inside = s == "## " + section
        elif inside and s.startswith(prefix):
            n += 1
    return n


def plural(n, word):
    return "%d %s%s" % (n, word, "" if n == 1 else "s")


summary = " · ".join((plural(count("Tools", "### "), "tool"),
                      plural(count("Prompts", "### "), "prompt"),
                      plural(count("Resources", "| `"), "resource")))

TOKENS = re.compile(r"`[^`]*`|\*\*[^*]+\*\*|\]\(#[^)]*\)|\[|\||\\_")


def span(cls, text):
    return '<span class="%s">%s</span>' % (cls, html.escape(text))


def inline(s):
    out, i = [], 0
    for m in TOKENS.finditer(s):
        out.append(html.escape(s[i:m.start()]))
        t = m.group(0)
        if t.startswith("`"):
            out.append(span("code", t))
        elif t.startswith("**"):
            out.append(span("bold", t))
        elif t.startswith("]("):
            out.append(span("punct", "](") + span("link", t[2:-1]) + span("punct", ")"))
        elif t == "\\_":
            out.append(span("esc", t))
        else:
            out.append(span("punct", t))
        i = m.end()
    out.append(html.escape(s[i:]))
    r = "".join(out)
    for word, cls in (("read-only", "safe"), ("idempotent", "safe"), ("destructive", "danger")):
        r = r.replace(" %s" % word, ' <span class="%s">%s</span>' % (cls, word))
    return r


def line(s):
    m = re.match(r"^(#{1,6}) ", s)
    if m:
        lvl = len(m.group(1))
        return '<span class="h%d">%s%s</span>' % (min(lvl, 3), span("hash", m.group(1)), inline(s[lvl:]))
    if s.startswith(">"):
        return '<span class="quote">%s</span>' % inline(s)
    if re.match(r"^\|[-| ]+\|$", s):
        return span("punct", s)
    return inline(s)


rows = "\n".join(
    '<div class="ln"><span class="no">%d</span><span class="tx">%s</span></div>' % (n, line(s) or "&nbsp;")
    for n, s in picked
)

page = """<!doctype html><html><head><meta charset="utf-8">
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;700&display=swap" rel="stylesheet">
<style>
html,body{margin:0;width:1280px;height:800px;background:#090c12;overflow:hidden}
.win{position:absolute;left:48px;top:48px;width:1184px;height:704px;background:#11161e;border:1px solid #232a35;border-radius:14px;overflow:hidden;
     font-family:'JetBrains Mono','DejaVu Sans Mono',monospace;font-size:14px;color:#c9d1d9}
.bar{height:36px;background:#161c26;display:flex;align-items:center;position:relative}
.dot{width:12px;height:12px;border-radius:50%;margin-left:8px}
.dot:first-child{margin-left:18px}
.title{position:absolute;left:0;right:0;text-align:center;font-size:13px;color:#8b949e}
.body{padding:16px 0 0 0}
.ln{display:flex;line-height:19.4px;white-space:pre-wrap}
.no{width:52px;text-align:right;padding-right:20px;color:#3d4654;flex:none}
.tx{overflow:hidden;padding-right:28px}
.safe{color:#7ee2a8}.danger{color:#ff7b72;font-weight:700}
.h1{color:#58d5e0;font-weight:700}.h2{color:#58d5e0;font-weight:700}.h3{color:#7cc7ff;font-weight:700}
.hash{color:#2f8b94}
.quote{color:#7d8794;font-style:italic}
.code{color:#e3b341}
.bold{color:#f0f6fc;font-weight:700}
.punct{color:#4b5563}
.link{color:#6e9ecf}
.esc{color:#7d8794}
.status{position:absolute;left:0;right:0;bottom:0;height:42px;border-top:1px solid #232a35;background:#0e131a;
        display:flex;align-items:center;padding-left:28px;font-size:14px}
.prompt{color:#3fb950;margin-right:10px}
.ok{color:#3fb950;margin-left:auto;margin-right:28px}
</style></head><body>
<div class="win">
 <div class="bar"><span class="dot" style="background:#ff5f57"></span><span class="dot" style="background:#febc2e"></span><span class="dot" style="background:#28c840"></span>
  <span class="title">docs/mcp-tools.md — mlc OpticScript MCP server</span></div>
 <div class="body">
@@ROWS@@
 </div>
 <div class="status"><span class="prompt">$</span>mcp-tester docs -c "bin/mlcos-mcp" -o docs/mcp-tools.md<span class="ok">@@SUMMARY@@</span></div>
</div></body></html>""".replace("@@ROWS@@", rows).replace("@@SUMMARY@@", summary)

open(out, "w", encoding="utf-8").write(page)
