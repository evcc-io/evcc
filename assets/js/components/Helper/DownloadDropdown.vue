<template>
	<div class="dropdown">
		<button
			type="button"
			class="btn btn-outline-secondary dropdown-toggle d-flex align-items-center gap-2"
			data-bs-toggle="dropdown"
			aria-expanded="false"
			data-testid="download-dropdown"
		>
			<shopicon-regular-download size="s" class="flex-shrink-0"></shopicon-regular-download>
			<span class="text-truncate">{{ label }}</span>
		</button>
		<ul class="dropdown-menu">
			<li>
				<a
					class="dropdown-item"
					:href="csvHref"
					download
					data-testid="download-csv"
					@click="handleDownloadClick($event, csvHref)"
				>
					CSV
				</a>
			</li>
			<li>
				<a
					class="dropdown-item"
					:href="xlsxHref"
					download
					data-testid="download-xlsx"
					@click="handleDownloadClick($event, xlsxHref)"
				>
					XLSX
				</a>
			</li>
		</ul>
	</div>
</template>

<script>
import { defineComponent } from "vue";
import "@h2d2/shopicons/es/regular/download";
import { handleDownloadClick } from "@/utils/native";

export default defineComponent({
	name: "DownloadDropdown",
	props: {
		label: { type: String, required: true },
		csvHref: { type: String, required: true },
		xlsxHref: { type: String, required: true },
	},
	methods: { handleDownloadClick },
});
</script>
