<template>
	<div class="btn-group">
		<a
			class="btn btn-outline-secondary"
			:href="hrefFor('xlsx')"
			download
			data-testid="download-default"
			@click="handleDownloadClick($event, hrefFor('xlsx'))"
		>
			{{ label }}
		</a>
		<button
			type="button"
			class="btn btn-outline-secondary dropdown-toggle dropdown-toggle-split"
			data-bs-toggle="dropdown"
			aria-expanded="false"
		>
			<span class="visually-hidden">{{ label }}</span>
		</button>
		<ul class="dropdown-menu">
			<li v-for="{ format, title } in formats" :key="format">
				<a
					class="dropdown-item"
					:href="hrefFor(format)"
					download
					:data-testid="`download-${format}`"
					@click="handleDownloadClick($event, hrefFor(format))"
				>
					{{ title }}
				</a>
			</li>
		</ul>
	</div>
</template>

<script>
import "bootstrap/js/dist/dropdown";
import { defineComponent } from "vue";
import { handleDownloadClick } from "@/utils/native";

export default defineComponent({
	name: "DownloadButton",
	props: {
		label: { type: String, required: true },
		href: { type: String, required: true },
	},
	computed: {
		formats() {
			return [
				{ format: "xlsx", title: this.$t("general.downloadXlsx") },
				{ format: "csv", title: this.$t("general.downloadCsv") },
				{ format: "json", title: this.$t("general.downloadJson") },
			];
		},
	},
	methods: {
		handleDownloadClick,
		hrefFor(format) {
			return `${this.href}${this.href.includes("?") ? "&" : "?"}format=${format}`;
		},
	},
});
</script>
