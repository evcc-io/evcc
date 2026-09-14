import Tooltip from "bootstrap/js/dist/tooltip";
import type { Directive, DirectiveBinding } from "vue";

function update(el: HTMLElement, { value, oldValue }: DirectiveBinding<string | undefined>) {
  if (value === oldValue) return;
  const instance = Tooltip.getInstance(el);
  if (!value) {
    instance?.dispose();
  } else if (instance) {
    instance.setContent({ ".tooltip-inner": value });
  } else {
    new Tooltip(el, { title: value });
  }
}

// v-tooltip="text": bootstrap tooltip bound to text, removed when text is empty
const tooltip: Directive<HTMLElement, string | undefined> = {
  mounted: update,
  updated: update,
  beforeUnmount(el) {
    Tooltip.getInstance(el)?.dispose();
  },
};

export default tooltip;
