# Glossary

Agreed terminology for the evcc UI, the app, the docs, device templates and API descriptions.
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

## Български

- Обръщение: учтива форма "Вие".
- Съществителните се пишат с малка буква; само режимите (Смарт, Бързо, Изкл.) и evcc са изключение.

## Català

- Tractament: "vós" quan la interfície s'adreça a l'usuari. Les accions que l'usuari demana a l'aplicació van en imperatiu de segona persona del singular ("Afegeix").
- Apòstrof tipogràfic ’ en comptes de ', i ela geminada amb punt volat (l·l).

## Čeština

- Oslovení: vykání, "vy" malým písmenem.

## Dansk

- Tiltaleform: "du", småt skrevet.
- Sammensatte ord skrives i ét ord, bindestreg efter forkortelser og symboler: prisgrænse, energipris, CO₂-grænse.

## Ελληνικά

- Προσφώνηση: πληθυντικός ευγενείας ("σας").
- Μόνο το πρώτο γράμμα κεφαλαίο σε τίτλους και ετικέτες, όχι κεφαλαία σε κάθε λέξη όπως στα αγγλικά.

## Español

- Forma de tratamiento: "tú".
- Español de España por ahora. Una variante latinoamericana (es-419) llegará más tarde, por eso se anotan los términos que difieren (contador/medidor, vertido/inyección, previsión/pronóstico).
- Sentence case en títulos y botones, nunca Title Case.
- "solo" sin tilde. Números con cifras ("7 días"), no con letras.

## Suomi

- Puhuttelu: sinuttelu.
- Lyhenteen ja sanan yhdyssana kirjoitetaan yhdysmerkillä: CO₂-raja, URL-osoite.

## Français

- Forme d'adresse : « vous ».
- Accent sur les majuscules : « Énergie », pas « Energie ».

## Hrvatski

- Obraćanje: "vi", malim slovom.

## Magyar

- Megszólítás: tegezés, kisbetűvel.
- Összetett szavak egybeírva: energiaár, árlimit, vendégjármű. Rövidítés vagy jel után kötőjellel: CO₂-limit.

## Italiano

- Forma di cortesia: "tu", minuscolo.
- Termini minuscoli anche nelle etichette; solo CO₂, nomi propri e nomi di pagina con maiuscola.

## 日本語

- 敬体（です・ます調）。指示は「〜してください」。
- 外来語はカタカナ表記、ラテン文字の英単語は製品名以外に使わない（「スマート」「ブースト」「メーター」）。

## Lëtzebuergesch

- Uried: nach net festgeluecht, d'Iwwersetzung mëscht haut "du" an "Dir".
- Komposita ginn als ee Wuert geschriwwen (Hausbatterie, Gaaschtgefier). Bindestréch nëmme virun engem friemen oder englesche Bestanddeel (CO₂-Grenz, Sponsor-Token).

## Lietuvių

- Kreipinys: „jūs“ (mandagioji forma).

## Latviešu

- Uzruna: "jūs" (formāla), mazajiem burtiem.
- Termini ar īpašības vārdu lieto noteikto galotni (dinamiskais tarifs, ārējais ierobežojums).

## Nederlands

- Aanspreekvorm: "u".
- Samenstellingen aaneengeschreven (gastvoertuig, netmeter), koppelteken na CO₂ (CO₂-limiet).

## Norsk

- Tiltaleform: "du", med liten forbokstav.
- Sammensatte ord skrives i ett ord: batteriutlading, ladepunkt, nettmåler. Ikke særskriving.

## Polski

- Forma zwracania się: "ty", zaimki małą literą.
- Nazwy trybów ładowania i rzeczowniki pospolite małą literą; wielką literą tylko nazwy własne i skróty (PV, CO₂).

## Português

- Forma de tratamento: formal, terceira pessoa com "o seu". Sem "tu" nem "você".
- Ortografia do português europeu: utilizador, palavra-passe, contador, poupança; não as formas brasileiras (usuário, senha, medidor, economia).

## Română

- Formă de adresare: "dumneavoastră", abreviat "dvs.". Verbe la persoana a II-a plural: "Adăugați", "Editați".
- Diacritice cu virgulă dedesubt (ș, ț), întotdeauna scrise.

## Русский

- Обращение: «вы», со строчной буквы.
- Буква «ё» пишется везде (счётчик, завершённых).

## Slovenčina

- Oslovenie: vykanie, "vy" a "váš" s malým písmenom.

## Slovenščina

- Oblika nagovora: vikanje ("vi").

## Svenska

- Tilltal: "du", gemener.
- Sammansättningar skrivs ihop, med bindestreck efter förkortning eller symbol: CO₂-gräns, sponsor-token, 3-fas.

## தமிழ்

- விளிப்பு: மரியாதைப் பன்மை "நீங்கள்" / "உங்கள்", கட்டளைகள் மரியாதை விகுதி "-கவும்" உடன்.
- கிரந்த எழுத்துகள் (ஜ, ஸ, ஷ, ஹ) தவிர்க்கப்படுகின்றன; கடன் சொற்கள் தமிழ் எழுத்துகளிலேயே எழுதப்படுகின்றன.

## Türkçe

- Hitap: "sen", samimi.
- "charging" için "doldurma" kullanılır, "şarj" değil (doldurma cihazı, doldurma noktası, doldurma oturumu).

## Українська

- Звертання: "ви", з малої літери.

## Adding a language section

Add a section when a language needs rules beyond the term mapping, typically the form of address (formal or informal), compound spelling, or terms that stay untranslated.
