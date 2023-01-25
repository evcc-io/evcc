<script setup>
import TargetChargePlan from "./TargetChargePlan.vue";

const now = new Date();

function createDate(hoursFromNow) {
	const result = new Date(now.getTime());
	result.setHours(result.getHours() + hoursFromNow);
	return result;
}

function createRate(price, hoursFromNow) {
	const start = new Date(now.getTime());
	start.setHours(start.getHours() + hoursFromNow);
	start.setMinutes(0);
	start.setSeconds(0);
	start.setMilliseconds(0);
	const end = new Date(start.getTime());
	end.setHours(start.getHours() + 1);
	return { start: start.toISOString(), end: end.toISOString(), price };
}

const rates = [
	545, 518, 545, 518, 213, 545, 527, 527, 536, 518, 400, 336, 336, 336, 336, 336, 336, 336, 372,
	400, 555, 555, 545, 555, 564, 545, 555, 545, 536, 545, 527, 536, 518, 545, 509, 336, 336, 336,
].map((price, i) => createRate(price, i));

const duration = 8695;
const plan = [createRate(213, 4), createRate(336, 11), createRate(336, 12)];
</script>

<template>
	<Story title="TargetChargePlan">
		<Variant title="standard">
			<TargetChargePlan
				:rates="rates"
				:plan="plan"
				:duration="duration"
				unit="gCO2eq"
				:target-time="createDate(14)"
			/>
		</Variant>
	</Story>
</template>
