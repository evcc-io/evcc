# Glossary

Agreed terminology for the evcc UI, the app, the docs and device templates. One entry per concept, not per UI string.

- `en.json`: English term plus a short explanation. The English term is the match key in Weblate, so it must use the exact wording of `i18n/en.json`.
- `<lang>.json`: mapping from key to the agreed term in that language.

Weblate shows matching terms and the explanation in the editor sidebar for every component of the project. Edit terms in [Weblate](https://hosted.weblate.org/projects/evcc/glossary/) or via pull request. English and German entries change via pull request only, other languages via Weblate.

This file holds the rules the term list cannot express: tone and spelling. Words to avoid are flagged as forbidden in Weblate.

## Rules for all languages

- Protocol and product names stay untranslated.
- Acronyms are uppercase in prose. Code-only acronyms such as SoC and HEMS never appear in UI text.
- Mention evcc only when needed for context. Inside the UI the self-reference is usually redundant.
- Keep separators and trailing punctuation (`: `, `…`) in the template, not in the translation.
- Drop a prefix when the surrounding card or dialog already gives the context.
- Prefer neutral, impersonal phrasing. Address the user directly only when they need to take action.

## English

- Form of address: "you".
- UK spelling.

## Deutsch

- Anrede: "du", kleingeschrieben.
- Komposita mit Bindestrich nach Duden.

## Adding a language section

Add a section when a language needs rules beyond the term mapping, typically the form of address (formal or informal), compound spelling, or terms that stay untranslated.
