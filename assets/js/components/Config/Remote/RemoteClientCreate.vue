<template>
	<div>
		<p>{{ $t("config.remote.addClientDescription") }}</p>
		<form @submit.prevent="submit">
			<div class="row">
				<div class="col-md-6">
					<FormRow
						id="newClientDeviceName"
						:label="$t('config.remote.deviceName')"
						example="blue-iphone"
					>
						<input
							id="newClientDeviceName"
							ref="deviceNameInput"
							v-model="username"
							name="deviceName"
							type="text"
							class="form-control border"
							pattern="[^:]+"
							autocomplete="off"
							data-lpignore="true"
							data-1p-ignore
							data-form-type="other"
							required
						/>
					</FormRow>
				</div>
				<div class="col-md-6">
					<FormRow id="newClientExpiration" :label="$t('config.remote.expiration')">
						<select
							id="newClientExpiration"
							v-model="expiresIn"
							class="form-select border"
						>
							<option
								v-for="opt in expirationOptions"
								:key="opt.value"
								:value="opt.value"
							>
								{{ opt.name }}
							</option>
						</select>
					</FormRow>
				</div>
			</div>
			<div class="d-flex justify-content-between mt-3">
				<button type="button" class="btn btn-outline-secondary" @click="$emit('cancel')">
					{{ $t("config.general.cancel") }}
				</button>
				<button type="submit" class="btn btn-primary">
					{{ $t("config.remote.createClient") }}
				</button>
			</div>
		</form>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import FormRow from "../FormRow.vue";
import formatter from "@/mixins/formatter";

export default defineComponent({
	name: "RemoteClientCreate",
	components: { FormRow },
	mixins: [formatter],
	emits: ["cancel", "submit"],
	data() {
		return {
			username: "",
			expiresIn: 0,
		};
	},
	computed: {
		expirationOptions() {
			const HOUR = 60 * 60;
			const DAY = 24 * HOUR;
			return [
				{ value: 0, name: this.$t("config.remote.expirationNone") },
				{ value: HOUR, name: this.fmtDurationParts({ hours: 1 }) },
				{ value: DAY, name: this.fmtDurationParts({ days: 1 }) },
				{ value: 7 * DAY, name: this.fmtDurationParts({ weeks: 1 }) },
			];
		},
	},
	mounted() {
		(this.$refs["deviceNameInput"] as HTMLInputElement | undefined)?.focus();
	},
	methods: {
		submit() {
			const username = this.username.trim();
			if (!username) return;
			this.$emit("submit", { username, expiresIn: this.expiresIn });
		},
	},
});
</script>
