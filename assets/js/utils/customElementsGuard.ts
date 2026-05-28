// Side-effect module: must be imported BEFORE any code that defines custom
// elements (e.g. before shopicons-style web-component imports).
//
// Some browsers re-execute the page's scripts a second time after stripping
// HTTP Basic Auth credentials from a `https://user:pw@host/...` URL (observed
// in Brave/Chromium and the Tesla MCU built-in browser). The same bundle is
// fetched and run twice in the same document, which means every
// `customElements.define(...)` call from the second execution throws a
// `NotSupportedError: "<name>" has already been used with this registry`.
// That uncaught error aborts framework initialization and the UI never renders
// past the static shell.
//
// Wrap the registry's `define` to silently skip names that are already
// registered. The first registration wins; further definitions of the same
// name are no-ops. Behaviour for new names is unchanged.

const WRAPPED = Symbol.for("evcc.customElementsGuard.wrapped");

if (
  typeof window !== "undefined" &&
  "customElements" in window &&
  !(customElements as unknown as Record<symbol, boolean>)[WRAPPED]
) {
  const originalDefine = customElements.define.bind(customElements);
  let warned = false;

  customElements.define = function (
    name: string,
    constructor: CustomElementConstructor,
    options?: ElementDefinitionOptions
  ): void {
    if (customElements.get(name)) {
      if (!warned) {
        console.warn(
          `customElementsGuard: suppressed duplicate registration of "${name}". ` +
            "Likely cause: browser re-executed scripts after stripping URL credentials. " +
            "Further duplicates silenced."
        );
        warned = true;
      }
      return;
    }
    originalDefine(name, constructor, options);
  };

  (customElements as unknown as Record<symbol, boolean>)[WRAPPED] = true;
}
