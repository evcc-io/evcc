export default {
  header: { support: "Support" },
  footer: {
    version: {
      version: "Version",
      availableShort: "verfügbar",
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
    sponsor: {
      sponsoredShort: "Danke",
      sponsoredLong: "Danke {sponsor}",
      supportProjectShort: "unterstützen",
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
      homePower: "Verbrauch",
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
      vehicleRange: "Reichweite",
      charged: "Geladen",
      duration: "Dauer",
      remaining: "Restzeit",
      tooltip: {
        scale1p: "Umschaltung auf 1-phasig {remaining}",
        scale3p: "Umschaltung auf 3-phasig {remaining}",
        inactive: "lädt {activePhases}-phasig",
      },
    },
  },
};
