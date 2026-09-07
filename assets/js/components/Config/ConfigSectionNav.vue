<template>
	<nav class="nav-list round-box box-pull-out p-0" data-testid="config-section-nav">
		<button
			v-for="section in sections"
			:key="section.slug"
			type="button"
			class="nav-row d-flex align-items-center gap-3 w-100 text-start px-3"
			:data-slug="section.slug"
			@click="open(section.slug)"
		>
			<div class="nav-icon d-flex align-items-center justify-content-center flex-shrink-0">
				<component :is="section.icon" v-if="section.icon" />
			</div>
			<div class="flex-grow-1 overflow-hidden">
				<div class="fw-bold text-truncate">{{ section.title }}</div>
				<div v-if="section.subline" class="small text-gray text-truncate">
					{{ section.subline }}
				</div>
			</div>
			<span
				v-if="section.error"
				class="circle-badge bg-danger flex-shrink-0"
				data-testid="section-error"
			>
				<span class="visually-hidden">{{ $t("general.error") }}</span>
			</span>
			<span
				v-else-if="section.warning"
				class="circle-badge bg-warning flex-shrink-0"
				data-testid="section-warning"
			>
				<span class="visually-hidden">{{ $t("general.warning") }}</span>
			</span>
			<span v-if="section.count" class="fw-bold text-gray flex-shrink-0">
				{{ section.count }}
			</span>
			<ChevronRight class="chevron flex-shrink-0" />
		</button>
	</nav>
</template>

<script lang="ts">
import { defineComponent, type Component, type PropType } from "vue";
import ChevronRight from "../MaterialIcon/ChevronRight.vue";
import { hapticFeedback } from "@/utils/haptic";

export interface SectionEntry {
	slug: string;
	title: string;
	icon?: Component | string;
	count?: number;
	error?: boolean;
	warning?: boolean;
	subline?: string;
}

export default defineComponent({
	name: "ConfigSectionNav",
	components: { ChevronRight },
	props: {
		sections: { type: Array as PropType<SectionEntry[]>, required: true },
	},
	emits: ["open"],
	methods: {
		open(slug: string) {
			hapticFeedback("light");
			this.$emit("open", slug);
		},
	},
});
</script>

<style scoped>
.nav-row {
	background: none;
	border: 0;
	border-bottom: 1px solid var(--bs-border-color-translucent);
	color: var(--evcc-default-text);
	min-height: 3.5rem;
	padding-top: 0.5rem;
	padding-bottom: 0.5rem;
}
/* carry the parent .round-box rounding so focus rings and :active follow it */
.nav-row:first-child {
	border-top-left-radius: 1rem;
	border-top-right-radius: 1rem;
}
.nav-row:last-child {
	border-bottom: none;
	border-bottom-left-radius: 1rem;
	border-bottom-right-radius: 1rem;
}
.nav-row:active {
	background: var(--evcc-box-border);
}
/* uniform icon slot; font-size 0 collapses the shopicons' inner inline-svg line box */
.nav-icon > * {
	display: block;
	width: 1.5rem;
	height: 1.5rem;
	font-size: 0;
}
.chevron {
	color: var(--evcc-gray);
	margin-right: -0.5rem;
}
</style>
