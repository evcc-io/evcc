<template>
	<div if="plan" class="small text-muted">
		<div>Ladung: ~{{ energyFormatted }} in {{ planDuration }}</div>
		<div>
			Zeitfenster:
			<span v-if="planStart && planEnd">{{ planStart }} bis {{ planEnd }}</span>
			<span v-else-if="planStart">{{ planStart }} bis unbekannt</span>
			<span v-else>noch unbekannt</span>
		</div>
		<div v-if="isCo2 && !incompletePlan">CO₂ Menge: {{ planCO2 }}</div>
		<div v-if="!isCo2 && !incompletePlan">Energiepreis: {{ planPrice }}</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";

export default {
	name: "TargetChargePlanMinimal",
	mixins: [formatter],
	props: {
		duration: Number,
		plan: Array,
		unit: String,
		power: Number,
	},
	computed: {
		planDuration() {
			return this.fmtShortDuration(this.duration / 1e9, true);
		},
		lastSlot() {
			if (this.plan?.length) {
				return this.plan[this.plan?.length - 1];
			}
			return null;
		},
		firstSlot() {
			if (this.plan?.length) {
				return this.plan[0];
			}
			return null;
		},
		planStart() {
			if (this.firstSlot) {
				return this.weekdayTime(new Date(this.firstSlot.start));
			}
			return null;
		},
		planEnd() {
			if (this.lastSlot && !this.incompletePlan) {
				return this.weekdayTime(new Date(this.lastSlot.end));
			}
			return null;
		},
		total() {
			if (this.plan?.length) {
				return this.plan.reduce(
					(total, slot) => {
						const hours =
							(new Date(slot.end).getTime() - new Date(slot.start).getTime()) / 3.6e6;
						const energy = hours * (this.power / 1e3);
						total.energy += energy;
						total.price += hours * slot.price * energy;
						console.log({ energy, hours, price: hours * slot.price * energy });
						return total;
					},
					{ energy: 0, price: 0 }
				);
			}
			return { energy: 0, price: 0 };
		},
		totalPrice() {
			return this.total.price;
		},
		incompletePlan() {
			return this.totalEnergy - this.total.energy > 1;
		},
		durationHours() {
			return this.duration / 3.6e12;
		},
		totalEnergy() {
			return (this.power / 1e3) * this.durationHours;
		},
		pricePerKwh() {
			return this.totalPrice / this.totalEnergy;
		},
		isCo2() {
			return this.unit === "gCO2eq";
		},
		planCO2() {
			return `${Math.round(this.pricePerKwh)}g/kWh`;
		},
		planPrice() {
			return this.fmtPricePerKWh(this.pricePerKwh, this.unit);
		},
		energyFormatted() {
			return this.fmtKWh(this.totalEnergy * 1e3, true, true, 1);
		},
	},
};
</script>

<style scoped></style>
