<template>
	<div class="test-result my-4 p-4">
		<div class="d-flex justify-content-between align-items-center">
			<strong>
				<span>{{ $t("config.validation.label") }}: </span>
				<span v-if="unknown">{{ $t("config.validation.unknown") }}</span>
				<span v-if="running">{{ $t("config.validation.running") }}</span>
				<span class="text-success" v-if="success">
					{{ $t("config.validation.success") }}
				</span>
				<span class="text-danger" v-if="failed">
					{{ $t("config.validation.failed") }}
				</span>
			</strong>
			<span
				v-if="running"
				class="spinner-border spinner-border-sm"
				role="status"
				aria-hidden="true"
			></span>
			<a v-else href="#" class="alert-link" @click.prevent="$emit('test')">
				{{ $t("config.validation.validate") }}
			</a>
		</div>
		<hr class="divider" v-if="error" />
		<div class="text-danger" v-if="error">
			{{ error }}
		</div>
		<hr class="divider" v-if="result" />
		<div v-if="result">
			<DeviceTags :tags="result" class="success-values" />
		</div>
	</div>
</template>

<script>
import DeviceTags from "./DeviceTags.vue";

export default {
	name: "TestResult",
	props: {
		success: Boolean,
		failed: Boolean,
		unknown: Boolean,
		running: Boolean,
		result: Object,
		error: String,
	},
	emits: ["test"],
	components: { DeviceTags },
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
