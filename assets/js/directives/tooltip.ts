import Tooltip from "bootstrap/js/dist/tooltip";
import type { Directive, DirectiveBinding } from "vue";

type Content = string | string[][] | undefined;

// rows render as a table, first column left and the rest right aligned. Built from
// DOM nodes, never markup
function element(rows: string[][]): HTMLElement {
  const table = document.createElement("table");
  for (const cells of rows) {
    const tr = document.createElement("tr");
    cells.forEach((text, i) => {
      const td = document.createElement("td");
      td.className = i ? "text-end text-nowrap ps-3" : "text-start text-nowrap";
      td.textContent = text;
      tr.append(td);
    });
    table.append(tr);
  }
  return table;
}

function update(el: HTMLElement, { value, oldValue }: DirectiveBinding<Content>) {
  if (value === oldValue) return;
  const instance = Tooltip.getInstance(el);
  const title = Array.isArray(value) ? element(value) : value;
  if (!title) {
    instance?.dispose();
  } else if (instance) {
    instance.setContent({ ".tooltip-inner": title });
  } else {
    new Tooltip(el, {
      title,
      html: true,
      customClass: Array.isArray(value) ? "tooltip-table" : "",
    });
  }
}

// v-tooltip="text" or v-tooltip="[[name, ...cells], ...]": bootstrap tooltip, removed when empty
const tooltip: Directive<HTMLElement, Content> = {
  mounted(el, binding) {
    update(el, binding);
    // the focus a click leaves behind would keep the tooltip open after the page moved
    // on; bootstrap shows on a zero timeout, so the hide has to run after that
    el.addEventListener("click", () => setTimeout(() => Tooltip.getInstance(el)?.hide()));
  },
  updated: update,
  beforeUnmount(el) {
    Tooltip.getInstance(el)?.dispose();
  },
};

export default tooltip;
