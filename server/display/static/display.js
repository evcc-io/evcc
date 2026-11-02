const stateUrl = "/api/state";
const refreshInterval = 3000;
const staleAfter = 9000;
const params = new URLSearchParams(window.location.search);
const requestedLocale = params.get("lang") || "de";
const screenOffSeconds = readIntegerParam("screenOff", 0);
const loadpointTimeoutSeconds = readIntegerParam("loadpointTimeout", 0);
const requestedLoadpoint = params.get("loadpoint");
const ignoredLoadpoints = new Set(
  (params.get("ignoreLoadpoints") || "")
    .split(",")
    .map((name) => name.trim().toLowerCase())
    .filter(Boolean)
);
const theme = params.get("theme") || "dark";
const autoSwitch = readBooleanParam("autoSwitch", true);
let locale = requestedLocale;
let translations = {};
let fallbackTranslations = {};
let numberFormat;
const elements = {
  screen: document.querySelector("#screen"),
  content: document.querySelector("#content"),
  title: document.querySelector("#loadpoint-title"),
  clock: document.querySelector("#clock"),
  warning: document.querySelector("#warning"),
  battery: document.querySelector("#home-battery"),
  batterySoc: document.querySelector("#battery-soc"),
  vehicle: document.querySelector("#vehicle"),
  vehicleIcon: document.querySelector("#vehicle-icon"),
  vehicleText: document.querySelector("#vehicle-text"),
  reading: document.querySelector("#reading"),
  value: document.querySelector("#main-value"),
  mainIcon: document.querySelector("#main-icon use"),
  idle: document.querySelector("#idle-reading"),
  idleIcon: document.querySelector("#idle-icon use"),
  idleText: document.querySelector("#idle-text"),
  status: document.querySelector("#status"),
  flow: document.querySelector("#energy-flow"),
  flowTrack: document.querySelector(".flow-track"),
  flowFill: document.querySelector("#flow-fill"),
  flowLeft: document.querySelector("#flow-left"),
  flowRight: document.querySelector("#flow-right"),
  tiles: document.querySelector("#tiles"),
  dots: document.querySelector("#page-dots"),
  modes: document.querySelectorAll(".mode"),
};

let state;
let selected = requestedLoadpoint > 0 ? requestedLoadpoint - 1 : 0;
let lastSuccess = 0;
let pointerStart;
let lastInputAt = Date.now();
let previousConnections;
let initialLoadpointResolved = false;
let initialLoadpoint = 0;
let loadpointTimeout;

if (theme === "light" || theme === "dark") {
  document.documentElement.dataset.theme = theme;
} else {
  console.warn("display: ignoring invalid theme parameter");
  document.documentElement.dataset.theme = "dark";
}

function readIntegerParam(name, fallback, minimum = 0) {
  const value = params.get(name);
  if (value === null) return fallback;

  const parsed = Number(value);
  if (!Number.isSafeInteger(parsed) || parsed < minimum) {
    console.warn(`display: ignoring invalid ${name} parameter`);
    return fallback;
  }
  return parsed;
}

function readBooleanParam(name, fallback) {
  const value = params.get(name);
  if (value === null) return fallback;
  if (/^(1|true|on)$/i.test(value)) return true;
  if (/^(0|false|off)$/i.test(value)) return false;

  console.warn(`display: ignoring invalid ${name} parameter`);
  return fallback;
}

function findTranslation(source, key) {
  return key.split(".").reduce((value, part) => value?.[part], source);
}

function t(key, values = {}) {
  const message = findTranslation(translations, key) ?? findTranslation(fallbackTranslations, key);
  if (typeof message !== "string") {
    console.error(`display: missing translation ${key}`);
    return key;
  }
  return message.replace(/\{(\w+)\}/g, (_, name) => String(values[name] ?? `{${name}}`));
}

async function fetchTranslations(language) {
  const response = await fetch(`/i18n/${encodeURIComponent(language)}.json`, { cache: "no-cache" });
  if (!response.ok) throw new Error(`translation request failed: ${response.status}`);
  return response.json();
}

async function loadTranslations() {
  let selectedTranslations;
  try {
    selectedTranslations = await fetchTranslations(requestedLocale);
  } catch (error) {
    console.warn(
      `display: unsupported language ${requestedLocale}, falling back to English`,
      error
    );
    locale = "en";
  }

  if (locale === "en") {
    fallbackTranslations = selectedTranslations || (await fetchTranslations("en"));
    translations = fallbackTranslations;
  } else {
    translations = selectedTranslations;
    fallbackTranslations = await fetchTranslations("en");
  }

  try {
    numberFormat = new Intl.NumberFormat(locale, {
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
    });
  } catch (error) {
    console.warn(`display: unsupported number format locale ${locale}`, error);
    locale = "en";
    numberFormat = new Intl.NumberFormat("en", {
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
    });
  }

  document.documentElement.lang = locale;
  document.title = t("display.pageTitle");
  document.querySelector('meta[name="description"]').content = t("display.description");
  elements.warning.setAttribute("aria-label", t("display.accessibility.noCurrentData"));
  elements.modes.forEach((element) => {
    const mode = element.dataset.mode;
    element.querySelector("span").textContent = t(`display.mode.${mode}`);
    element.querySelector("small").textContent = t(`display.mode.${mode}Description`);
  });
  document
    .querySelector(".mode-segments")
    .setAttribute("aria-label", t("display.accessibility.chargingMode"));
  elements.dots.setAttribute("aria-label", t("display.accessibility.loadpoint"));
}

function isIgnoredLoadpoint(loadpoint) {
  return [loadpoint?.name, loadpoint?.title].some(
    (name) => typeof name === "string" && ignoredLoadpoints.has(name.trim().toLowerCase())
  );
}

function visibleLoadpoints(loadpoints) {
  return loadpoints
    .map((loadpoint, index) => ({ loadpoint, index }))
    .filter(({ loadpoint }) => loadpoint?.disabled !== true && !isIgnoredLoadpoint(loadpoint));
}

function selectInitialLoadpoint(loadpoints) {
  if (initialLoadpointResolved || loadpoints.length === 0) return;

  const visible = visibleLoadpoints(loadpoints);
  let index = -1;
  if (requestedLoadpoint !== null) {
    if (/^\d+$/.test(requestedLoadpoint)) {
      index = Number(requestedLoadpoint) - 1;
    } else {
      index = loadpoints.findIndex(
        (loadpoint) =>
          loadpoint?.name === requestedLoadpoint || loadpoint?.title === requestedLoadpoint
      );
      if (index < 0) {
        const match = /^lp-(\d+)$/.exec(requestedLoadpoint);
        if (match) index = Number(match[1]) - 1;
      }
    }
  }

  if (visible.some((entry) => entry.index === index)) {
    selected = index;
  } else {
    if (requestedLoadpoint !== null) {
      console.warn("display: ignoring unknown or excluded loadpoint parameter");
    }
    selected = visible[0]?.index ?? 0;
  }
  initialLoadpoint = selected;
  initialLoadpointResolved = true;
}

function selectLoadpoint(index) {
  if (index === selected) return;

  selected = index;
  window.clearTimeout(loadpointTimeout);
  loadpointTimeout = undefined;

  if (loadpointTimeoutSeconds > 0 && selected !== initialLoadpoint) {
    loadpointTimeout = window.setTimeout(() => {
      selected = Math.min(initialLoadpoint, Math.max((state?.loadpoints?.length || 0) - 1, 0));
      loadpointTimeout = undefined;
      render();
    }, loadpointTimeoutSeconds * 1000);
  }
}

function switchLoadpoint(direction) {
  const visible = visibleLoadpoints(state?.loadpoints || []);
  if (visible.length < 2) return;

  const currentIndex = visible.findIndex((entry) => entry.index === selected);
  const nextIndex = Math.max(0, Math.min(visible.length - 1, currentIndex + direction));
  if (nextIndex === currentIndex) return;

  selectLoadpoint(visible[nextIndex].index);
  render();
  elements.content.classList.remove("enter-from-left", "enter-from-right");
  void elements.content.offsetWidth;
  elements.content.classList.add(direction > 0 ? "enter-from-left" : "enter-from-right");
}

const isNumber = (value) => typeof value === "number" && Number.isFinite(value);
const powerValue = (watts) => numberFormat.format(watts / 1000);
const powerText = (watts) => `${powerValue(watts)} kW`;
const secondsText = (seconds) =>
  t("display.duration", {
    hours: Math.floor(seconds / 3600),
    minutes: String(Math.floor(seconds / 60) % 60).padStart(2, "0"),
  });
const countdownText = (seconds) =>
  `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;

function batterySoc(snapshot) {
  return isNumber(snapshot.battery?.soc) ? snapshot.battery.soc : snapshot.batterySoc;
}

function batteryPower(snapshot) {
  return isNumber(snapshot.battery?.power) ? snapshot.battery.power : snapshot.batteryPower;
}

function gridPower(snapshot) {
  return isNumber(snapshot.grid?.power) ? snapshot.grid.power : snapshot.gridPower;
}

function solarSurplus(snapshot) {
  const battery = batterySoc(snapshot) >= 0 ? batteryPower(snapshot) || 0 : 0;
  const grid = gridPower(snapshot);
  return Math.max(
    0,
    isNumber(grid)
      ? -(grid + battery)
      : isNumber(snapshot.pvPower)
        ? snapshot.pvPower - (snapshot.homePower || 0)
        : 0
  );
}

function loadpointMode(loadpoint) {
  if (loadpoint.mode === "now") return "fast";
  if (loadpoint.mode === "minpv") return "slow";
  if (loadpoint.mode === "pv") return "smart";
  if (loadpoint.mode === "smart") {
    return ["on", "once", true].includes(loadpoint.alwaysCharge) ? "slow" : "smart";
  }
  return "";
}

function setTile(index, label, value, muted = false, visible = true) {
  const tile = document.querySelector(`#tile-${index}`);
  tile.hidden = !visible;
  if (!visible) return;
  document.querySelector(`#tile-label-${index}`).textContent = label;
  const valueElement = document.querySelector(`#tile-value-${index}`);
  valueElement.textContent = value;
  valueElement.classList.toggle("is-muted", muted);
}

function setFlow(left, right, share, live, type = "solar") {
  elements.flow.hidden = false;
  elements.flowTrack.classList.toggle("is-flow", type === "flow");
  elements.flowFill.style.width = `${Math.max(0, Math.min(100, share))}%`;
  elements.flowTrack.style.opacity = live ? "1" : "0.45";
  elements.flowLeft.textContent = left;
  elements.flowRight.textContent = right;
}

function updateDots(count, selectedIndex) {
  elements.dots.replaceChildren();
  if (count < 2) return;
  for (let index = 0; index < count; index += 1) {
    const dot = document.createElement("span");
    dot.className = `page-dot${index === selectedIndex ? " is-current" : ""}`;
    dot.setAttribute("aria-hidden", "true");
    elements.dots.append(dot);
  }
}

function statusText(loadpoint, house, live) {
  if (!live) return t("display.status.offline");
  if (!loadpoint) return t("display.status.missingLoadpoint");
  if (!loadpoint.connected && house) {
    const minCurrent = loadpoint.effectiveMinCurrent || loadpoint.minCurrent || 0;
    const phases =
      loadpoint.phasesConfigured === 1 || loadpoint.phasesConfigured === 3
        ? loadpoint.phasesConfigured
        : loadpoint.chargerPhases1p3p
          ? 1
          : 3;
    const minPower = minCurrent * 230 * phases;
    if (minPower > 0) {
      return solarSurplus(state) >= minPower
        ? t("display.status.surplusEnough")
        : t("display.status.surplusStarts", { power: powerText(minPower) });
    }
    return t("display.status.pvSurplus");
  }
  if (!loadpoint.connected) return t("display.status.chooseMode");
  if (loadpoint.mode === "off") return t("display.status.off");
  if (loadpoint.charging) {
    const phases =
      loadpoint.phasesActive > 0 ? t("display.phase", { count: loadpoint.phasesActive }) : "";
    if (loadpoint.pvAction === "disable" && loadpoint.pvRemaining > 0) {
      return t("display.status.chargingStops", {
        phases,
        time: countdownText(loadpoint.pvRemaining),
      });
    }
    return t("display.status.charging", { phases });
  }
  if (loadpoint.pvAction === "enable" && loadpoint.pvRemaining > 0) {
    return t("display.status.pvStarts", { time: countdownText(loadpoint.pvRemaining) });
  }
  if (loadpoint.mode === "smart") return t("display.status.waitSurplus");
  if (loadpoint.planActive || loadpoint.smartCostActive) return t("display.status.waitCheap");
  return t("display.status.connectedIdle");
}

function render() {
  const loadpoints = Array.isArray(state?.loadpoints) ? state.loadpoints : [];
  const visible = visibleLoadpoints(loadpoints);
  let selectedIndex = visible.findIndex((entry) => entry.index === selected);
  if (selectedIndex < 0 && visible.length > 0) {
    selectedIndex = 0;
    selected = visible[0].index;
  }
  const loadpoint = visible[selectedIndex]?.loadpoint;
  const live = Date.now() - lastSuccess < staleAfter;
  const charging = live && visible.some((entry) => entry.loadpoint?.charging === true);
  const screenOff =
    screenOffSeconds > 0 && !charging && Date.now() - lastInputAt >= screenOffSeconds * 1000;
  elements.screen.classList.toggle("is-screen-off", screenOff);
  const house = Boolean(
    loadpoint && !loadpoint.connected && isNumber(state.pvPower) && state.pvPower >= 0
  );
  const connected = Boolean(loadpoint?.connected);
  const showPower = connected;
  const homeBatterySoc = batterySoc(state || {});
  const mode = loadpoint ? loadpointMode(loadpoint) : "";

  elements.title.textContent =
    loadpoint?.title ||
    (loadpoints.length ? t("display.titleNumbered", { number: selected + 1 }) : "evcc");
  elements.clock.textContent = new Intl.DateTimeFormat(locale, {
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date());
  elements.warning.hidden = live;
  elements.battery.hidden = !loadpoint || house || !isNumber(homeBatterySoc) || homeBatterySoc < 0;
  elements.batterySoc.textContent = `${Math.round(homeBatterySoc)} %`;
  const vehicleText = house
    ? t("display.idle.noVehicle")
    : connected
      ? [
          loadpoint.vehicleTitle,
          loadpoint.vehicleSoc > 0 ? `${Math.round(loadpoint.vehicleSoc)} %` : "",
        ]
          .filter(Boolean)
          .join(" · ")
      : "";
  elements.vehicleText.textContent = vehicleText;
  elements.vehicleIcon.setAttribute("href", house ? "#plug" : "#car");
  elements.vehicle.classList.toggle("is-visible", Boolean(vehicleText));

  elements.reading.hidden = !showPower && !house;
  elements.idle.hidden = showPower || house;
  if (showPower || house) {
    const watts = showPower ? loadpoint.chargePower : solarSurplus(state);
    elements.value.textContent = powerValue(isNumber(watts) ? watts : 0);
    elements.mainIcon.setAttribute("href", house ? "#sun" : "#bolt");
    elements.reading.classList.toggle(
      "is-muted",
      !live || (showPower ? !loadpoint.charging : watts < (loadpoint.minCurrent || 0) * 230)
    );
  } else {
    const idleIcon = !loadpoint ? "#alert" : !live ? "#cloud" : "#plug";
    elements.idleIcon.setAttribute("href", idleIcon);
    elements.idleText.textContent = !loadpoint
      ? state
        ? t("display.idle.noLoadpoint")
        : t("display.idle.connecting")
      : !live
        ? t("display.idle.noCurrentData")
        : t("display.idle.noVehicle");
    elements.idle.classList.toggle("is-error", !live || !loadpoint);
  }

  elements.status.textContent = statusText(loadpoint, house, live);
  elements.status.className = "status";
  if (!live || !loadpoint) elements.status.classList.add("is-error");
  else if (loadpoint.pvAction === "disable" || loadpoint.mode === "off")
    elements.status.classList.add("is-warning");
  else if (
    house &&
    solarSurplus(state) >= (loadpoint.effectiveMinCurrent || loadpoint.minCurrent || 0) * 230
  )
    elements.status.classList.add("is-solar");

  elements.flow.hidden = true;
  elements.flowTrack.classList.remove("is-flow");
  if (house && homeBatterySoc >= 0 && isNumber(gridPower(state))) {
    const battery = Math.abs(batteryPower(state) || 0);
    const grid = Math.abs(gridPower(state));
    const share = battery + grid >= 50 ? (battery / (battery + grid)) * 100 : 0;
    const batteryLabel =
      battery < 50
        ? t(
            homeBatterySoc >= 99.5
              ? "display.flow.homeBatteryFull"
              : "display.flow.homeBatteryPaused"
          )
        : t(
            batteryPower(state) < 0
              ? "display.flow.homeBatteryCharging"
              : "display.flow.homeBatteryDischarging",
            { power: powerText(battery) }
          );
    const gridLabel = t(
      gridPower(state) > 0 ? "display.flow.gridImport" : "display.flow.gridExport",
      { power: powerText(grid) }
    );
    setFlow(batteryLabel, gridLabel, share, live, "flow");
  } else if (showPower && loadpoint.charging && isNumber(state.greenShareLoadpoints)) {
    const share = Math.max(0, Math.min(1, state.greenShareLoadpoints)) * 100;
    setFlow(
      t("display.flow.pvNow", { share: Math.round(share) }),
      t("display.flow.gridShare", { share: 100 - Math.round(share) }),
      share,
      live
    );
  }

  const showSessionTiles =
    showPower &&
    (loadpoint.charging || (loadpoint.sessionEnergy || loadpoint.chargedEnergy || 0) > 0);
  elements.tiles.hidden = !house && !showSessionTiles;
  if (house) {
    const batteryAvailable = isNumber(homeBatterySoc) && homeBatterySoc >= 0;
    elements.tiles.classList.toggle("two-tiles", !batteryAvailable);
    setTile(0, t("display.tile.pv"), powerText(state.pvPower), !live);
    setTile(1, t("display.tile.home"), powerText(state.homePower || 0), !live);
    setTile(
      2,
      t("display.tile.homeBattery"),
      batteryAvailable ? `${Math.round(homeBatterySoc)} %` : "",
      !live,
      batteryAvailable
    );
  } else if (showSessionTiles) {
    elements.tiles.classList.remove("two-tiles");
    const energy = isNumber(loadpoint.sessionEnergy)
      ? loadpoint.sessionEnergy
      : loadpoint.chargedEnergy || 0;
    setTile(0, t("display.tile.charged"), `${numberFormat.format(energy)} kWh`, !live);
    setTile(1, t("display.tile.duration"), secondsText(loadpoint.chargeDuration || 0), !live);
    setTile(
      2,
      t("display.tile.averagePV"),
      `${Math.round(loadpoint.sessionSolarPercentage || 0)} %`,
      !live
    );
  }

  elements.modes.forEach((element) => {
    element.classList.toggle("is-current", element.dataset.mode === mode);
  });
  updateDots(visible.length, selectedIndex);
}

async function poll() {
  try {
    const response = await fetch(stateUrl, { cache: "no-store" });
    if (!response.ok) throw new Error(`state request failed: ${response.status}`);
    const payload = await response.json();
    const nextState = payload.result || payload;
    if (!Array.isArray(nextState.loadpoints)) throw new Error("state response has no loadpoints");
    selectInitialLoadpoint(nextState.loadpoints);
    const connections = nextState.loadpoints.map(
      (loadpoint) => !isIgnoredLoadpoint(loadpoint) && Boolean(loadpoint?.connected)
    );
    if (autoSwitch && previousConnections) {
      const connectedIndex = connections.findIndex(
        (connected, index) => connected && !previousConnections[index]
      );
      if (connectedIndex >= 0) selectLoadpoint(connectedIndex);
    }
    previousConnections = connections;
    state = nextState;
    lastSuccess = Date.now();
  } catch (error) {
    console.error("display: failed to load evcc state", error);
  }
  render();
  window.setTimeout(poll, refreshInterval);
}

elements.screen.addEventListener("pointerdown", (event) => {
  if (!event.isPrimary) return;
  const wakeOnly = elements.screen.classList.contains("is-screen-off");
  lastInputAt = Date.now();
  elements.screen.classList.remove("is-screen-off");
  pointerStart = { x: event.clientX, y: event.clientY, id: event.pointerId, wakeOnly };
});

elements.screen.addEventListener("pointerup", (event) => {
  if (!pointerStart || pointerStart.id !== event.pointerId) return;
  if (pointerStart.wakeOnly) {
    pointerStart = undefined;
    return;
  }
  const dx = event.clientX - pointerStart.x;
  const dy = event.clientY - pointerStart.y;
  pointerStart = undefined;

  if (Math.abs(dx) >= 48 && Math.abs(dx) >= Math.abs(dy) * 1.25) {
    switchLoadpoint(dx < 0 ? 1 : -1);
    return;
  }
  if (Math.hypot(dx, dy) > 16) return;

  const bounds = elements.screen.getBoundingClientRect();
  const x = event.clientX - bounds.left;
  if (x < bounds.width * 0.25) switchLoadpoint(-1);
  else if (x >= bounds.width * 0.75) switchLoadpoint(1);
});

elements.screen.addEventListener("pointercancel", () => {
  pointerStart = undefined;
});

async function startDisplay() {
  await loadTranslations();
  window.setInterval(render, 1000);
  poll();
}

startDisplay().catch((error) => console.error("display: failed to initialize", error));
