<script setup lang="ts">
import Status from "./Status.vue";
import { CURRENCY } from "@/types/evcc";

function getFutureTime(hours: number, minutes: number) {
	const now = new Date();
	now.setHours(now.getHours() + hours);
	now.setMinutes(now.getMinutes() + minutes);
	return now.toISOString();
}

const planProjectedStart = getFutureTime(3, 21);
const effectivePlanTime = getFutureTime(6, 54);
const planProjectedEnd = getFutureTime(5, 43);
</script>

<template>
	<Story title="VehicleStatus" :layout="{ type: 'grid', iframe: false, width: 320 }">
		<Variant title="status: disconnected">
			<Status />
		</Variant>
		<Variant title="status: connected">
			<Status connected />
		</Variant>
		<Variant title="status: enabled">
			<Status connected enabled />
		</Variant>
		<Variant title="status: charging">
			<Status connected charging />
		</Variant>
		<Variant title="solar: pv enable">
			<Status connected pvAction="enable" :pvRemainingInterpolated="90" />
		</Variant>
		<Variant title="solar: charging">
			<Status connected enabled charging :sessionSolarPercentage="94" />
		</Variant>
		<Variant title="solar: pv disable">
			<Status connected charging pvAction="disable" :pvRemainingInterpolated="90" />
		</Variant>
		<Variant title="solar: pv reduce phases">
			<Status connected charging phaseAction="scale1p" :phaseRemainingInterpolated="181" />
		</Variant>
		<Variant title="solar: pv increase phases">
			<Status connected charging phaseAction="scale3p" :phaseRemainingInterpolated="44" />
		</Variant>
		<Variant title="min soc">
			<Status connected charging :minSoc="20" :vehicleSoc="10" />
		</Variant>
		<Variant title="plan: start soon">
			<Status
				connected
				:effectivePlanTime="effectivePlanTime"
				:planProjectedStart="planProjectedStart"
			/>
		</Variant>
		<Variant title="plan: start soon, charging">
			<Status
				connected
				charging
				:effectivePlanTime="effectivePlanTime"
				:planProjectedStart="planProjectedStart"
			/>
		</Variant>
		<Variant title="plan: active">
			<Status
				connected
				charging
				planActive
				:effectivePlanTime="effectivePlanTime"
				:planProjectedStart="planProjectedStart"
				:planProjectedEnd="planProjectedEnd"
			/>
		</Variant>
		<Variant title="plan: active, not reachable">
			<Status
				connected
				charging
				planActive
				:effectivePlanTime="effectivePlanTime"
				:planProjectedStart="planProjectedStart"
				:planProjectedEnd="planProjectedEnd"
				:planOverrun="1829"
				planTimeUnreachable
			/>
		</Variant>
		<Variant title="vehicle: climating">
			<Status connected enabled vehicleClimaterActive />
		</Variant>
		<Variant title="vehicle: limit">
			<Status connected enabled :vehicleSoc="33" :vehicleLimitSoc="70" />
		</Variant>
		<Variant title="vehicle: limit reached">
			<Status connected enabled :vehicleSoc="70" :vehicleLimitSoc="70" />
		</Variant>
		<Variant title="vehicle: limit unreachable">
			<Status
				connected
				enabled
				:vehicleSoc="40"
				:vehicleLimitSoc="70"
				:effectivePlanSoc="80"
			/>
		</Variant>
		<Variant title="smart cost: clean energy set">
			<Status
				connected
				enabled
				charging
				:tariffCo2="600"
				:smartCostLimit="500"
				smartCostType="co2"
			/>
		</Variant>
		<Variant title="smart cost: clean energy next start">
			<Status
				connected
				enabled
				:smartCostLimit="500"
				smartCostType="co2"
				:smartCostNextStart="planProjectedStart"
			/>
		</Variant>
		<Variant title="smart cost: clean energy active">
			<Status
				connected
				enabled
				charging
				:tariffCo2="400"
				:smartCostLimit="500"
				smartCostType="co2"
				smartCostActive
			/>
		</Variant>
		<Variant title="smart cost: cheap energy set">
			<Status
				connected
				enabled
				charging
				:currency="CURRENCY.EUR"
				:tariffGrid="0.32"
				:smartCostLimit="0.12"
				smartCostType="price"
			/>
		</Variant>
		<Variant title="smart cost: cheap but not connected">
			<Status
				:currency="CURRENCY.EUR"
				:tariffGrid="0.091"
				:smartCostLimit="0.12"
				smartCostType="price"
			/>
		</Variant>
		<Variant title="smart cost: cheap energy next start">
			<Status
				connected
				enabled
				charging
				:currency="CURRENCY.EUR"
				:smartCostLimit="0.12"
				smartCostType="price"
				:smartCostNextStart="planProjectedStart"
			/>
		</Variant>
		<Variant title="smart cost: cheap energy active">
			<Status
				connected
				enabled
				charging
				:currency="CURRENCY.EUR"
				:tariffGrid="0.11"
				:smartCostLimit="0.12"
				smartCostType="price"
				smartCostActive
			/>
		</Variant>
		<Variant title="combination: minsoc, cheap">
			<Status
				connected
				enabled
				charging
				:currency="CURRENCY.EUR"
				:smartCostLimit="0.15"
				smartCostType="price"
				:minSoc="20"
				:vehicleSoc="10"
			/>
		</Variant>
		<Variant title="combination: pv disable, plan">
			<Status
				connected
				charging
				pvAction="disable"
				:pvRemainingInterpolated="181"
				:effectivePlanTime="effectivePlanTime"
				:planProjectedStart="planProjectedStart"
			/>
		</Variant>
		<Variant title="combination: vehicle limit, plan">
			<Status
				connected
				charging
				:sessionSolarPercentage="94"
				:vehicleLimitSoc="80"
				:effectiveLimitSoc="90"
				:effectivePlanTime="effectivePlanTime"
				:planProjectedStart="planProjectedStart"
			/>
		</Variant>
		<Variant title="combination: maximal">
			<Status
				connected
				enabled
				:sessionSolarPercentage="94"
				:minSoc="20"
				:vehicleSoc="10"
				:tariffGrid="0.33"
				:smartCostLimit="0.2"
				smartCostType="price"
				:smartCostNextStart="planProjectedStart"
				:effectivePlanTime="effectivePlanTime"
				:planProjectedStart="planProjectedStart"
				vehicleClimaterActive
			/>
		</Variant>
		<Variant title="welcome charge">
			<Status connected charging vehicleWelcomeActive />
		</Variant>
	</Story>
</template>
