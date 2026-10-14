import snarkdown from "snarkdown";

// snarkdown has no tables, they are converted before it sees the text. A table needs
// a leading and trailing pipe, that keeps prose containing a pipe from matching.
const TABLE_ROW = /^\s*\|.*\|\s*$/;
const TABLE_DELIMITER = /^\s*\|[\s:|-]*-[\s:|-]*\|\s*$/;

type Segment = { table: boolean; lines: string[] };

const splitCells = (row: string): string[] =>
  row
    .trim()
    .replace(/^\|/, "")
    .replace(/\|$/, "")
    .split("|")
    .map((cell) => cell.trim());

const renderRow = (cells: string[], tag: "th" | "td"): string =>
  `<tr>${cells.map((cell) => `<${tag}>${snarkdown(cell)}</${tag}>`).join("")}</tr>`;

const renderTable = (lines: string[]): string => {
  const [head, , ...body] = lines;
  const columns = splitCells(head).length;

  // a short row is padded, a long one truncated, browsers render ragged tables badly
  const cells = (row: string) => {
    const res = splitCells(row).slice(0, columns);
    while (res.length < columns) res.push("");
    return res;
  };

  return (
    `<table><thead>${renderRow(cells(head), "th")}</thead>` +
    `<tbody>${body.map((row) => renderRow(cells(row), "td")).join("")}</tbody></table>`
  );
};

// segments splits the text into markdown and table blocks. Fenced code is passed
// through, its pipes are not table syntax.
const segments = (markdown: string): Segment[] => {
  const res: Segment[] = [];
  const lines = markdown.split("\n");
  let fence = "";

  const push = (table: boolean, line: string) => {
    const last = res[res.length - 1];
    if (last && last.table === table) {
      last.lines.push(line);
    } else {
      res.push({ table, lines: [line] });
    }
  };

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();

    if (fence) {
      if (trimmed.startsWith(fence)) fence = "";
      push(false, line);
      continue;
    }

    if (trimmed.startsWith("```") || trimmed.startsWith("~~~")) {
      fence = trimmed.slice(0, 3);
      push(false, line);
      continue;
    }

    if (TABLE_ROW.test(line) && TABLE_DELIMITER.test(lines[i + 1] || "")) {
      const table = [line, lines[i + 1]];
      let j = i + 2;
      while (j < lines.length && TABLE_ROW.test(lines[j])) table.push(lines[j++]);

      res.push({ table: true, lines: table });
      i = j - 1;
      continue;
    }

    push(false, line);
  }

  return res;
};

// renderMarkdown converts markdown to html, links open in a new window
export function renderMarkdown(markdown: string): string {
  const html = segments(markdown)
    .map((segment) =>
      segment.table ? renderTable(segment.lines) : snarkdown(segment.lines.join("\n"))
    )
    .join("");

  return html.replace(/<a href=/g, '<a target="_blank" rel="noopener noreferrer" href=');
}
