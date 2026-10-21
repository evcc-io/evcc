<template>
	<GenericModal
		id="onboardingModal"
		config-modal-name="onboarding"
		data-testid="onboarding-modal"
		uncloseable
		hide-header
		@open="onOpen"
	>
		<template v-if="step !== 'welcome'">
			<div
				class="progress"
				role="progressbar"
				:aria-valuenow="progress"
				:style="{ '--progress': `${progress}%` }"
			>
				<Transition appear name="grow">
					<div class="progress-bar bg-primary"></div>
				</Transition>
			</div>
			<div class="small fw-bold text-uppercase evcc-gray mt-3 mb-2">
				{{ $t("onboarding.step", { step: stepIndex + 1, total: steps.length }) }}
			</div>
		</template>

		<div>
			<div :key="step">
				<!-- welcome -->
				<div v-if="step === 'welcome'" class="text-center">
					<div class="d-flex justify-content-between evcc-gray mb-5 icon-row">
						<component
							:is="bandIcon"
							v-for="(bandIcon, i) in bandIcons"
							:key="i"
							:class="{ morph: i === startIconIndex }"
						/>
					</div>
					<div class="d-flex justify-content-center mb-4">
						<BrandLogo />
					</div>
					<p class="mx-auto description">{{ $t("onboarding.welcome.description") }}</p>
					<button type="button" class="btn btn-primary px-4 mt-3 mb-2" @click="start">
						{{ $t("onboarding.welcome.start") }}
					</button>
				</div>

				<!-- done -->
				<div v-else-if="step === 'done'" class="text-center pt-3">
					<shopicon-regular-checkmark
						size="xl"
						class="text-primary d-block mx-auto mb-2"
					></shopicon-regular-checkmark>
					<h2 class="mb-2">{{ $t("onboarding.done.title") }}</h2>
					<p class="mx-auto description">{{ $t("onboarding.done.description") }}</p>
					<div class="d-flex flex-column align-items-center gap-3 mt-4">
						<button
							v-if="restartNeeded"
							type="button"
							class="btn btn-primary px-4"
							@click="restart"
						>
							{{ $t("onboarding.done.restart") }}
						</button>
						<button
							type="button"
							:class="
								restartNeeded
									? 'btn btn-link btn-sm evcc-gray text-decoration-none p-0'
									: 'btn btn-primary px-4'
							"
							@click="closeModal()"
						>
							{{ $t("onboarding.done.close") }}
						</button>
					</div>
				</div>

				<template v-else>
					<h2 class="mb-3 d-flex align-items-center gap-2">
						<component
							:is="icon"
							v-if="icon"
							size="m"
							class="icon text-primary flex-shrink-0 morph"
						/>
						{{ $t(`onboarding.${step}.title`) }}
					</h2>

					<!-- password -->
					<template v-if="step === 'password'">
						<p>{{ $t("onboarding.password.description") }}</p>
						<PasswordStep @saved="next" />
					</template>

					<!-- site -->
					<template v-else-if="step === 'site'">
						<p>{{ $t("onboarding.site.description") }}</p>
						<SiteForm compact id-prefix="onboardingSite" @saved="next">
							<template #default="{ saving }">
								<div class="d-flex justify-content-end align-items-center gap-4">
									<button
										type="button"
										class="btn btn-link btn-sm evcc-gray text-decoration-none p-0"
										@click="skipSetup"
									>
										{{ $t("onboarding.skip") }}
									</button>
									<button
										type="submit"
										class="btn btn-primary"
										:disabled="saving"
									>
										{{ $t("config.general.continue") }}
									</button>
								</div>
							</template>
						</SiteForm>
					</template>

					<!-- battery, pv, grid, consumer, charging point or heater -->
					<template v-else-if="meterItem || step === 'control'">
						<p>{{ $t(`onboarding.${step}.description`) }}</p>
						<template v-if="configuredDevices.length">
							<DeviceList :items="configuredDevices" @edit="editDevice" />
							<div class="d-flex flex-wrap gap-3 mt-2">
								<template v-if="step === 'control'">
									<button
										type="button"
										class="btn btn-link btn-sm evcc-gray text-decoration-none p-0 d-flex align-items-center gap-1"
										@click="addLoadpoint('charging')"
									>
										<shopicon-regular-plus size="s"></shopicon-regular-plus>
										{{ $t("config.loadpoint.option.charging") }}
									</button>
									<button
										type="button"
										class="btn btn-link btn-sm evcc-gray text-decoration-none p-0 d-flex align-items-center gap-1"
										@click="addLoadpoint('heating')"
									>
										<shopicon-regular-plus size="s"></shopicon-regular-plus>
										{{ $t("config.loadpoint.option.heating") }}
									</button>
								</template>
								<button
									v-else-if="step !== 'grid'"
									type="button"
									class="btn btn-link btn-sm evcc-gray text-decoration-none p-0 d-flex align-items-center gap-1"
									@click="addDevice()"
								>
									<shopicon-regular-plus size="s"></shopicon-regular-plus>
									{{ $t("onboarding.another") }}
								</button>
							</div>
						</template>
						<template v-else-if="meterItem">
							<p v-if="confirmSkip" class="alert alert-warning">
								{{ $t("onboarding.grid.confirm") }}
							</p>
							<div class="d-flex gap-3 flex-column-reverse flex-sm-row">
								<button
									type="button"
									class="btn btn-outline-secondary w-100 py-3"
									@click="skipMeter"
								>
									{{
										confirmSkip
											? $t("onboarding.grid.confirmNone")
											: $t(`onboarding.${step}.none`)
									}}
								</button>
								<button
									type="button"
									class="btn btn-outline-primary w-100 py-3 fw-bold"
									@click="addDevice()"
								>
									{{ $t(`onboarding.${step}.add`) }}
								</button>
							</div>
						</template>
						<template v-else>
							<div class="d-flex gap-3 flex-column flex-sm-row mb-3">
								<button
									type="button"
									class="btn btn-outline-primary w-100 py-3 fw-bold d-flex flex-column align-items-center gap-2"
									@click="addLoadpoint('charging')"
								>
									<LoadpointIcon :size="ICON_SIZE.M" />
									{{ $t("onboarding.item.loadpoint") }}
								</button>
								<button
									type="button"
									class="btn btn-outline-primary w-100 py-3 fw-bold d-flex flex-column align-items-center gap-2"
									@click="addLoadpoint('heating')"
								>
									<HeaterIcon class="icon" />
									{{ $t("onboarding.item.heater") }}
								</button>
							</div>
							<button
								type="button"
								class="btn btn-outline-secondary w-100 py-3"
								@click="next"
							>
								{{ $t("onboarding.control.none") }}
							</button>
						</template>
					</template>

					<!-- tariffs -->
					<template v-else-if="step === 'tariff'">
						<p>{{ $t("onboarding.tariff.description") }}</p>
						<DeviceList
							:items="tariffRows"
							@add="(type) => openModal('tariff', { type })"
							@edit="editTariff"
						/>
					</template>
				</template>
			</div>
		</div>

		<div v-if="canGoBack" class="d-flex justify-content-between align-items-center gap-3 mt-4">
			<button
				type="button"
				class="btn btn-link btn-sm evcc-gray text-decoration-none p-0"
				@click="prev"
			>
				{{ $t("onboarding.back") }}
			</button>
			<button v-if="canContinue" type="button" class="btn btn-primary" @click="next">
				{{ $t("config.general.continue") }}
			</button>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/checkmark";
import "@h2d2/shopicons/es/regular/plus";
import "@h2d2/shopicons/es/regular/home";
import { defineComponent, type Component, type PropType } from "vue";
import confetti from "canvas-confetti";
import GenericModal from "../../Helper/GenericModal.vue";
import SiteForm from "../SiteForm.vue";
import PasswordStep from "./PasswordStep.vue";
import DeviceList, { type DeviceListItem } from "../DeviceList.vue";
import LoadpointIcon from "../../MaterialIcon/Loadpoint.vue";
import HeaterIcon from "../../VehicleIcon/Heater.vue";
import KeyIcon from "../../MaterialIcon/Key.vue";
import BrandLogo from "../../BrandLogo.vue";
import { ONBOARDING_ICONS } from "./icons";
import { closeModal, getModal, openModal, replaceModal } from "@/configModal";
import { clearSession, dismiss, finish, ONBOARDING_ITEMS } from "@/onboarding";
import auth from "../../Auth/auth";
import restart, { performRestart } from "@/restart";
import {
	CURRENCY,
	ICON_SIZE,
	type ConfigLoadpoint,
	type ConfigMeter,
	type Entity,
	type OnboardingItem,
} from "@/types/evcc";
import formatter, { POWER_UNIT } from "@/mixins/formatter";

type DeviceValues = Record<string, Record<string, Record<string, { value?: unknown }>>>;

type Step =
	| "welcome"
	| "password"
	| "site"
	| "battery"
	| "pv"
	| "grid"
	| "control"
	| "consumer"
	| "tariff"
	| "done";

const STEPS: Step[] = [
	"password",
	"site",
	"battery",
	"pv",
	"grid",
	"control",
	"consumer",
	"tariff",
	"done",
];

const METER_ITEMS: Step[] = ["battery", "pv", "grid", "consumer"];

export default defineComponent({
	name: "OnboardingModal",
	mixins: [formatter],
	components: {
		GenericModal,
		SiteForm,
		PasswordStep,
		DeviceList,
		LoadpointIcon,
		HeaterIcon,
		BrandLogo,
	},
	props: {
		// solar decides whether the export tariff row is shown
		pv: Boolean,
		batteryMeters: { type: Array as PropType<ConfigMeter[]>, default: () => [] },
		pvMeters: { type: Array as PropType<ConfigMeter[]>, default: () => [] },
		gridMeter: Object as PropType<ConfigMeter>,
		consumerMeters: { type: Array as PropType<ConfigMeter[]>, default: () => [] },
		chargingLoadpoints: { type: Array as PropType<ConfigLoadpoint[]>, default: () => [] },
		heaterLoadpoints: { type: Array as PropType<ConfigLoadpoint[]>, default: () => [] },
		gridTariff: Object as PropType<Entity | null>,
		feedInTariff: Object as PropType<Entity | null>,
		// live readings per device type and name, one is shown per row as proof of life
		deviceValues: { type: Object as PropType<DeviceValues>, default: () => ({}) },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		authDisabled: Boolean,
	},
	data() {
		return {
			ICON_SIZE,
			// skipping the grid meter asks twice, it is the basis for surplus charging
			confirmSkip: false,
		};
	},
	computed: {
		steps(): Step[] {
			return STEPS.filter((s) => s !== "password" || !this.authDisabled);
		},
		step(): Step {
			const type = getModal("onboarding")?.type as Step | undefined;
			return type && this.steps.includes(type) ? type : "welcome";
		},
		stepIndex(): number {
			return this.steps.indexOf(this.step);
		},
		progress(): number {
			return Math.round(((this.stepIndex + 1) / this.steps.length) * 100);
		},
		icon() {
			if (this.step === "password") return KeyIcon;
			if (this.step === "site") return "shopicon-regular-home";
			const item = this.step === "tariff" ? "tariff" : this.meterItem;
			return item ? ONBOARDING_ICONS[item] : null;
		},
		// a configured password keeps its place in the count but is not shown again
		startStep(): Step {
			return this.steps.find((s) => s !== "password" || !auth.configured)!;
		},
		// the band icon that morphs into the first step's heading icon
		startIconIndex(): number {
			return this.bandIcons.indexOf(
				this.startStep === "site" ? "shopicon-regular-home" : KeyIcon
			);
		},
		// what lies ahead, in step order
		bandIcons(): (string | Component)[] {
			const items = ONBOARDING_ITEMS.filter((i) => i !== "co2" && i !== "solar");
			return [
				...(this.authDisabled ? [] : [KeyIcon]),
				"shopicon-regular-home",
				...items.map((i) => ONBOARDING_ICONS[i]),
			];
		},
		// steps that add a meter, the step name doubles as checklist item and meter type
		meterItem(): OnboardingItem | undefined {
			return METER_ITEMS.includes(this.step) ? (this.step as OnboardingItem) : undefined;
		},
		// item (or tariffGrid / tariffFeedIn) -> configured devices with one formatted live value
		devices(): Record<string, DeviceListItem[]> {
			const meterRow = (m: ConfigMeter, tag: string) => ({
				id: m.id,
				title: m.deviceTitle || m.deviceProduct,
				value: this.liveValue("meter", m.name, tag),
			});
			const loadpointRow = (lp: ConfigLoadpoint) => ({
				id: lp.id!,
				title: lp.title,
				value: this.liveValue("charger", lp.charger, "power"),
			});
			const tariffRow = (t: Entity) => ({
				id: t.id,
				title: t.name,
				value: this.liveValue("tariff", t.name, "price"),
			});
			return {
				battery: this.batteryMeters.map((m) => meterRow(m, "soc")),
				pv: this.pvMeters.map((m) => meterRow(m, "power")),
				grid: this.gridMeter ? [meterRow(this.gridMeter, "power")] : [],
				consumer: this.consumerMeters.map((m) => meterRow(m, "power")),
				loadpoint: this.chargingLoadpoints.map(loadpointRow),
				heater: this.heaterLoadpoints.map(loadpointRow),
				tariffGrid: this.gridTariff ? [tariffRow(this.gridTariff)] : [],
				tariffFeedIn: this.feedInTariff ? [tariffRow(this.feedInTariff)] : [],
			};
		},
		configuredDevices(): DeviceListItem[] {
			if (this.step === "control") {
				return [...this.devices["loadpoint"]!, ...this.devices["heater"]!];
			}
			return this.meterItem ? this.devices[this.meterItem]! : [];
		},
		// export tariff only makes sense with solar
		tariffRows(): DeviceListItem[] {
			const types = this.pv ? ["grid", "feedIn"] : ["grid"];
			return types.map((type) => {
				const device = this.tariffDevice(type);
				return {
					id: type,
					title: this.$t(`onboarding.tariff.${type}`),
					value: device?.value,
					add: !device,
				};
			});
		},
		canGoBack(): boolean {
			const previous = this.steps[this.stepIndex - 1];
			return !!previous && previous !== "password" && this.step !== "done";
		},
		// device steps continue once something exists, the tariff step is optional
		canContinue(): boolean {
			return this.step === "tariff" || this.configuredDevices.length > 0;
		},
		restartNeeded(): boolean {
			return restart.restartNeeded;
		},
	},
	watch: {
		step(newStep: Step) {
			this.confirmSkip = false;
			if (newStep === "done") this.celebrate();
			this.$nextTick(this.focusFirst);
		},
	},
	methods: {
		onOpen() {
			if (this.step === "done") this.celebrate();
		},
		// morph the modal between steps where the browser supports it, hard swap otherwise
		goTo(step: Step) {
			const swap = () => replaceModal("onboarding", { type: step });
			const visible = document.getElementById("onboardingModal")?.classList.contains("show");
			if (!visible || !document.startViewTransition) return swap();
			return document.startViewTransition(async () => {
				swap();
				await this.$nextTick();
			});
		},
		async start() {
			document.documentElement.classList.add("onboarding-start");
			await this.goTo(this.startStep)?.finished;
			document.documentElement.classList.remove("onboarding-start");
		},
		next() {
			this.goTo(this.steps[this.stepIndex + 1]!);
		},
		prev() {
			this.goTo(this.steps[this.stepIndex - 1]!);
		},
		// form field or primary action, never a text link or a "none" button
		focusFirst() {
			const modal = document.getElementById("onboardingModal")!;
			modal
				.querySelector<HTMLElement>("input, select, .btn-primary, .btn-outline-primary")
				?.focus({ preventScroll: true });
		},
		openModal,
		closeModal,
		addLoadpoint(type: string) {
			openModal("loadpoint", { type });
		},
		addDevice() {
			openModal("meter", { type: this.meterItem });
		},
		editDevice(id: number) {
			if (this.step === "control") {
				openModal("loadpoint", { id });
			} else {
				openModal("meter", { type: this.meterItem, id });
			}
		},
		liveValue(type: string, name: string, tag: string): string {
			const value = this.deviceValues[type]?.[name]?.[tag]?.value;
			if (typeof value !== "number") return "";
			if (tag === "soc") return this.fmtPercentage(value, 1);
			if (tag === "price") return this.fmtPricePerKWh(value, this.currency, true);
			return this.fmtW(value, POWER_UNIT.AUTO);
		},
		// devices are keyed tariffGrid / tariffFeedIn
		tariffDevice(type: string): DeviceListItem | undefined {
			return this.devices[`tariff${type[0]!.toUpperCase()}${type.slice(1)}`]?.[0];
		},
		editTariff(type: string) {
			openModal("tariff", { type, id: this.tariffDevice(type)!.id as number });
		},
		async skipMeter() {
			if (this.step === "grid" && !this.confirmSkip) {
				this.confirmSkip = true;
				return;
			}
			await dismiss(this.meterItem!);
			this.next();
		},
		async skipSetup() {
			await finish();
			clearSession();
			closeModal();
		},
		async celebrate() {
			await finish();
			clearSession();
			// above the modal and its backdrop
			const style = getComputedStyle(document.documentElement);
			// energy flow colors; some tokens carry an alpha suffix (#rrggbbaa) canvas-confetti can't parse
			const colors = ["self", "battery", "export", "price", "co2"].map((c) =>
				style.getPropertyValue(`--evcc-${c}`).trim().slice(0, 7)
			);
			confetti({
				particleCount: 120,
				spread: 90,
				origin: { y: 0.6 },
				scalar: 1.2,
				zIndex: 2000,
				colors,
			});
		},
		async restart() {
			await performRestart();
			closeModal();
		},
	},
});
</script>

<style scoped>
.progress {
	--bs-progress-height: 4px;
	--bs-progress-bg: var(--evcc-gray-25);
}
.progress-bar {
	width: var(--progress);
}
/* bootstrap transitions the width, so the bar grows from zero when it first appears */
.grow-enter-from {
	width: 0;
}
.description {
	max-width: 24rem;
}
.icon {
	font-size: 32px;
	line-height: 0;
}
</style>

<style>
/* view transitions live on the root, GenericModal teleports, so this block is global on purpose */
#onboardingModal .modal-content {
	view-transition-name: onboarding;
}
#onboardingModal .modal-body {
	view-transition-name: onboarding-body;
}
/* leaving the welcome step: old out, the band icon flies into the heading, new in, so it never crosses content */
html.onboarding-start .morph {
	view-transition-name: onboarding-icon;
}
html.onboarding-start::view-transition-old(onboarding-body) {
	animation: 100ms ease-out both vt-fade-out;
}
html.onboarding-start::view-transition-new(onboarding-body) {
	animation: 150ms 200ms ease-in both vt-fade-in;
}
::view-transition-group(onboarding-icon) {
	animation-duration: 250ms;
	animation-delay: 60ms;
	animation-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
}
@keyframes vt-fade-out {
	to {
		opacity: 0;
	}
}
@keyframes vt-fade-in {
	from {
		opacity: 0;
	}
}
</style>
