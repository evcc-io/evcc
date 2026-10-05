<template>
	<div class="btn-group">
		<a
			class="btn btn-outline-secondary"
			:href="hrefFor('xlsx')"
			download
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
			<li v-for="format in formats" :key="format">
				<a
					class="dropdown-item"
					:href="hrefFor(format)"
					download
					@click="handleDownloadClick($event, hrefFor(format))"
				>
					{{ format.toUpperCase() }}
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
			return ["xlsx", "csv", "json"];
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

<style scoped>
.dropdown-menu {
	--bs-dropdown-min-width: 0;
}
</style>
