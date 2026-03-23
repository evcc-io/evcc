<template>
	<div class="zones-field">
		<div
			v-for="(zone, index) in modelValue"
			:key="index"
			data-testid="property-zone"
			class="mb-2"
		>
			<!-- Summary View (read-only) -->
			<PropertyZoneForm
				v-if="editIndex === index"
				:zone="zone"
				:currency="currency"
				:index="index"
				@save="saveEdit"
				@cancel="cancelEdit"
			/>
			<PropertyZoneSummary
				v-else
				:zone="zone"
				:currency="currency"
				@edit="startEdit(index)"
				@remove="removeZone(index)"
			/>
		</div>

		<!-- Add zone button -->
		<div class="d-flex align-items-center">
			<button
				type="button"
				class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
				@click="addZone"
			>
				<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
				{{ $t("config.tariff.zones.add") }}
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import { type PropType } from "vue";
import "@h2d2/shopicons/es/regular/plus";
import formatter from "@/mixins/formatter";
import PropertyZoneSummary from "./PropertyZoneSummary.vue";
import PropertyZoneForm from "./PropertyZoneForm.vue";
import { CURRENCY, type Zone } from "@/types/evcc";

export default {
	name: "PropertyZonesField",
	components: { PropertyZoneSummary, PropertyZoneForm },
	mixins: [formatter],
	props: {
		modelValue: {
			type: Array as PropType<Zone[]>,
			default: () => [],
		},
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
	},
	emits: ["update:modelValue"],
	data() {
		return {
			editIndex: null as number | null,
		};
	},
	methods: {
		addZone() {
			const currentZones = this.modelValue || [];
			const newZones = [...currentZones, { price: 0, hours: "", days: "", months: "" }];
			this.$emit("update:modelValue", newZones);
			// Automatically start editing the new zone
			this.$nextTick(() => {
				this.startEdit(newZones.length - 1);
			});
		},
		removeZone(index: number) {
			const newZones = this.modelValue.filter((_, i) => i !== index);
			this.$emit("update:modelValue", newZones);
		},
		startEdit(index: number) {
			this.editIndex = index;
		},
		saveEdit(convertedZone: Zone) {
			const index = this.editIndex;
			if (index === null) return;
			const newZones = [...this.modelValue];
			newZones[index] = convertedZone;
			this.$emit("update:modelValue", newZones);
			this.editIndex = null;
		},
		cancelEdit() {
			const index = this.editIndex;
			if (index !== null) {
				const zone = this.modelValue[index];
				if (zone && zone.price === null) {
					const newZones = this.modelValue.filter((_, i) => i !== index);
					this.$emit("update:modelValue", newZones);
				}
			}
			this.editIndex = null;
		},
	},
};
</script>

<style scoped>
.zones-field {
	width: 100%;
}
</style>
