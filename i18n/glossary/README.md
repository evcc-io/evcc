# Glossary

Agreed terminology for the evcc UI, the app, the docs and device templates.
One entry per concept, not per UI string.
The same English word may appear under several keys when it covers several concepts (range, charge).
If the language has only one word for them, repeat it.
A concept that needs a shorter form for tight spaces, such as the tab bar, gets a second key with the suffix `Short`.
Repeat the term if the language needs no short form.

- `en.json`: English term plus a short explanation. The English term is the match key in Weblate and the target wording for `i18n/en.json`; strings that still differ get renamed to match.
- `<lang>.json`: mapping from key to the agreed term in that language.
- Terms are stored as written inside a sentence, in base form: singular, no article. Capitalize only where the language's grammar does. Notes on gender or article belong in the language section below.

Weblate shows matching terms and the explanation in the editor sidebar for every component of the project. Edit terms in [Weblate](https://hosted.weblate.org/projects/evcc/glossary/) or via pull request. English and German entries change via pull request only, other languages via Weblate.

This file holds the rules the term list cannot express: tone and spelling. Words to avoid are flagged as forbidden in Weblate.

## Rules for all languages

- Acronyms are uppercase in prose. Code-only acronyms such as SoC and HEMS never appear in UI text.
- Mention evcc only when needed for context. Inside the UI the self-reference is usually redundant.
- Keep separators and trailing punctuation (`: `, `…`) in the template, not in the translation.
- Drop a prefix when the surrounding card or dialog already gives the context.
- Prefer neutral, impersonal phrasing. Address the user directly only when they need to take action.

## Technical terms

Protocol and standard names are not glossary entries.
They keep their official spelling in every language.

- EEBUS: formerly spelled EEBus, now uppercase ([eebus.org](https://www.eebus.org/), [Wikipedia](https://de.wikipedia.org/wiki/EEBUS)). Mixed spellings are still common online.
- OCPP
- MQTT
- Modbus
- SunSpec
- SG Ready
- SEMP
- RFID

## English

- Form of address: "you".
- UK spelling.

## Deutsch

- Anrede: "du", kleingeschrieben.
- Komposita mit Bindestrich nach Duden.

## Adding a language section

Add a section when a language needs rules beyond the term mapping, typically the form of address (formal or informal), compound spelling, or terms that stay untranslated.
