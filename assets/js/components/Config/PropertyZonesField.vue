<template>
	<div class="zones-field">
		<div v-for="(zone, index) in zones" :key="index" class="zone-row mb-2">
			<div class="row g-2">
				<div class="col-md-3">
					<div class="d-flex w-50 w-min-200">
						<input
							:value="zone.price"
							type="number"
							step="any"
							class="form-control text-end"
							style="border-top-right-radius: 0; border-bottom-right-radius: 0"
							:placeholder="$t('config.tariff.zones.pricePlaceholder')"
							required
							@change="updateZonePrice(index, $event.target.value)"
						/>
						<span
							class="input-group-text"
							style="border-top-left-radius: 0; border-bottom-left-radius: 0"
							>{{ priceUnit }}</span
						>
					</div>
					<small class="text-muted">{{ $t("config.tariff.zones.price") }}</small>
				</div>
				<div class="col-md-3">
					<input
						:value="zone.hours"
						type="text"
						class="form-control"
						:placeholder="$t('config.tariff.zones.hoursPlaceholder')"
						@input="updateZoneField(index, 'hours', $event.target.value)"
					/>
					<small class="text-muted">{{ $t("config.tariff.zones.hours") }}</small>
				</div>
				<div class="col-md-2">
					<input
						:value="zone.days"
						type="text"
						class="form-control"
						:placeholder="$t('config.tariff.zones.daysPlaceholder')"
						@input="updateZoneField(index, 'days', $event.target.value)"
					/>
					<small class="text-muted">{{ $t("config.tariff.zones.days") }}</small>
				</div>
				<div class="col-md-3">
					<input
						:value="zone.months"
						type="text"
						class="form-control"
						:placeholder="$t('config.tariff.zones.monthsPlaceholder')"
						@input="updateZoneField(index, 'months', $event.target.value)"
					/>
					<small class="text-muted">{{ $t("config.tariff.zones.months") }}</small>
				</div>
				<div class="col-md-1 d-flex align-items-start">
					<button
						v-if="zones.length > 1"
						type="button"
						class="btn btn-link text-danger p-0"
						:aria-label="$t('config.tariff.zones.remove')"
						@click="removeZone(index)"
					>
						&times;
					</button>
				</div>
			</div>
		</div>
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

<script>
import "@h2d2/shopicons/es/regular/plus";
import formatter from "@/mixins/formatter";

export default {
	name: "PropertyZonesField",
	mixins: [formatter],
	props: {
		modelValue: {
			type: Array,
			default: () => [],
		},
		currency: { type: String, default: "EUR" },
	},
	emits: ["update:modelValue"],
	computed: {
		displayFactor() {
			return this.pricePerKWhDisplayFactor(this.currency);
		},
		priceUnit() {
			return this.pricePerKWhUnit(this.currency);
		},
		zones() {
			// Ensure we always have at least one zone for initial display
			// Convert prices from base unit to display unit
			const rawZones =
				this.modelValue && this.modelValue.length > 0
					? this.modelValue
					: [{ price: null, hours: "", days: "", months: "" }];

			return rawZones.map((zone) => {
				if (zone.price == null) return { ...zone, price: null };
				const value = zone.price * this.displayFactor;
				// Round to 6 decimals to eliminate floating-point errors
				return { ...zone, price: Math.round(value * 1e6) / 1e6 };
			});
		},
	},
	methods: {
		addZone() {
			const newZones = [...this.modelValue, { price: null, hours: "", days: "", months: "" }];
			this.$emit("update:modelValue", newZones);
		},
		removeZone(index) {
			const newZones = this.modelValue.filter((_, i) => i !== index);
			this.$emit("update:modelValue", newZones);
		},
		updateZonePrice(index, displayValue) {
			const newZones = [...this.modelValue];
			// Convert from display unit to base unit
			newZones[index] = {
				...newZones[index],
				price: displayValue ? parseFloat(displayValue) / this.displayFactor : null,
			};
			this.$emit("update:modelValue", newZones);
		},
		updateZoneField(index, field, value) {
			const newZones = [...this.modelValue];
			newZones[index] = {
				...newZones[index],
				[field]: value,
			};
			this.$emit("update:modelValue", newZones);
		},
	},
};
</script>

<style scoped>
.zones-field {
	width: 100%;
}

.zone-row small {
	display: block;
	font-size: 0.75rem;
	margin-top: 0.25rem;
}

.zone-row input {
	font-size: 0.875rem;
}

.zone-row .btn-link {
	font-size: 1.5rem;
	line-height: 1;
	text-decoration: none;
}

.zone-row .btn-link:hover {
	text-decoration: none;
}

.btn-outline-secondary {
	margin-left: -0.5rem;
}

.w-min-200 {
	min-width: min(200px, 100%);
}
</style>
