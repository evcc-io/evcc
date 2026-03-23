<template>
	<JsonModal
		name="uilock"
		:title="$t('config.uilock.title')"
		endpoint="/config/uilock"
		state-key="uilock"
		:transform-read-values="transformReadValues"
		:transform-write-values="transformWriteValues"
		:confirm-remove="$t('config.uilock.confirmRemove')"
		@changed="$emit('changed')"
	>
		<template #default="{ values }">
			<FormRow
				id="uilockEnabled"
				:label="$t('config.uilock.labelEnabled')"
				:help="$t('config.uilock.descriptionEnabled')"
			>
				<div class="form-check form-switch">
					<input
						id="uilockEnabled"
						v-model="values.enabled"
						class="form-check-input"
						type="checkbox"
						role="switch"
					/>
				</div>
			</FormRow>

			<FormRow
				id="uilockTimeout"
				:label="$t('config.uilock.labelTimeout')"
				:help="$t('config.uilock.descriptionTimeout')"
			>
				<input
					id="uilockTimeout"
					v-model.number="values.timeout"
					class="form-control w-50"
					type="number"
					min="30"
					max="86400"
					step="1"
					required
				/>
			</FormRow>

			<FormRow
				id="uilockIps"
				:label="$t('config.uilock.labelIps')"
				:help="$t('config.uilock.descriptionIps')"
			>
				<textarea
					id="uilockIps"
					v-model="ipsText"
					class="form-control font-monospace"
					rows="3"
					spellcheck="false"
				/>
			</FormRow>

			<FormRow
				id="uilockTrusted"
				:label="$t('config.uilock.labelTrustedProxies')"
				:help="$t('config.uilock.descriptionTrustedProxies')"
				optional
			>
				<textarea
					id="uilockTrusted"
					v-model="trustedText"
					class="form-control font-monospace"
					rows="3"
					spellcheck="false"
				/>
			</FormRow>

			<FormRow
				id="uilockPin"
				:label="$t('config.uilock.labelPin')"
				:help="$t('config.uilock.descriptionPin')"
				optional
			>
				<input
					id="uilockPin"
					v-model="values.pin"
					class="form-control"
					type="password"
					autocomplete="new-password"
				/>
			</FormRow>
		</template>
	</JsonModal>
</template>

<script>
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";

const MASK = "***";

export default {
	name: "UILockModal",
	components: { JsonModal, FormRow },
	emits: ["changed"],
	data() {
		return {
			ipsText: "",
			trustedText: "",
		};
	},
	methods: {
		transformReadValues(v) {
			const ips = v?.ips || [];
			const trusted = v?.trustedProxies || [];
			this.ipsText = ips.join("\n");
			this.trustedText = trusted.join("\n");
			const pin = v?.pin;
			return {
				...v,
				pin: pin === MASK ? "" : pin || "",
			};
		},
		transformWriteValues(values) {
			const v = { ...values };
			delete v.pinConfigured;
			v.ips = this.ipsText
				.split(/[\n,]+/)
				.map((s) => s.trim())
				.filter(Boolean);
			v.trustedProxies = this.trustedText
				.split(/[\n,]+/)
				.map((s) => s.trim())
				.filter(Boolean);
			if (v.pin === MASK || v.pin === "") {
				delete v.pin;
			}
			return v;
		},
	},
};
</script>
