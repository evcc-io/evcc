<template>
	<div class="root">
		<h1>Plan</h1>
		<div class="prices">
			<div class="me-3">
				<shopicon-regular-powersupply
					class="gridIcon"
					size="m"
				></shopicon-regular-powersupply>
			</div>
			<div v-for="price in priceSlots" :key="price.start" class="price">
				{{ fmtPricePerKWh(price.price, currency).replace("/kWh", "") }}
				<div class="box" :style="priceStyle(price.price)"></div>
			</div>
		</div>
		<div class="divider">
			<div class="divider_line"></div>
			<div class="divider_target"></div>
		</div>
		<div class="chargingSlots">
			<div class="me-3">
				<shopicon-regular-lightning
					class="chargingIcon"
					size="m"
				></shopicon-regular-lightning>
			</div>
			<div
				v-for="chargingSlot in chargingSlots"
				:key="chargingSlot.start"
				class="chargingSlot"
			>
				<!--2,3 h-->
				<div class="d-flex h-100 justify-content-between align-items-center">
					<div>14:30</div>
					<div>18:00</div>
				</div>
			</div>
			<div class="chargingSlot chargingSlot--finish">
				<!--2,3 h-->
				<div class="d-flex h-100 justify-content-center align-items-center">
					<div>21:30</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/lightning";
import "@h2d2/shopicons/es/regular/powersupply";
import formatter from "../mixins/formatter";

export default {
	name: "TargetChargePlan",
	mixins: [formatter],
	props: {
		priceSlots: Array,
		co2Slots: Array,
		chargingSlots: Array,
		currency: String,
		energyPrice: Number,
		targetTime: String,
	},
	computed: {
		maxPrice() {
			let result = 0;
			this.priceSlots.forEach(({ price }) => {
				result = Math.max(result, price);
			});
			return result;
		},
	},
	methods: {
		priceStyle(price) {
			return {
				height: `${(100 / this.maxPrice) * price}%`,
				opacity: price < 0.3 ? 1 : 0.4,
			};
		},
	},
};
</script>

<style scoped>
.root {
	overflow: hidden;
	--height: 80px;
	position: relative;
}
.prices {
	display: flex;
	height: var(--height);
	justify-content: stretch;
	align-items: flex-end;
}
.price {
	flex-basis: 0;
	flex-grow: 1;
	flex-shrink: 1;
	margin: 4px;
	margin-bottom: 0;
	text-align: center;
	height: 100%;
	display: flex;
	justify-content: flex-end;
	flex-direction: column;
}
.price:last-child .box {
	background: transparent;
	border: 2px solid var(--bs-primary);
}
.box {
	background: var(--bs-primary);
	border-radius: 12px;
}
.chargingSlots {
	display: flex;
	align-items: center;
}
.chargingSlot {
	margin-left: 16.5%;
	width: 25%;
	background-color: var(--bs-primary);
	height: 24px;
	border-radius: 6px;
	padding: 0 6px;
	background-image: linear-gradient(
		45deg,
		rgba(255, 255, 255, 0.15) 25%,
		transparent 25%,
		transparent 50%,
		rgba(255, 255, 255, 0.15) 50%,
		rgba(255, 255, 255, 0.15) 75%,
		transparent 75%,
		transparent
	);
	background-size: 50px 50px;

	color: white;
}

.chargingSlot--finish {
	margin-left: 34%;
	width: auto;
	color: var(--evcc-default-text);
	background-color: transparent;
}

.gridIcon,
.chargingIcon {
	color: var(--bs-primary);
}
.divider {
	height: 2rem;
	position: relative;
}
.divider_line {
	position: absolute;
	top: 50%;
	right: 0;
	left: 0;
	border-bottom: 1px solid var(--bs-gray-light);
}
.divider_target {
	position: absolute;
	top: 30%;
	left: 84%;
	width: 1px;
	height: 40%;
	background: var(--bs-gray-light);
}
.target {
	position: absolute;
	top: 0;
	bottom: 0;
	left: 87.5%;
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-bottom: 6px;
}
.target_bar {
	background: var(--bs-gray-light);
	width: 1px;
	flex-grow: 1;
	margin: 6px 0;
}
</style>
