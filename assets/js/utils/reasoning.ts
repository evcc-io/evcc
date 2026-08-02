// models leak the special tokens of their chat template into the reasoning text,
// e.g. `<|tool_call|>{"name":"getState"}`. The calls are listed separately anyway
const SPECIAL_TOKEN = /<\|([^|]*)\|>/g;
const CALL_TOKEN = /tool|call|function/i;

// cleanReasoning removes the special tokens and the tool calls spelled out between them
export function cleanReasoning(text: string): string {
  const parts: string[] = [];
  let last = 0;
  let skip = false;

  for (const match of text.matchAll(SPECIAL_TOKEN)) {
    if (!skip) parts.push(text.slice(last, match.index));
    skip = CALL_TOKEN.test(match[1] as string);
    last = match.index + match[0].length;
  }

  if (!skip) parts.push(text.slice(last));

  return parts
    .join("")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}
