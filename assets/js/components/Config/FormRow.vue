<!-- eslint-disable vue/no-v-html -->
<template>
	<div class="mb-4">
		<label :for="id">
			<div class="form-label">
				{{ label }}
				<small v-if="deprecated" class="evcc-gray">{{
					$t("config.form.deprecated")
				}}</small>
				<small v-else-if="optional" class="evcc-gray">{{
					$t("config.form.optional")
				}}</small>
			</div>
		</label>
		<div class="w-100">
			<slot />
		</div>
		<div v-if="error" class="invalid-feedback d-block">{{ error }}</div>
		<div class="form-text evcc-gray">
			<div v-if="example" class="hyphenate">
				{{ $t("config.form.example") }}: {{ example }}
			</div>
			<div v-if="help">
				<Markdown :markdown="help" class="text-gray hyphenate" />
				<a v-if="link" class="text-gray" :href="link" target="_blank">
					{{ $t("config.general.docsLink") }}
				</a>
			</div>
			<div v-if="danger" class="alert alert-danger my-2" role="alert">
				<strong>{{ $t("config.form.danger") }}:</strong> {{ danger }}
			</div>
		</div>
	</div>
</template>

<script>
import { docsPrefix } from "@/i18n";
import Markdown from "./Markdown.vue";

export default {
	name: "FormRow",
	components: { Markdown },
	props: {
		id: String,
		label: String,
		help: String,
		optional: Boolean,
		deprecated: Boolean,
		error: String,
		danger: String,
		example: String,
		docsLink: String,
	},
	computed: {
		link() {
			return this.docsLink ? `${docsPrefix()}${this.docsLink}` : null;
		},
	},
};
</script>
