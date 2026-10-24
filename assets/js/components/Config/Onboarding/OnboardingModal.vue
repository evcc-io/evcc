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
				{{ $t("onboarding.step", { step: stepIndex + 1, total: STEPS.length }) }}
			</div>
		</template>

		<div>
			<div :key="step">
				<!-- welcome -->
				<div v-if="step === 'welcome'" class="text-center">
					<div
						ref="band"
						class="band position-relative d-flex justify-content-between evcc-gray mb-5 icon-row"
						:class="{ bar: starting }"
					>
						<component
							:is="bandIcon"
							v-for="(bandIcon, i) in BAND_ICONS"
							:key="i"
							:class="{ current: i === startIconIndex }"
						/>
						<div class="line"></div>
					</div>
					<div class="d-flex justify-content-center mb-4">
						<BrandLogo />
					</div>
					<p class="mx-auto description">{{ $t("onboarding.welcome.description") }}</p>
					<button type="button" class="btn btn-primary px-4 mt-3 mb-2" @click="start">
						{{ $t("onboarding.welcome.start") }}
					</button>
					<div v-if="canSkip" class="mt-2">
						<button
							type="button"
							class="btn btn-link btn-sm evcc-gray text-decoration-none p-0"
							@click="skipSetup"
						>
							{{ $t("onboarding.skip") }}
						</button>
					</div>
				</div>

				<!-- done -->
				<div v-else-if="step === 'done'" class="text-center pt-3">
					<CheckIcon :size="ICON_SIZE.XL" class="text-primary mx-auto mb-2" />
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
							class="icon text-primary flex-shrink-0"
						/>
						{{ $t(`onboarding.${step}.title`) }}
					</h2>

					<!-- password -->
					<template v-if="step === 'password'">
						<template v-if="authDisabled">
							<i18n-t
								keypath="onboarding.password.disabled"
								tag="div"
								class="alert alert-warning"
								role="alert"
								scope="global"
							>
								<template #flag><code>--disable-auth</code></template>
							</i18n-t>
						</template>
						<template v-else>
							<p>{{ $t("onboarding.password.description") }}</p>
							<PasswordStep @saved="next" />
						</template>
					</template>

					<!-- site -->
					<template v-else-if="step === 'site'">
						<p>{{ $t("onboarding.site.description") }}</p>
						<SiteForm compact id-prefix="onboardingSite" @saved="next" />
					</template>

					<!-- battery, pv, grid, charging point or heater -->
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
				</template>
			</div>
		</div>

		<div v-if="footer" class="d-flex justify-content-between align-items-center gap-3 mt-4">
			<button
				type="button"
				class="btn btn-link btn-sm evcc-gray text-decoration-none p-0"
				@click="prev"
			>
				{{ $t("onboarding.back") }}
			</button>
			<button
				v-if="canContinue"
				:type="form ? 'submit' : 'button'"
				:form="form"
				class="btn btn-primary"
				@click="continueStep"
			>
				{{ continueLabel }}
			</button>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/plus";
import "@h2d2/shopicons/es/regular/home";
import { defineComponent, markRaw, type PropType } from "vue";
import confetti from "canvas-confetti";
import GenericModal from "../../Helper/GenericModal.vue";
import SiteForm from "../SiteForm.vue";
import PasswordStep from "./PasswordStep.vue";
import DeviceList, { type DeviceListItem } from "../DeviceList.vue";
import LoadpointIcon from "../../MaterialIcon/Loadpoint.vue";
import HeaterIcon from "../../VehicleIcon/Heater.vue";
import KeyIcon from "../../MaterialIcon/Key.vue";
import CheckIcon from "../../MaterialIcon/Check.vue";
import BrandLogo from "../../BrandLogo.vue";
import { ONBOARDING_ICONS } from "./icons";
import { closeModal, getModal, openModal, replaceModal } from "@/configModal";
import { clearSession, dismiss, finish } from "@/onboarding";
import auth from "../../Auth/auth";
import restart, { performRestart } from "@/restart";
import {
	ICON_SIZE,
	type ConfigLoadpoint,
	type ConfigMeter,
	type OnboardingItem,
} from "@/types/evcc";
import formatter, { POWER_UNIT } from "@/mixins/formatter";

type DeviceValues = Record<string, Record<string, Record<string, { value?: unknown }>>>;

type Step = "welcome" | "password" | "site" | "battery" | "pv" | "grid" | "control" | "done";

const STEPS: Step[] = ["password", "site", "battery", "pv", "grid", "control", "done"];

const METER_ITEMS: Step[] = ["battery", "pv", "grid"];
// what lies ahead, in step order
const BAND_ICONS = markRaw([
	KeyIcon,
	"shopicon-regular-home",
	...["battery", "pv", "grid", "loadpoint", "heater"].map(
		(i) => ONBOARDING_ICONS[i as OnboardingItem]
	),
	CheckIcon,
]);

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
		CheckIcon,
	},
	props: {
		batteryMeters: { type: Array as PropType<ConfigMeter[]>, default: () => [] },
		pvMeters: { type: Array as PropType<ConfigMeter[]>, default: () => [] },
		gridMeter: Object as PropType<ConfigMeter>,
		chargingLoadpoints: { type: Array as PropType<ConfigLoadpoint[]>, default: () => [] },
		heaterLoadpoints: { type: Array as PropType<ConfigLoadpoint[]>, default: () => [] },
		// live readings per device type and name, one is shown per row as proof of life
		deviceValues: { type: Object as PropType<DeviceValues>, default: () => ({}) },
		authDisabled: Boolean,
	},
	data() {
		return {
			ICON_SIZE,
			STEPS,
			BAND_ICONS,
			// skipping the grid meter asks twice, it is the basis for surplus charging
			confirmSkip: false,
			starting: false,
		};
	},
	computed: {
		step(): Step {
			const type = getModal("onboarding")?.type as Step | undefined;
			return type && STEPS.includes(type) ? type : "welcome";
		},
		stepIndex(): number {
			return STEPS.indexOf(this.step);
		},
		progress(): number {
			return Math.round(((this.stepIndex + 1) / STEPS.length) * 100);
		},
		icon() {
			if (this.step === "password") return KeyIcon;
			if (this.step === "site") return "shopicon-regular-home";
			return this.meterItem ? ONBOARDING_ICONS[this.meterItem] : null;
		},
		// a configured password keeps its place in the count but is not shown again
		passwordShown(): boolean {
			return !auth.configured || this.authDisabled;
		},
		startStep(): Step {
			return this.passwordShown ? "password" : "site";
		},
		// a missing password is mandatory, a disabled one only warns
		canSkip(): boolean {
			return auth.configured || this.authDisabled;
		},
		// the band icon of the first step lights up before the band turns into the progress bar
		startIconIndex(): number {
			return this.startStep === "site" ? 1 : 0;
		},
		// steps that add a meter, the step name doubles as checklist item and meter type
		meterItem(): OnboardingItem | undefined {
			return METER_ITEMS.includes(this.step) ? (this.step as OnboardingItem) : undefined;
		},
		// item -> configured devices with one formatted live value
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
			return {
				battery: this.batteryMeters.map((m) => meterRow(m, "soc")),
				pv: this.pvMeters.map((m) => meterRow(m, "power")),
				grid: this.gridMeter ? [meterRow(this.gridMeter, "power")] : [],
				loadpoint: this.chargingLoadpoints.map(loadpointRow),
				heater: this.heaterLoadpoints.map(loadpointRow),
			};
		},
		configuredDevices(): DeviceListItem[] {
			if (this.step === "control") {
				return [...this.devices["loadpoint"]!, ...this.devices["heater"]!];
			}
			return this.meterItem ? this.devices[this.meterItem]! : [];
		},
		footer(): boolean {
			return this.step !== "welcome" && this.step !== "done";
		},
		// the form the continue button submits, password and site live in one
		form(): string | undefined {
			if (this.step === "password" && !this.authDisabled) return "onboardingPassword";
			if (this.step === "site") return "onboardingSiteForm";
			return undefined;
		},
		continueLabel(): string {
			return this.form === "onboardingPassword"
				? this.$t("passwordModal.newPassword")
				: this.$t("config.general.continue");
		},
		// device steps continue once something exists
		canContinue(): boolean {
			return ["password", "site"].includes(this.step) || this.configuredDevices.length > 0;
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
		// the band collapses into a line first, then the line slides into place as the progress bar
		async start() {
			this.starting = true;
			await this.$nextTick();
			const band = this.$refs["band"] as HTMLElement;
			await Promise.all(band.getAnimations({ subtree: true }).map((a) => a.finished));
			document.documentElement.classList.add("onboarding-start");
			await this.goTo(this.startStep)?.finished;
			document.documentElement.classList.remove("onboarding-start");
			this.starting = false;
		},
		next() {
			this.goTo(STEPS[this.stepIndex + 1]!);
		},
		// forms advance through their saved event
		continueStep() {
			if (!this.form) this.next();
		},
		prev() {
			const previous = STEPS[this.stepIndex - 1];
			const shown = previous && (previous !== "password" || this.passwordShown);
			this.goTo(shown ? previous : "welcome");
		},
		// form field or primary action, never a text link or a "none" button
		focusFirst() {
			const modal = document.getElementById("onboardingModal")!;
			modal
				.querySelector<HTMLElement>("input, select, .btn-primary, .btn-outline-primary")
				?.focus({ preventScroll: true });
		},
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
			return this.fmtW(value, POWER_UNIT.AUTO);
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
/* one icon per step: the first lights up, then the icons dissolve into a line, the box keeps its height */
.band {
	height: 24px;
}
.band > :not(.line) {
	transition:
		color var(--evcc-transition-fast) ease,
		opacity var(--evcc-transition-fast) ease var(--evcc-transition-fast);
}
.band.bar > :not(.line) {
	opacity: 0;
}
.band.bar > .current {
	color: var(--bs-primary);
}
/* sits exactly where the progress bar lands, so the view transition only crossfades in place */
.line {
	position: absolute;
	inset: 0 0 auto;
	height: 4px;
	border-radius: var(--bs-border-radius);
	background-color: var(--evcc-gray-25);
	opacity: 0;
	transition: opacity var(--evcc-transition-fast) ease var(--evcc-transition-fast);
}
.band.bar > .line {
	opacity: 1;
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
/* leaving the welcome step: old out, the collapsed icon band continues as the progress bar, new in */
html.onboarding-start #onboardingModal .line,
html.onboarding-start #onboardingModal .progress {
	view-transition-name: onboarding-progress;
}
html.onboarding-start::view-transition-old(onboarding-body) {
	animation: 100ms ease-out both vt-fade-out;
}
html.onboarding-start::view-transition-new(onboarding-body) {
	animation: 150ms 200ms ease-in both vt-fade-in;
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
