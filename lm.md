## Heute (ohne LM)

je Ladepunkt:

- Leistung Site aktualisieren
- Leistung alle Ladepunkte aktualisieren
- Gesamtbudget berechnen
- Aktuellen Ladepunkt steuern

## Lastmanagement #8427

Setup:

- alle Circuits hierarchisch dem Parent vMeter zurodnen

je Ladepunkt:

- Leistung Site aktualisieren
- Leistung aller Ladepunkte aktualisieren
- Gesamtbudget berechnen
- Aktuellen Ladepunkt steuern
  - dabei Strom/Leistung durch Circuit begrenzen
    - Circuit aktualisieren oder aggregierte Strom/Leistung aus vMeter verwenden
      -> u.U. mehrere Circuit-Zähler auszulesen
  - eigenen Strom/Leistung an LM zurück melden

## Vorschlag zur Vereinfachung der vMeter

Setup:

- ENTFÄLLT: alle Circuits hierarchisch dem Parent vMeter zuordnen

je Ladepunkt:

- Leistung Site aktualisieren
- Leistung alle Ladepunkte aktualisieren
- NEU: Ströme alle Ladepunkte aktualisieren (falls vorhanden)
- NEU: Leistung aller Circuits depth-first aktualisieren
  - dafür Werte der Ladepunkte verwenden wo kein Circuit Meter vorhanden
- Gesamtbudget berechnen
- Aktuellen Ladepunkt steuern
  - dabei Strom/Leistung durch Circuit begrenzen
    - ENTFÄLLT: Circuit aktualisieren oder aggregierte Strom/Leistung aus vMeter verwenden
