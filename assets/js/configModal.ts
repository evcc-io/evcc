import { reactive, watch } from "vue";
import type { Router } from "vue-router";
import Modal from "bootstrap/js/dist/modal";

export interface ModalEntry {
  name: string;
  id?: number;
  type?: string;
  choices?: string[];
}

export interface ModalResult {
  action: "added" | "updated" | "removed" | "cancelled";
  name?: string;
  id?: number;
  type?: string;
}

const configModal = reactive({
  stack: [] as ModalEntry[],
});

export type ModalFade = "left" | "right" | undefined;

let _router: Router | null = null;
const _resolvers: Array<(result: ModalResult) => void> = [];
const _modals = new Map<string, HTMLElement>();
const _dismissingViaRoute = new Set<string>();

// --- Modal element registry (called by GenericModal) ---

export function registerModal(name: string, el: HTMLElement): void {
  _modals.set(name, el);
  syncModal(name);
}

export function unregisterModal(name: string): void {
  _modals.delete(name);
  _dismissingViaRoute.delete(name);
}

// Called by GenericModal on hidden.bs.modal (user ESC/backdrop)
export function onModalHidden(name: string): void {
  if (_dismissingViaRoute.has(name)) {
    _dismissingViaRoute.delete(name);
    return;
  }
  // User dismissed via backdrop/ESC — sync route
  if (configModal.stack.some((m) => m.name === name)) {
    closeModal();
  }
}

// Reactive fade direction for a named modal
export function getModalFade(name: string): ModalFade {
  const stackLen = configModal.stack.length;
  const idx = configModal.stack.findIndex((m) => m.name === name);
  if (idx === -1) return undefined;
  if (idx === stackLen - 1 && stackLen > 1) return "right";
  if (idx < stackLen - 1) return "left";
  return undefined;
}

// --- Internal Bootstrap show/hide ---

function showElement(el: HTMLElement): void {
  const instance = Modal.getOrCreateInstance(el);
  // @ts-expect-error bs internal
  if (!el._isShown) {
    instance.show();
  }
}

function hideElement(name: string, el: HTMLElement): void {
  const instance = Modal.getInstance(el);
  if (instance) {
    // Check if modal is actually visible
    const isVisible = el.classList.contains("show") || el.style.display !== "none";
    if (isVisible) {
      _dismissingViaRoute.add(name);
      instance.hide();
    }
  }
}

function syncModal(name: string): void {
  const el = _modals.get(name);
  if (!el) {
    return;
  }
  const inStack = configModal.stack.some((m) => m.name === name);
  const onTop = isTopModal(name);
  if (inStack && onTop) {
    showElement(el);
  } else {
    hideElement(name, el);
  }
}

function syncAllModals(): void {
  for (const name of _modals.keys()) {
    syncModal(name);
  }
}

// Parse brackets: "meter[type:grid]" => { name: "meter", type: "grid" }
// "meter[choices:pv,battery]" => { name: "meter", choices: ["pv", "battery"] }
export function parseKey(key: string): { name: string; type?: string; choices?: string[] } {
  const bracketMatch = key.match(/^([^[]+)\[([^\]]+)\]$/);
  if (!bracketMatch) {
    return { name: key };
  }
  const name = bracketMatch[1]!;
  const inner = bracketMatch[2]!;
  // inner is "type:grid" or "choices:pv,battery"
  const colonIdx = inner.indexOf(":");
  if (colonIdx === -1) {
    return { name };
  }
  const paramKey = inner.substring(0, colonIdx);
  const paramValue = inner.substring(colonIdx + 1);
  if (paramKey === "type") {
    return { name, type: paramValue };
  }
  if (paramKey === "choices") {
    return { name, choices: paramValue.split(",") };
  }
  return { name };
}

// Parse raw query string into ordered stack entries
export function parseQueryString(queryString: string): ModalEntry[] {
  if (!queryString) return [];
  const entries: ModalEntry[] = [];
  const parts = queryString.split("&");
  for (const part of parts) {
    const eqIdx = part.indexOf("=");
    let key: string;
    let value: string | undefined;
    if (eqIdx === -1) {
      key = decodeURIComponent(part);
    } else {
      key = decodeURIComponent(part.substring(0, eqIdx));
      value = decodeURIComponent(part.substring(eqIdx + 1));
    }
    // skip non-modal query params (e.g. callbackCompleted, callbackError)
    if (key.includes("callback") || key.includes("Callback")) {
      continue;
    }
    const parsed = parseKey(key);
    const entry: ModalEntry = { name: parsed.name };
    if (value !== undefined && value !== "") {
      entry.id = parseInt(value, 10);
      if (isNaN(entry.id)) {
        entry.id = undefined;
      }
    }
    if (parsed.type) entry.type = parsed.type;
    if (parsed.choices) entry.choices = parsed.choices;
    entries.push(entry);
  }
  return entries;
}

// Build query object from stack entries for router.push
export function buildQuery(stack: ModalEntry[]): Record<string, string> {
  const query: Record<string, string> = {};
  for (const entry of stack) {
    let key = entry.name;
    if (entry.type) {
      key += `[type:${entry.type}]`;
    } else if (entry.choices?.length) {
      key += `[choices:${entry.choices.join(",")}]`;
    }
    query[key] = entry.id !== undefined ? String(entry.id) : "";
  }
  return query;
}

// Extract raw query string from fullPath
export function extractQueryString(fullPath: string): string {
  const qIdx = fullPath.indexOf("?");
  if (qIdx === -1) return "";
  return fullPath.substring(qIdx + 1);
}

export function initConfigModal(router: Router): void {
  _router = router;

  watch(
    () => configModal.stack,
    () => syncAllModals()
  );

  router.afterEach((to) => {
    if (to.path !== "/config") {
      // Clear stack when leaving config page
      if (configModal.stack.length > 0) {
        // Resolve any pending promises
        while (_resolvers.length > 0) {
          const resolve = _resolvers.pop();
          resolve?.({ action: "cancelled" });
        }
        configModal.stack = [];
      }
      return;
    }

    const newStack = parseQueryString(extractQueryString(to.fullPath));
    const oldStack = configModal.stack;

    // Resolve promises for modals that were removed from stack (browser back, etc.)
    if (newStack.length < oldStack.length) {
      const removed = oldStack.length - newStack.length;
      for (let i = 0; i < removed; i++) {
        const resolve = _resolvers.pop();
        resolve?.({ action: "cancelled" });
      }
    }

    configModal.stack = newStack;
  });
}

export function openModal(
  name: string,
  params?: { id?: number; type?: string; choices?: string[] }
): Promise<ModalResult> {
  if (!_router) {
    return Promise.resolve({ action: "cancelled" });
  }

  const entry: ModalEntry = { name };
  if (params?.id !== undefined) entry.id = params.id;
  if (params?.type) entry.type = params.type;
  if (params?.choices) entry.choices = params.choices;

  const newStack = [...configModal.stack, entry];
  const query = buildQuery(newStack);

  return new Promise<ModalResult>((resolve) => {
    _resolvers.push(resolve);
    _router!.push({ path: "/config", query });
  });
}

export async function closeModal(result?: ModalResult): Promise<void> {
  if (!_router) {
    return;
  }
  if (configModal.stack.length === 0) {
    return;
  }

  const resolve = _resolvers.pop();
  const currentModal = configModal.stack[configModal.stack.length - 1];
  const newStack = configModal.stack.slice(0, -1);
  const query = buildQuery(newStack);

  // Merge type from modal stack into result if not provided
  const finalResult = result ?? { action: "cancelled" };
  if (currentModal?.type && !finalResult.type) {
    finalResult.type = currentModal.type;
  }

  // Update stack synchronously to prevent double-close from GenericModal's handleHidden
  configModal.stack = newStack;

  await _router.push({ path: "/config", query });
  resolve?.(finalResult);
}

export function replaceModal(
  name: string,
  params?: { id?: number; type?: string; choices?: string[] }
): void {
  if (!_router) return;

  const entry: ModalEntry = { name };
  if (params?.id !== undefined) entry.id = params.id;
  if (params?.type) entry.type = params.type;
  if (params?.choices) entry.choices = params.choices;

  const newStack = [...configModal.stack.slice(0, -1), entry];
  const query = buildQuery(newStack);

  _router.replace({ path: "/config", query });
}

export function getModal(name: string): ModalEntry | undefined {
  return configModal.stack.find((m) => m.name === name);
}

export function topModal(): ModalEntry | undefined {
  return configModal.stack[configModal.stack.length - 1];
}

export function isTopModal(name: string): boolean {
  return topModal()?.name === name;
}

export default configModal;
