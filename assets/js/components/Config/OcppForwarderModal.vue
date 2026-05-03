<template>
	<JsonModal
		name="ocppforwarder"
		:title="$t('config.ocppforwarder.title')"
		:description="$t('config.ocppforwarder.description')"
		endpoint="/config/ocppforwarder"
		state-key="ocppforwarder"
		store-values-in-array
		disable-remove
		size="xl"
		@changed="$emit('changed')"
	>
		<template #default="{ values }: { values: OcppForwarderRule[] }">
			<div class="mb-3">
				<div v-for="(rule, index) in values" :key="index" data-testid="ocppforwarder-rule">
					<div class="d-block">
						<hr class="mt-5" />
						<h5>
							<div class="inner mb-4">
								{{ $t("config.ocppforwarder.rule", { number: index + 1 }) }}
							</div>
						</h5>
					</div>
					<div class="row d-inline d-lg-flex mb-3">
						<div class="col-lg-5" data-testid="charger-box">
							<div class="border rounded px-3 pt-4 pb-3">
								<div class="d-lg-block">
									<h5 class="box-heading">
										<div class="inner">
											{{ $t("config.ocppforwarder.charger") }}
										</div>
									</h5>
								</div>
								<FormRow
									:id="`ocppforwarderStationId-${index}`"
									:label="$t('config.ocppforwarder.stationId')"
									:help="$t('config.ocppforwarder.stationIdHelp')"
								>
									<input
										:id="`ocppforwarderStationId-${index}`"
										v-model="rule.stationId"
										type="text"
										class="form-control"
										placeholder="*"
										spellcheck="false"
										autocomplete="off"
										required
									/>
								</FormRow>
							</div>
						</div>
						<div
							class="col-lg-2 d-none d-lg-flex justify-content-center evcc-gray"
							style="padding-top: 2.5rem"
						>
							<shopicon-regular-arrowright
								size="l"
								class="flex-shrink-0"
							></shopicon-regular-arrowright>
						</div>
						<div class="col d-flex d-lg-none justify-content-center evcc-gray my-3">
							<shopicon-regular-arrowdown
								size="l"
								class="flex-shrink-0"
							></shopicon-regular-arrowdown>
						</div>
						<div class="col-lg-5" data-testid="upstream-box">
							<div class="border rounded px-3 pt-4 pb-3">
								<div class="d-lg-block">
									<h5 class="box-heading">
										<div class="inner">
											{{ $t("config.ocppforwarder.upstream") }}
										</div>
									</h5>
								</div>
								<FormRow
									:id="`ocppforwarderUpstreamUrl-${index}`"
									:label="$t('config.ocppforwarder.upstreamUrl')"
									:help="$t('config.ocppforwarder.upstreamUrlHelp')"
								>
									<div class="d-flex align-items-center gap-2">
										<input
											:id="`ocppforwarderUpstreamUrl-${index}`"
											v-model="rule.upstreamUrl"
											type="text"
											class="form-control"
											inputmode="url"
											spellcheck="false"
											autocomplete="off"
											required
										/>
										<span
											v-if="rule.upstreamUrl"
											class="badge flex-shrink-0"
											:class="upstreamStatusClass(rule.upstreamUrl)"
											:title="
												$t(
													`config.ocppforwarder.${upstreamStatusKey(rule.upstreamUrl)}Help`
												)
											"
											data-bs-toggle="tooltip"
										>
											{{
												$t(
													`config.ocppforwarder.${upstreamStatusKey(rule.upstreamUrl)}`
												)
											}}
										</span>
									</div>
								</FormRow>
								<FormRow
									:id="`ocppforwarderUpstreamStationId-${index}`"
									:label="$t('config.ocppforwarder.upstreamStationId')"
									:help="$t('config.ocppforwarder.upstreamStationIdHelp')"
								>
									<input
										:id="`ocppforwarderUpstreamStationId-${index}`"
										v-model="rule.upstreamStationId"
										type="text"
										class="form-control"
										:placeholder="rule.stationId"
										spellcheck="false"
										autocomplete="off"
									/>
								</FormRow>
								<FormRow
									:id="`ocppforwarderPassword-${index}`"
									:label="$t('config.ocppforwarder.password')"
									:help="$t('config.ocppforwarder.passwordHelp')"
								>
									<input
										:id="`ocppforwarderPassword-${index}`"
										v-model="rule.password"
										type="password"
										class="form-control"
										autocomplete="new-password"
									/>
								</FormRow>
								<FormRow
									:id="`ocppforwarderInsecure-${index}`"
									:label="$t('config.ocppforwarder.labelInsecure')"
								>
									<div class="d-flex">
										<input
											:id="`ocppforwarderInsecure-${index}`"
											v-model="rule.insecure"
											class="form-check-input"
											type="checkbox"
										/>
										<label
											class="form-check-label ms-2"
											:for="`ocppforwarderInsecure-${index}`"
										>
											{{ $t("config.ocppforwarder.labelCheckInsecure") }}
										</label>
									</div>
								</FormRow>
								<PropertyCollapsible>
									<template #advanced>
										<FormRow
											:id="`ocppforwarderCaCert-${index}`"
											:label="$t('config.ocppforwarder.labelCaCert')"
											optional
										>
											<PropertyCertField
												:id="`ocppforwarderCaCert-${index}`"
												v-model="rule.caCert"
											/>
										</FormRow>
										<FormRow
											:id="`ocppforwarderReadOnly-${index}`"
											:label="$t('config.ocppforwarder.readOnly.label')"
											:help="getReadOnlyHelp(rule.readOnly)"
										>
											<SelectGroup
												:id="`ocppforwarderReadOnly-${index}`"
												:model-value="rule.readOnly ? 'true' : 'false'"
												class="w-100"
												:options="readOnlyOptions"
												transparent
												@update:model-value="
													rule.readOnly = $event === 'true'
												"
											/>
										</FormRow>
									</template>
								</PropertyCollapsible>
							</div>
						</div>
					</div>
					<button
						type="button"
						class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray ms-auto"
						:aria-label="$t('config.general.remove')"
						tabindex="0"
						@click="values.splice(index, 1)"
					>
						<shopicon-regular-trash
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-trash>
						{{ $t("config.general.remove") }}
					</button>
				</div>

				<hr class="my-5" />

				<button
					type="button"
					class="d-flex btn btn-sm align-items-center gap-2 mb-5"
					:class="
						values.length === 0
							? 'btn-secondary'
							: 'btn-outline-secondary border-0 evcc-gray'
					"
					data-testid="ocppforwarder-add"
					tabindex="0"
					@click="addRule(values)"
				>
					<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
					{{ $t("config.ocppforwarder.add") }}
				</button>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/arrowright";
import "@h2d2/shopicons/es/regular/arrowdown";
import "@h2d2/shopicons/es/regular/plus";
import "@h2d2/shopicons/es/regular/trash";
import { defineComponent } from "vue";
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import SelectGroup from "@/components/Helper/SelectGroup.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import PropertyCertField from "./PropertyCertField.vue";
import type { OcppForwarderRule, OcppForwarderSession } from "@/types/evcc";
import store from "@/store";

export default defineComponent({
	name: "OcppForwarderModal",
	components: { JsonModal, FormRow, SelectGroup, PropertyCollapsible, PropertyCertField },
	emits: ["changed"],
	computed: {
		sessions(): OcppForwarderSession[] {
			return store.state?.ocppforwarderstatus || [];
		},
		readOnlyOptions() {
			return ["false", "true"].map((value) => ({
				value,
				name: this.$t(`config.ocppforwarder.option.${value}`),
			}));
		},
	},
	methods: {
		getReadOnlyHelp(readOnly?: boolean): string {
			return this.$t(`config.ocppforwarder.readOnly.help.${readOnly ? "true" : "false"}`);
		},
		// Returns true if any active session has an upstream connection to this URL.
		upstreamStatusKey(url: string): string {
			if (!this.sessions.length) return "upstreamIdle";
			const base = url.replace(/\/$/, "");
			const connected = this.sessions.some(
				(s) => s.upstreamUrl === base && s.upstreamConnected
			);
			return connected ? "upstreamConnected" : "upstreamDisconnected";
		},
		upstreamStatusClass(url: string): string {
			if (!this.sessions.length) return "bg-secondary";
			const base = url.replace(/\/$/, "");
			const connected = this.sessions.some(
				(s) => s.upstreamUrl === base && s.upstreamConnected
			);
			return connected ? "bg-success" : "bg-warning";
		},
		addRule(values: OcppForwarderRule[]) {
			values.push({ stationId: "", upstreamUrl: "" });
		},
	},
});
</script>

<style scoped>
h5 {
	position: relative;
	display: flex;
	top: -25px;
	margin-bottom: -0.5rem;
	padding: 0 0.5rem;
	justify-content: center;
}
h5.box-heading {
	top: -34px;
	margin-bottom: -24px;
}
h5 .inner {
	padding: 0 0.5rem;
	background-color: var(--evcc-box);
	font-weight: normal;
	color: var(--evcc-gray);
	text-align: center;
}
</style>
