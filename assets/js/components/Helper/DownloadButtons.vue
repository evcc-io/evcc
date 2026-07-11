<template>
	<div class="btn-group" role="group" :aria-label="label">
		<span class="btn btn-outline-secondary disabled label">{{ label }}</span>
		<a
			class="btn btn-outline-secondary"
			:href="csvHref"
			download
			data-testid="download-csv"
			@click="handleDownloadClick($event, csvHref)"
		>
			CSV
		</a>
		<a
			class="btn btn-outline-secondary"
			:href="xlsxHref"
			download
			data-testid="download-xlsx"
			@click="handleDownloadClick($event, xlsxHref)"
		>
			XLSX
		</a>
	</div>
</template>

<script>
import { defineComponent } from "vue";
import { handleDownloadClick } from "@/utils/native";

export default defineComponent({
	name: "DownloadButtons",
	props: {
		label: { type: String, required: true },
		csvHref: { type: String, required: true },
		xlsxHref: { type: String, required: true },
	},
	methods: { handleDownloadClick },
});
</script>

<style scoped>
/* label reads as static context, not a clickable option */
.label {
	color: var(--evcc-default-text);
	opacity: 1;
}
</style>
