export default function escapeHtml(s: string): string {
  const el = document.createElement("div");
  el.textContent = s;
  return el.innerHTML;
}
