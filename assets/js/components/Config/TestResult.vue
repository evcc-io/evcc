<template>
	<div class="test-result my-4 p-4">
		<div class="d-flex justify-content-between align-items-center">
			<strong>
				<span>{{ $t("config.validation.label") }}: </span>
				<span v-if="unknown">{{ $t("config.validation.unknown") }}</span>
				<span v-if="running">{{ $t("config.validation.running") }}</span>
				<span v-if="success" class="text-success">
					{{ $t("config.validation.success") }}
				</span>
				<span v-if="failed" class="text-danger">
					{{ $t("config.validation.failed") }}
				</span>
			</strong>
			<span
				v-if="running"
				class="spinner-border spinner-border-sm"
				role="status"
				aria-hidden="true"
			></span>
			<a v-else href="#" class="alert-link" tabindex="0" @click.prevent="$emit('test')">
				{{ $t("config.validation.validate") }}
			</a>
		</div>
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

<script>
import DeviceTags from "./DeviceTags.vue";

export default {
	name: "TestResult",
	components: { DeviceTags },
	props: {
		success: Boolean,
		failed: Boolean,
		unknown: Boolean,
		running: Boolean,
		result: Object,
		error: String,
	},
	emits: ["test"],
};
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
