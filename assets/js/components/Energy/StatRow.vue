<template>
	<div :class="rowClass">
		<component
			:is="picks[index] ? 'button' : 'div'"
			v-for="(stat, index) in stats"
			:key="stat.key"
			:type="picks[index] ? 'button' : undefined"
			:class="itemClasses[index]"
			:aria-pressed="picks[index] ? stat.key === selected : undefined"
			:data-testid="`energy-stat-${stat.key}`"
			@click="select(stat)"
		>
			<Stat
				:label="stat.label"
				:number="stat.number"
				:format="stat.format"
				:sub="stat.sub"
				:tooltip="stat.tooltip"
				:value-class="stat.accent"
				:inline="inline"
			>
				<!-- a missing value, the way to the settings only while no tariff is set up -->
				<template v-if="stat.empty">
					<div class="text-muted small">{{ stat.empty }}</div>
					<router-link v-if="stat.configure" to="/config#tariffs" class="small">
						{{ $t("energy.stat.configure") }}
					</router-link>
				</template>
			</Stat>
		</component>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Stat from "./Stat.vue";
import type { StatItem } from "./types";

// numbers side by side: a grid of full size stats, or `inline` a compact row for a
// card header
export default defineComponent({
	name: "StatRow",
	components: { Stat },
	props: {
		stats: { type: Array as PropType<StatItem[]>, default: () => [] },
		inline: Boolean,
		itemClass: { type: String, default: "col-6" },
		// key of the chosen stat. Set, every stat is a button that asks to be chosen
		selected: String,
	},
	emits: ["select"],
	computed: {
		// which stats are buttons
		picks(): boolean[] {
			return this.stats.map((s) => this.selected !== undefined && s.selectable !== false);
		},
		// spread over the row they get below lg
		spreads(): boolean {
			return this.inline && this.stats.length > 1 && this.stats.length < 4;
		},
		rowClass(): string {
			if (!this.inline) return "row gx-4 gy-4";
			const row = "d-flex flex-wrap align-items-start column-gap-3 column-gap-lg-4 row-gap-2";
			if (!this.spreads) return row;
			// three know their row's width: too narrow to share it, they wrap left aligned
			const three = this.stats.length === 3 ? "row--three" : "";
			return `${row} flex-grow-1 justify-content-between ${three}`;
		},
		itemClasses(): string[] {
			return this.stats.map((_, index) => {
				if (!this.inline) return this.itemClass;
				// on a row of their own the tiles align towards its ends
				const last = index === this.stats.length - 1;
				const text = !this.spreads || !index ? "" : last ? "tile--end" : "tile--center";
				const align = `tile text-start ${text}`;
				if (!this.picks[index]) return align;
				const on = this.stats[index]?.key === this.selected ? "pick--on" : "";
				return `pick border-0 bg-transparent p-0 text-reset ${align} ${on}`;
			});
		},
	},
	methods: {
		select(stat: StatItem) {
			if (this.selected !== undefined && stat.selectable !== false)
				this.$emit("select", stat.key);
		},
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

/* on a row of their own three tiles share it evenly: start, center, end. The header
   is the query container, too narrow for one row they wrap left aligned */
.row--three {
	display: grid !important;
	grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
}
.tile--center {
	justify-self: center;
	text-align: center !important;
}
.tile--end {
	justify-self: end;
	text-align: end !important;
}
@container (max-width: 24rem) {
	.row--three {
		display: flex !important;
	}
	.row--three > .tile {
		text-align: start !important;
	}
}
/* beside the title the tiles keep their width, changing numbers must not move them */
@media (--lg-and-up) {
	.row--three {
		display: flex !important;
	}
	.tile--center,
	.tile--end {
		text-align: start !important;
	}
	.tile {
		min-width: 7.5rem;
	}
}
/* the chosen stat is underlined, while another is hovered that one instead */
.pick--on :deep(.label),
.pick--on :deep(.value),
.pick:hover :deep(.label),
.pick:hover :deep(.value) {
	text-decoration: underline;
}
div:has(.pick:hover) > .pick--on:not(:hover) :deep(.label),
div:has(.pick:hover) > .pick--on:not(:hover) :deep(.value) {
	text-decoration: none;
}
</style>
