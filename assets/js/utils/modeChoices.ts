import { CHARGE_MODE } from "@/types/evcc";

interface ChargeModeChoice {
	key: CHARGE_MODE | '';
	name: string;
}

interface ChargeModeChoicesOptions {
	includeEmpty?: boolean;
	pvPossible?: boolean;
	smartCostAvailable?: boolean;
	t?: (key: string) => string;
}

/**
 * Get available charge modes based on system capabilities.
 * Filters modes based on pvPossible and smartCostAvailable.
 */
function getAvailableChargeModes(
	pvPossible: boolean = false,
	smartCostAvailable: boolean = false
): CHARGE_MODE[] {
	if (pvPossible) {
		return [CHARGE_MODE.OFF, CHARGE_MODE.PV, CHARGE_MODE.MINPV, CHARGE_MODE.NOW];
	}
	if (smartCostAvailable) {
		return [CHARGE_MODE.OFF, CHARGE_MODE.PV, CHARGE_MODE.NOW];
	}
	return [CHARGE_MODE.OFF, CHARGE_MODE.NOW];
}

/**
 * Get mode label with smart renaming.
 * Renames 'pv' to 'smart' for non-pv and dynamic tariffs scenarios.
 */
function getChargeModeLabel(
	mode: CHARGE_MODE,
	t: (key: string) => string,
	pvPossible: boolean = false,
	smartCostAvailable: boolean = false
): string {
	// rename pv mode to smart for non-pv and dynamic tariffs scenarios
	if (mode === CHARGE_MODE.PV && !pvPossible && smartCostAvailable) {
		return t("main.mode.smart");
	}
	return t(`main.mode.${mode}`);
}

/**
 * Generate mode choices for select/choice fields.
 * Includes empty option if specified for forms where it's optional.
 */
export function getChargeModeChoices(options: ChargeModeChoicesOptions = {}): ChargeModeChoice[] {
	const {
		includeEmpty = false,
		pvPossible = false,
		smartCostAvailable = false,
		t = (key: string) => key,
	} = options;

	const choices: ChargeModeChoice[] = [];

	if (includeEmpty) {
		choices.push({ key: "", name: "---" });
	}

	const modes = getAvailableChargeModes(pvPossible, smartCostAvailable);

	modes.forEach((mode) => {
		choices.push({
			key: mode,
			name: getChargeModeLabel(mode, t, pvPossible, smartCostAvailable),
		});
	});

	return choices;
}
