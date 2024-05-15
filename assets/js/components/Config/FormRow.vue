<!-- eslint-disable vue/no-v-html -->
<template>
	<div class="mb-4">
		<label :for="id">
			<div class="form-label">
				{{ label }}
				<small v-if="optional" class="evcc-gray">{{ $t("config.form.optional") }}</small>
			</div>
		</label>
		<div class="w-100">
			<slot />
		</div>
		<div class="form-text evcc-gray">
			<div v-if="example">{{ $t("config.form.example") }}: {{ example }}</div>
			<div v-if="help">
				<span class="text-gray" v-html="helpHtml"></span>
				<a class="ms-1 text-gray" v-if="link" :href="link" target="_blank">
					{{ $t("config.general.docsLink") }}
				</a>
			</div>
		</div>
	</div>
</template>

<script>
import linkify from "../../utils/linkify";
import { docsPrefix } from "../../i18n";

export default {
	name: "FormRow",
	props: {
		id: String,
		label: String,
		help: String,
		optional: Boolean,
		example: String,
		docsLink: String,
	},
	computed: {
		helpHtml() {
			return linkify(this.help);
		},
		link() {
			return this.docsLink ? `${docsPrefix()}${this.docsLink}` : null;
		},
	},
};
</script>
