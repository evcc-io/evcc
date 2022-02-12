export default {
  header: { docs: "Documentation", blog: "Blog", github: "GitHub", about: "About evcc" },
  footer: {
    version: {
      versionShort: "v{installed}",
      versionLong: "Version {installed}",
      availableShort: "update",
      availableLong: "update available",
      modalTitle: "Update available",
      modalUpdateStarted: "Evcc will restart after the update..",
      modalInstalledVersion: "Currently installed version",
      modalNoReleaseNotes:
        "No release notes available. More information about the new version can be found here:",
      modalCancel: "Cancel",
      modalUpdate: "Update",
      modalUpdateNow: "Update now",
      modalDownload: "Download",
      modalUpdateStatusStart: "Update started: ",
      modalUpdateStatusFailed: "Update failed: ",
    },
    savings: {
      footerShort: "{percent}% solar",
      footerLong: "{percent}% solar energy",
      modalTitleShort: "{total} kWh charged · {percent}% solar",
      modalTitleLong: "{total} kWh charged · {percent}% solar energy",
      modalChartGrid: "Grid energy {grid} kWh",
      modalChartSelf: "Solar energy {self} kWh",
      modalSavingsPrice: "Effective energy price",
      modalSavingsTotal: "Savings compared to grid",
      modalExplaination: "Calculation",
      modalExplainationGrid: "grid tariff {gridPrice}",
      modalExplainationFeedIn: "feed-in rate {feedInPrice}",
      modalServerStart: "since server start {since}.",
      modalNoData: "nothing charged yet",
      experimentalLabel: "Experimental",
      experimentalText: "Implausible values? Questions about this view? Feel free to join our ",
    },
    sponsor: {
      thanks: "Thanks for your support, {sponsor}! It helps us with the further development.",
      confetti: "Ready for some sponsor confetti?",
      supportUs:
        "We want to make efficient home charging the standard for as many people as possible. Help us by supporting the further development and maintenance of the project.",
      sticker: "...or evcc stickers?",
      confettiPromise: "There will be stickers and digital confetti ;)",
      becomeSponsor: "Become a GitHub Sponsor",
    },
  },
  notifications: {
    modalTitle: "Notifications",
    dismissAll: "Dismiss all",
  },
  main: {
    energyflow: {
      noEnergy: "No Energyflow",
      homePower: "Consumption",
      loadpoints: "Loadpoint | Loadpoint | {count} Loadpoints",
      pvProduction: "Production",
      battery: "Battery",
      batteryCharge: "Battery charge",
      batteryDischarge: "Battery discharge",
      gridImport: "Grid import",
      selfConsumption: "Self consumption",
      pvExport: "Grid export",
    },
    mode: {
      title: "Mode",
      stop: "Stop",
      now: "Now",
      minpvShort: "Min",
      minpvLong: "Min + PV",
      pvShort: "PV",
      pvLong: "PV only",
    },
    loadpoint: {
      fallbackName: "Loadpoint",
      remoteDisabledSoft: "{source}: adaptive PV charging disabled",
      remoteDisabledHard: "{source}: disabled",
    },
    vehicle: {
      fallbackName: "Vehicle",
    },
    vehicleSoC: {
      disconnected: "disconnected",
      charging: "charging",
      ready: "ready",
      connected: "connected",
    },
    vehicleSubline: {
      mincharge: "minimum charging to {soc}%",
    },
    targetCharge: {
      inactiveLabel: "Target time",
      activeLabel: "{time}",
      modalTitle: "Set Target Time",
      description: "When should the vehicle be charged to <strong>{targetSoC}%</strong>?",
      today: "today",
      tomorrow: "tomorrow",
      targetIsInThePast: "The chosen time is in the past.",
      remove: "Remove",
      activate: "Activate",
      experimentalLabel: "Experimental",
      experimentalText: `
        This function is at an early stage.
        The algorithm is not perfect yet.
        The target time is currently not persisted - this means, it will be lost when your server restarts.
        So do not rely too much on this function.
        However, we look forward to your experiences and suggestions for improvement in the
      `,
    },
    loadpointDetails: {
      power: "Power",
      vehicleRange: "Range",
      charged: "Charged",
      duration: "Duration",
      remaining: "Remaining",
      tooltip: {
        phases: {
          scale1p: "Switching to single-phase in {remaining}.",
          scale3p: "Switching to three-phase in {remaining}.",
          charge1p: "Single-phase charging.",
          charge2p: "Two-phase charging.",
          charge3p: "Three-phase charging.",
        },
        pv: {
          enable: "Solar available. Resume charging in {remaining}.",
          disable: "Not enough solar. Pause charging in {remaining}.",
        },
      },
    },
  },
};
