<template>
	<div
		class="alert my-4"
		:class="{
			'alert-secondary': unknown || running,
			'alert-success': success,
			'alert-danger': failed,
		}"
		role="alert"
	>
		<div class="d-flex justify-content-between align-items-center">
			<div>
				{{ $t("config.validation.label") }}:
				<span v-if="unknown">{{ $t("config.validation.unknown") }}</span>
				<span v-if="running">{{ $t("config.validation.running") }}</span>
				<strong v-if="success">{{ $t("config.validation.success") }}</strong>
				<strong v-if="failed">{{ $t("config.validation.failed") }}</strong>
			</div>
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
		<hr v-if="error" />
		<div v-if="error">
			{{ error }}
		</div>
		<hr v-if="result" />
		<div v-if="result">
			<DeviceTags :tags="result" />
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
