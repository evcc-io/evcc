export default {
  header: { support: "Documentation" },
  footer: {
    version: {
      version: "version",
      availableShort: "available",
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
    sponsor: {
      sponsoredShort: "thanks",
      sponsoredLong: "thanks {sponsor}",
      supportProjectShort: "support",
      supportProjectLong: "support the project",
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
      activeLabel: "until {time}",
      modalTitle: "Set Target Time",
      description: "When should the vehicle be chargerd to <strong>{targetSoC}%</strong>?",
      today: "today",
      tomorrow: "tomorrow",
      targetIsInThePast: "The chosen time is in the past.",
      remove: "Remove",
      activate: "Activate",
      experimentalLabel: "experimental",
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
    },
  },
};
