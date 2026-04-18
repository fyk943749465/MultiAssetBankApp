# -*- coding: utf-8 -*-
"""Render docs/大庙捐款支持公示.md to HTML and save a full-page PNG screenshot."""

from __future__ import annotations

import re
from pathlib import Path

import markdown
from playwright.sync_api import sync_playwright

DOCS = Path(__file__).resolve().parent
MD_FILE = DOCS / "大庙捐款支持公示.md"
HTML_FILE = DOCS / "_preview_大庙捐款支持公示.html"
PNG_FILE = DOCS / "大庙捐款支持公示-预览.png"


def _wrap_html(body: str, title: str) -> str:
    return f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{title}</title>
  <style>
    * {{ box-sizing: border-box; }}
    body {{
      margin: 0 auto;
      padding: 28px 32px 48px;
      max-width: 980px;
      font-family: "Microsoft YaHei", "PingFang SC", "Noto Sans SC", SimSun, sans-serif;
      font-size: 15px;
      line-height: 1.65;
      color: #1a1a1a;
      background: #fafafa;
    }}
    h1 {{ font-size: 1.75rem; margin: 0 0 1rem; border-bottom: 2px solid #333; padding-bottom: 0.35rem; }}
    h2 {{ font-size: 1.25rem; margin: 1.35rem 0 0.6rem; color: #222; }}
    h3 {{ font-size: 1.1rem; margin: 1.1rem 0 0.5rem; }}
    hr {{ border: none; border-top: 1px solid #ccc; margin: 1.25rem 0; }}
    table {{
      width: 100%;
      border-collapse: collapse;
      margin: 0.75rem 0 1.25rem;
      background: #fff;
      box-shadow: 0 1px 3px rgba(0,0,0,.06);
    }}
    th, td {{
      border: 1px solid #c8c8c8;
      padding: 8px 10px;
      vertical-align: top;
    }}
    th {{ background: #f0f0f0; font-weight: 600; text-align: left; }}
    table:not(.signature-names) td:last-child,
    table:not(.signature-names) th:last-child {{
      text-align: right;
      white-space: nowrap;
    }}
    table:not(.signature-names) tr:nth-child(even) td {{ background: #fcfcfc; }}
    /* 签字人名 3×6：无格线、无斑马底、与页面底同色、字距收紧 */
    table.signature-names {{
      width: auto;
      margin: 0.45rem auto 0.85rem;
      background: transparent !important;
      box-shadow: none !important;
    }}
    table.signature-names td {{
      border: none !important;
      padding: 0 3px !important;
      background: transparent !important;
      vertical-align: middle;
      text-align: center !important;
      line-height: 1.15;
    }}
    table.signature-names tr:nth-child(even) td {{
      background: transparent !important;
    }}
    strong {{ font-weight: 600; }}
    .meta {{ color: #555; font-size: 13px; margin-bottom: 1rem; }}
  </style>
</head>
<body>
{body}
</body>
</html>
"""


def main() -> None:
    md_text = MD_FILE.read_text(encoding="utf-8")
    body = markdown.markdown(
        md_text,
        extensions=["tables", "nl2br", "sane_lists"],
        extension_configs={},
    )
    # 允许文内 HTML（如 <div>/<span>）；若被转义则还原常见实体
    if "&lt;div" in body or "&lt;span" in body:
        body = re.sub(r"&lt;(\/?)(div|span)([^&]*)&gt;", r"<\1\2\3>", body)

    html = _wrap_html(body, "大庙捐款支持公示")
    HTML_FILE.write_text(html, encoding="utf-8")

    uri = HTML_FILE.resolve().as_uri()
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=True)
            page = browser.new_page(
                viewport={"width": 1024, "height": 720},
                device_scale_factor=2,
            )
            page.goto(uri, wait_until="load")
            page.wait_for_timeout(500)
            page.screenshot(path=str(PNG_FILE), full_page=True)
            browser.close()
    finally:
        HTML_FILE.unlink(missing_ok=True)

    print("Wrote", PNG_FILE)


if __name__ == "__main__":
    main()
