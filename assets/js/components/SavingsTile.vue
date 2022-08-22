<template>
	<div class="flex-grow-1 d-flex mb-4 mb-lg-0 align-items-center align-items-lg-start">
		<shopicon-regular-sun v-if="icon === 'sun'" class="tile-icon"></shopicon-regular-sun>
		<shopicon-regular-coinjar
			v-if="icon === 'coinjar'"
			class="tile-icon"
		></shopicon-regular-coinjar>
		<shopicon-regular-receivepayment
			v-if="icon === 'receivepayment'"
			class="tile-icon"
		></shopicon-regular-receivepayment>
		<div class="ms-3 d-flex flex-grow-1 d-lg-block align-items-center justify-content-between">
			<div>
				<p class="my-0 fw-bold">{{ title }}</p>
				<strong class="d-flex align-items-baseline lh-sm">
					<span class="fs-1">{{ value }}</span>
					<span class="ms-1 unit">{{ unit }}</span>
				</strong>
			</div>
			<small
				class="d-block mt-0 ms-3 ms-lg-0 text-end text-lg-start flex-grow-1 flex-shrink-1"
			>
				{{ sub1 }} <br />
				{{ sub2 }}
			</small>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/coinjar";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/receivepayment";

export default {
	name: "SavingsTile",
	props: {
		title: String,
		icon: String,
		value: [String, Number],
		unit: String,
		sub1: String,
		sub2: String,
	},
};
</script>
<style scoped>
.tile-icon {
	width: 40px;
	flex: 0 0 auto;
}
@media (--lg-and-up) {
	.tile-icon {
		width: 70px;
	}
}
.unit {
	font-size: var(--bs-body-font-size);
}
</style>
