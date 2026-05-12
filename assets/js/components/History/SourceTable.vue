<template>
	<div class="source-section mt-4">
		<hr class="source-divider" />
		<h5 class="fw-normal text-muted mt-4 mb-3">
			{{ $t("main.history.sourceTitle") }}
		</h5>
		<div class="source-table-wrapper">
			<table class="table table-sm table-borderless table-striped source-table mb-0">
				<thead>
					<tr>
						<th
							v-for="(h, i) in headers"
							:key="i"
							class="text-muted fw-normal"
							:class="{ 'text-end': i > 0 }"
						>
							{{ h }}
						</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="(row, ri) in rows" :key="ri">
						<td
							v-for="(cell, ci) in row"
							:key="ci"
							class="tabular"
							:class="[{ 'text-end': ci > 0 }, cell.class]"
						>
							{{ cell.text }}
						</td>
					</tr>
				</tbody>
			</table>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";

export interface SourceTableCell {
	text: string;
	class?: string;
}

export default defineComponent({
	name: "SourceTable",
	props: {
		headers: { type: Array as PropType<string[]>, required: true },
		rows: { type: Array as PropType<SourceTableCell[][]>, required: true },
	},
});
</script>

<style scoped>
.source-divider {
	margin: 2rem -1rem;
	border: 0;
	border-top: 2px solid var(--evcc-background);
	opacity: 1;
	background: transparent;
}
@media (min-width: 576px) {
	.source-divider {
		margin: 2.5rem -1.5rem;
	}
}
.source-table-wrapper {
	overflow-x: auto;
}
.source-table {
	font-size: 0.875rem;
	width: auto;
}
.source-table th {
	white-space: nowrap;
	font-weight: normal;
	padding-top: 0;
	border-bottom: 1px solid var(--bs-border-color-translucent);
}
.source-table td {
	white-space: nowrap;
}
.source-table th,
.source-table td {
	padding: 0.5rem 1.25rem;
	border-right: 1px solid var(--bs-border-color-translucent);
}
.source-table th:last-child,
.source-table td:last-child {
	border-right: none;
}
</style>
