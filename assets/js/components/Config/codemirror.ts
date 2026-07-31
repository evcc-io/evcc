// Separate module so YamlEditor.vue can dynamic-import it, keeping CodeMirror out of the main chunk.
import { EditorState, Compartment, StateField, StateEffect } from "@codemirror/state";
import {
  EditorView,
  lineNumbers,
  keymap,
  drawSelection,
  Decoration,
  type DecorationSet,
} from "@codemirror/view";
import {
  indentOnInput,
  foldGutter,
  syntaxHighlighting,
  bracketMatching,
} from "@codemirror/language";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
import { yaml } from "@codemirror/lang-yaml";
import { classHighlighter } from "@lezer/highlight";
import { cleanYaml } from "@/utils/cleanYaml";

// 1-based line, null clears
const setErrorLineEffect = StateEffect.define<number | null>();

const errorLineField = StateField.define<number | null>({
  create: () => null,
  update(value, tr) {
    let next = value;
    for (const e of tr.effects) {
      if (e.is(setErrorLineEffect)) next = e.value;
    }
    return next;
  },
});

function errorLineFrom(state: EditorState) {
  const line = state.field(errorLineField);
  if (!line || line < 1 || line > state.doc.lines) return null;
  return state.doc.line(line);
}

const errorLineExtension = [
  errorLineField,
  EditorView.decorations.compute([errorLineField, "doc"], (state): DecorationSet => {
    const line = errorLineFrom(state);
    if (!line) return Decoration.none;
    return Decoration.set([Decoration.line({ class: "cm-errorLine" }).range(line.from)]);
  }),
];

export interface EditorOptions {
  parent: HTMLElement;
  doc: string;
  readOnly: boolean;
  onChange: (value: string) => void;
  getRemoveKey: () => string | undefined;
}

export interface EditorController {
  view: EditorView;
  setDoc(text: string): void;
  setErrorLine(line: number | null): void;
  setReadOnly(readOnly: boolean): void;
  destroy(): void;
}

export function createEditor(opts: EditorOptions): EditorController {
  const readOnlyCompartment = new Compartment();
  const readOnlyExt = (readOnly: boolean) => [
    EditorState.readOnly.of(readOnly),
    EditorView.editable.of(!readOnly),
  ];

  const holder: { view?: EditorView } = {};

  const state = EditorState.create({
    doc: opts.doc ?? "",
    extensions: [
      lineNumbers(),
      foldGutter(),
      drawSelection(),
      history(),
      bracketMatching(),
      yaml(),
      indentOnInput(),
      keymap.of([indentWithTab, ...defaultKeymap, ...historyKeymap]),
      errorLineExtension,
      // emits stable tok-* classes so colors/theme live in YamlEditor.vue CSS
      syntaxHighlighting(classHighlighter),
      readOnlyCompartment.of(readOnlyExt(opts.readOnly)),
      EditorView.updateListener.of((update) => {
        if (!update.docChanged) return;
        const value = update.state.doc.toString();
        // strip a pasted wrapper key (e.g. "power:") and unindent, matching the editor's removeKey
        const pasted = update.transactions.some((tr) => tr.isUserEvent("input.paste"));
        const removeKey = pasted ? opts.getRemoveKey() : undefined;
        if (removeKey) {
          const cleaned = cleanYaml(value, removeKey);
          if (cleaned !== value) {
            // dispatch outside the update cycle to avoid a re-entrant transaction
            queueMicrotask(() => {
              const view = holder.view;
              if (!view) return;
              view.dispatch({
                changes: { from: 0, to: view.state.doc.length, insert: cleaned },
              });
            });
            return;
          }
        }
        opts.onChange(value);
      }),
    ],
  });

  const view = new EditorView({ state, parent: opts.parent });
  holder.view = view;

  return {
    view,
    setDoc(text: string) {
      if (view.state.doc.toString() === text) return;
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
    },
    setErrorLine(line: number | null) {
      view.dispatch({ effects: setErrorLineEffect.of(line || null) });
    },
    setReadOnly(readOnly: boolean) {
      view.dispatch({ effects: readOnlyCompartment.reconfigure(readOnlyExt(readOnly)) });
    },
    destroy() {
      view.destroy();
    },
  };
}
