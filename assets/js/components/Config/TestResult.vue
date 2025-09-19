<template>
	<div class="test-result my-4 p-4" data-testid="test-result">
		<div class="d-flex justify-content-between align-items-center">
			<strong>
				<span>{{ $t("config.validation.label") }}: </span>
				<span v-if="isUnknown">{{ $t("config.validation.unknown") }}</span>
				<span v-if="isRunning">{{ $t("config.validation.running") }}</span>
				<span v-if="isSuccess" class="text-success">
					{{ $t("config.validation.success") }}
				</span>
				<span v-if="isError" class="text-danger">
					{{ $t("config.validation.failed") }}
				</span>
			</strong>
			<div v-if="!showTokenRequired">
				<span
					v-if="isRunning"
					class="spinner-border spinner-border-sm"
					role="status"
					aria-hidden="true"
				></span>
				<a v-else href="#" class="alert-link" tabindex="0" @click.prevent="test">
					{{ $t("config.validation.validate") }}
				</a>
			</div>
		</div>
		<hr v-if="showTokenRequired" class="divider" />
		<SponsorTokenRequired v-if="showTokenRequired" compact />
		<hr v-if="error" class="divider" />
		<div v-if="error" class="text-danger">
			{{ error }}
		</div>
		<hr v-if="result" class="divider" />
		<div v-if="result">
			<DeviceTags :tags="result" class="success-values" />
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import DeviceTags from "./DeviceTags.vue";
import SponsorTokenRequired from "./DeviceModal/SponsorTokenRequired.vue";

export default defineComponent({
	name: "TestResult",
	components: { DeviceTags, SponsorTokenRequired },
	props: {
		isUnknown: Boolean,
		isSuccess: Boolean,
		isError: Boolean,
		isRunning: Boolean,
		result: Object as PropType<Record<string, any> | null>,
		error: String as PropType<string | null>,
		sponsorTokenRequired: Boolean,
	},
	emits: ["test"],
	data() {
		return {
			showTokenRequired: false,
		};
	},
	watch: {
		sponsorTokenRequired() {
			this.showTokenRequired = false;
		},
	},
	methods: {
		test() {
			if (this.sponsorTokenRequired) {
				this.showTokenRequired = true;
			} else {
				this.$emit("test");
			}
		},
	},
});
</script>
<style scoped>
.test-result {
	border: 1px solid var(--bs-border-color);
	border-radius: var(--bs-border-radius);
}
.divider {
	border-top-color: 1px solid var(--bs-border-color);
	margin-left: -1.5rem;
	margin-right: -1.5rem;
}
</style>
