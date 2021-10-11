export default {
  header: { support: "Support" },
  footer: {
    version: {
      versionShort: "v{installed}",
      versionLong: "Version {installed}",
      availableShort: "Update",
      availableLong: "Update verfügbar",
      modalTitle: "Update verfügbar",
      modalUpdateStarted: "Nach der Aktualisierung wird evcc neu gestartet.",
      modalInstalledVersion: "Aktuell installierte Version",
      modalNoReleaseNotes:
        "Keine Releasenotes verfügbar. Mehr Informationen zur neuen Version findest du hier:",
      modalCancel: "Abbrechen",
      modalUpdate: "Aktualisieren",
      modalUpdateNow: "Jetzt aktualisieren",
      modalDownload: "Download",
      modalUpdateStatusStart: "Aktualisierung gestartet: ",
      modalUpdateStatusFailed: "Aktualisierung nicht möglich: ",
    },
    savings: {
      footerShort: "{percent}%",
      footerLong: "{percent}% Sonnenstrom geladen",
      modalTitle: "{percent}% Sonnenstrom geladen",
      modalText:
        "<strong>{self} kWh (von {total} kWh)</strong> waren eigener Sonnenstrom. Das entspricht einer <strong>Ersparnis von ~{savingEuro}</strong>.",
      modalSupportUs:
        "Wir möchten, das effizientes Zuhause Laden für möglichst viele Menschen zum Standard wird. Unterstütze uns auf dem Weg indem du die Weiterentwicklung und Pflege des Projekts unterstützt.",
      modalButtonBecomeSponsor: "Werde GitHub Sponsor",
      modalButtonConfetti: "Danke {sponsor}! Lust auf Konfetti?",
      modalSticker: "...oder evcc Sticker?",
      modalExplaination: "Annahmen:",
      modalExplainationGrid: "Strompreis {gridPrice}ct (fest)",
      modalExplainationFeedin: "Einspeisung {feedinPrice}ct",
      modalExplainationAdjust: "Tarifdaten anpassen",
      modalExplainationCalculation: "Berechnungsmethode",
      modalExplainationSince: "Zeitraum: ab Serverstart {since}",
    },
    sponsor: {
      sponsoredShort: "Danke",
      sponsoredLong: "Danke {sponsor}",
      supportProjectShort: "spenden",
      supportProjectLong: "Projekt unterstützen",
    },
  },
  notifications: {
    modalTitle: "Meldungen",
    dismissAll: "Meldungen entfernen",
  },

  main: {
    energyflow: {
      noEnergy: "Kein Energiefluss",
      houseConsumption: "Verbrauch",
      pvProduction: "Erzeugung",
      loadpoints: "Ladepunkt | Ladepunkt | {count} Ladepunkte",
      battery: "Batterie",
      batteryCharge: "Batterie laden",
      batteryDischarge: "Batterie entladen",
      gridImport: "Netzbezug",
      selfConsumption: "Eigenverbrauch",
      pvExport: "Einspeisung",
    },
    mode: {
      title: "Modus",
      stop: "Stop",
      now: "Sofort",
      minpvShort: "Min",
      minpvLong: "Min + PV",
      pvShort: "PV",
      pvLong: "Nur PV",
    },
    loadpoint: {
      fallbackName: "Ladepunkt",
      remoteDisabledSoft: "{source}: Adaptives PV-Laden deaktiviert",
      remoteDisabledHard: "{source}: Deaktiviert",
    },
    vehicle: {
      fallbackName: "Fahrzeug",
    },
    vehicleSoC: {
      disconnected: "getrennt",
      charging: "lädt",
      ready: "bereit",
      connected: "verbunden",
    },
    vehicleSubline: {
      mincharge: "Mindestladung bis {soc}%",
    },
    targetCharge: {
      inactiveLabel: "Zielzeit",
      activeLabel: "bis {time} Uhr",
      modalTitle: "Zielzeit festlegen",
      description: "Wann soll das Fahrzeug auf <strong>{targetSoC}%</strong> geladen sein?",
      today: "heute",
      tomorrow: "morgen",
      targetIsInThePast: "Zeitpunkt liegt in der Vergangenheit.",
      remove: "Keine Zielzeit",
      activate: "Zielzeit aktivieren",
      experimentalLabel: "experimentell",
      experimentalText: `
        Diese Funktion ist in einem frühen Stadium. Der Algorithmus ist noch
        nicht perfekt. Die Zielzeit wird aktuell nicht persistiert - das
        heißt sie geht beim Neustart verloren. Verlasse dich also noch nicht
        zu sehr auf diese Funktion. Wir freuen uns aber über deine
        Erfahrungen und Verbessungsvorschläge in den
      `,
    },
    loadpointDetails: {
      power: "Leistung",
      range: "Reichweite",
      charged: "Geladen",
      duration: "Dauer",
      remaining: "Restzeit",
    },
  },
};
