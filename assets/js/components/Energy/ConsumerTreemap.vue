<template>
	<div data-testid="consumer-treemap">
		<div ref="root" class="treemap position-relative">
			<div
				v-for="tile in layout"
				:key="tile.name"
				v-tooltip="`${tile.name}: ${fmtKWh(tile.value)}`"
				class="tile position-absolute d-flex flex-column justify-content-center align-items-center overflow-hidden text-center"
				:class="{
					'tile--label': tile.width >= 56 && tile.height >= 36,
					'tile--dim': selected && selected !== tile.key,
				}"
				role="button"
				tabindex="0"
				@click="$emit('select', tile.key)"
				@keydown.enter.prevent="$emit('select', tile.key)"
				:style="{
					left: `${tile.x}px`,
					top: `${tile.y}px`,
					width: `${tile.width}px`,
					height: `${tile.height}px`,
					backgroundColor: tile.color,
					borderRadius: tile.radius,
				}"
			>
				<span class="fw-bold text-truncate mw-100 px-1">{{ tile.name }}</span>
				<span class="text-truncate mw-100 px-1">{{ fmtKWh(tile.value) }}</span>
			</div>
		</div>
		<LegendList
			class="mt-4"
			:legends="legends"
			:device-colors="deviceColors"
			columns
			@focus="$emit('select', $event.focusKey)"
		/>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import colors, { resolveColors } from "@/colors";
import { CURRENCY, type DeviceColors } from "@/types/evcc";
import type { ConsumerEnergy } from "./types";
import LegendList from "../Sessions/LegendList.vue";
import type { Legend } from "../Sessions/types";
import { squarify, type TreemapRect } from "./treemap";

// wide tiles read better than squares, phones get a taller map with squarer tiles
const ASPECT = 2;
const ASPECT_NARROW = 1.2;
const NARROW_WIDTH = 576;
const GAP = 2;
const RADIUS = 8;

export const OTHERS = "others";

interface Tile {
	key: string; // consumer title or OTHERS
	name: string;
	value: number;
	color: string;
}

export default defineComponent({
	name: "ConsumerTreemap",
	components: { LegendList },
	mixins: [formatter],
	props: {
		consumers: { type: Array as PropType<ConsumerEnergy[]>, default: () => [] },
		others: { type: Number, default: 0 },
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
		price: Number, // per kWh, undefined without tariff data
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		selected: { type: String as PropType<string | null>, default: null },
	},
	emits: ["select"],
	data() {
		return { width: 0, height: 0, observer: null as ResizeObserver | null };
	},
	computed: {
		palette(): Record<string, string> {
			const titles = this.consumers.map((e) => e.title).sort();
			return resolveColors(titles, this.deviceColors);
		},
		tiles(): Tile[] {
			const list = this.consumers
				.filter((e) => e.energy > 0)
				.map((e) => ({
					key: e.title,
					name: e.title,
					value: e.energy,
					color: this.palette[e.title] || "",
				}));
			if (this.others > 0) {
				list.push({
					key: OTHERS,
					name: this.$t("energy.consumers.others"),
					value: this.others,
					color: colors.muted || "",
				});
			}
			return list.sort((a, b) => b.value - a.value);
		},
		layout(): (Tile & TreemapRect & { radius: string })[] {
			const rects = squarify(
				this.tiles.map((t) => t.value),
				this.width,
				this.height,
				this.width < NARROW_WIDTH ? ASPECT_NARROW : ASPECT
			);
			const eps = 0.5;
			return this.tiles.map((t, i) => {
				const r = rects[i] as TreemapRect;
				// round only the corners on the outside of the map
				const left = r.x < eps;
				const top = r.y < eps;
				const right = r.x + r.width > this.width - eps;
				const bottom = r.y + r.height > this.height - eps;
				const radius = [left && top, right && top, right && bottom, left && bottom]
					.map((outer) => (outer ? `${RADIUS}px` : "0"))
					.join(" ");
				return {
					...t,
					x: r.x + GAP / 2,
					y: r.y + GAP / 2,
					width: Math.max(0, r.width - GAP),
					height: Math.max(0, r.height - GAP),
					radius,
				};
			});
		},
		legends(): Legend[] {
			return this.tiles.map((t) => ({
				label: t.name,
				color: t.color,
				value:
					this.price === undefined
						? this.fmtKWh(t.value)
						: [this.fmtKWh(t.value), this.fmtCost(t.value * this.price)],
				// pickable only for real devices, "Others" keeps the muted color
				id: t.key === OTHERS ? undefined : t.key,
				focusable: true,
				focusKey: t.key,
				dim: !!this.selected && this.selected !== t.key,
			}));
		},
	},
	mounted() {
		const root = this.$refs["root"] as HTMLElement;
		this.observer = new ResizeObserver(() => this.measure());
		this.observer.observe(root);
		this.measure();
	},
	beforeUnmount() {
		this.observer?.disconnect();
	},
	methods: {
		measure() {
			const root = this.$refs["root"] as HTMLElement | undefined;
			if (!root) return;
			this.width = root.clientWidth;
			this.height = root.clientHeight;
		},
		fmtCost(v: number): string {
			return `${this.fmtMoney(v, this.currency)} ${this.fmtCurrencySymbol(this.currency)}`;
		},
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

.treemap {
	height: 360px;
}
@media (--sm-and-up) {
	.treemap {
		height: 260px;
	}
}
.tile {
	color: var(--bs-white);
	font-size: 12px;
	line-height: 1.3;
}
.tile--dim {
	opacity: 0.35;
}
.tile > span {
	visibility: hidden;
}
.tile--label > span {
	visibility: visible;
}
</style>
