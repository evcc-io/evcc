<template>
	<div class="mb-3">
		<label :for="id">
			<div class="form-label">
				{{ label }}
				<small v-if="optional">{{ $t("config.form.optional") }}</small>
			</div>
		</label>
		<div :class="smallValue ? 'w-50' : 'w-100'">
			<slot />
		</div>
		<div class="form-text">
			<div v-if="example">{{ $t("config.form.example") }}: {{ example }}</div>
			<div v-if="help" v-linkify:options="{ target: '_blank' }">{{ help }}</div>
		</div>
	</div>
</template>

<script>
export default {
	name: "FormRow",
	props: {
		id: String,
		label: String,
		help: String,
		optional: Boolean,
		smallValue: Boolean,
		example: String,
	},
};
</script>
