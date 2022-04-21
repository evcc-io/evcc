export default {
  header: {
    docs: "Dokumentacija",
    blog: "Tinklaraštis",
    github: "GitHub",
    login: "Prisijungimas",
  },
  footer: {
    version: {
      versionShort: "v{installed}",
      versionLong: "Versija {installed}",
      availableShort: "Naujinimas",
      availableLong: "Prieinama naujesnė versija",
      modalTitle: "Prieinama naujesnė versija",
      modalUpdateStarted: "Pasibaigus naujinimui EVCC startuos iš naujo..",
      modalInstalledVersion: "Dabartinė versija",
      modalNoReleaseNotes:
        "Naujinimo detalių nėra. Daugiau informacijos rasite čia:",
      modalCancel: "Atšaukti",
      modalUpdate: "Naujinti",
      modalUpdateNow: "Naujinti dabar",
      modalDownload: "Atsisiųsti",
      modalUpdateStatusStart: "Naujinimas prasidėjo: ",
      modalUpdateStatusFailed: "Naujinimas nepavyko: ",
    },
    savings: {
      footerShort: "{percent}% saulės",
      footerLong: "{percent}% saulės energija",
      modalTitleShort: "{total} kWh įkrauta · {percent}% saulės",
      modalTitleLong: "{total} kWh įkrauta · {percent}% saulės energija",
      modalChartGrid: "Energija iš tinklo {grid} kWh",
      modalChartSelf: "Saulės energija {self} kWh",
      modalSavingsPrice: "Faktinė energijos kaina",
      modalSavingsTotal: "Sutaupyta, palyginus su tinklu",
      modalExplaination: "Skaičiavimas",
      modalExplainationGrid: "kaina iš tinklo {gridPrice}",
      modalExplainationFeedIn: "Kompensavimas už patiektą į tinklą energiją {feedInPrice}",
      modalServerStart: "nuo EVCC starto {since}.",
      modalNoData: "įkrovimo duomenų dar nėra",
      experimentalLabel: "eksperimentinis",
      experimentalText: "Neteisingi duomenys? Turite klausimų apie šiuos skaičiavimus? Prisijunkite prie mūsų",
    },
    sponsor: {
      thanks: "Ačiū, kad mus remiate {sponsor}! Tuo prisidedate prie projekto vystymo.",
      confetti: "Ar pasiruošę priimti rėmėjo konfeti?",
      supportUs:
        "Norime, kad efektyvesnis elektromobilių įkrovimas taptų pasiekiamas kuo daugiau žmonių. Tapdami rėmėjais prisidedate prie nuolatinio projekto vystymo ir palaikymo.",
      sticker: "... ar evcc lipdukų?",
      confettiPromise: "Gausite skaitmeninių lipdukų ir konfeti ;)",
      becomeSponsor: "Taptkite GitHub rėmėju!",
    },
  },
  notifications: {
    modalTitle: "Pranešimai",
    dismissAll: "Išvalyti visus",
  },
  main: {
    energyflow: {
      noEnergy: "Energija neteka",
      homePower: "Namo suvartojimas",
      loadpoints: "Įkroviklis | Įkroviklis | {count} Įkrovikliai",
      pvProduction: "Gamyba",
      battery: "Baterija",
      batteryCharge: "Baterijos įkrovimas",
      batteryDischarge: "Baterijos iškrovimas",
      gridImport: "Tinklo importas",
      selfConsumption: "Sunaudojama iškart",
      pvExport: "Tinklo eksportas",
    },
    mode: {
      title: "Darbo režimas",
      stop: "Stop",
      now: "Dabar",
      minpvShort: "Min",
      minpvLong: "Min + PV",
      pvShort: "PV",
      pvLong: "Tik PV",
    },
    loadpoint: {
      fallbackName: "Įkroviklis",
      remoteDisabledSoft: "{source}: adaptyvus PV įkrovimas išjungtas",
      remoteDisabledHard: "{source}: Išjungtas",
    },
    vehicle: {
      fallbackName: "Automobilis",
    },
    vehicleSoC: {
      disconnected: "neprijungtas",
      charging: "vyksta įkrovimas",
      ready: "leidžiama įkrauti",
      connected: "prijungtas",
    },
    vehicleSubline: {
      mincharge: "minimalus įkrovimas iki {soc}%",
    },
    provider: {
      login: "prisijungti",
      logout: "atsijungti",
    },
    targetCharge: {
      inactiveLabel: "Suplanuota įkrovimo pabaiga",
      activeLabel: "įkrauti iki {time}",
      modalTitle: "Nustatyti įkrovimo pabaigos laiką",
      description: "Kada automobilis turėtų būti įkrautas iki <strong>{targetSoC}%</strong>?",
      today: "šiandien",
      tomorrow: "rytoj",
      targetIsInThePast: "Pasirinktas laikas yra praeityje.",
      remove: "Panaikinti",
      activate: "Aktyvuoti",
      experimentalLabel: "eksperimentinis",
      experimentalText: `
        Ši funkcija yra ankstyvoje integravimo stadijoje.
        Algoritmas kol kas nėra tobulas.
        Nustatytas laikas neišsaugojamas - neišlieka po serverio restarto.
        Dėl šių priežasčių pernelyg nepasitikėkite šia eksperimentine funkcija.
        Visgi, norėtume gauti atsiliepimų apie jos veikimą bei pasiūlymų tobulinimui mūsų
      `,
    },
    loadpointDetails: {
      power: "Galia",
      vehicleRange: "Apytikris nuvažiuojamas atstumas",
      charged: "Įkrauta",
      duration: "Įkrovimas vyksta",
      remaining: "Pabaiga už",
      tooltip: {
        phases: {
          scale1p: "Perjungimas į vienfazį įkrovimą už {remaining}.",
          scale3p: "Perjungimas į trifazį įkrovimą už {remaining}.",
          charge1p: "Vienfazis įkrovimas.",
          charge2p: "Dvifazis įkrovimas.",
          charge3p: "Trifazis įkrovimas.",
        },
        pv: {
          enable: "Saulės energijos pakanka, įkrovimas tęsiamas už {remaining}.",
          disable: "Nepakanka saulės energijos, įkrovimo pauzė už {remaining}.",
        },
      },
    },
  },
};
