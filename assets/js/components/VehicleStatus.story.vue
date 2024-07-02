<script setup>
import VehicleStatus from "./VehicleStatus.vue";
</script>

<template>
	<Story title="VehicleStatus" :layout="{ type: 'grid', iframe: false, width: 420 }">
		<Variant title="status: disconnected">
			<VehicleStatus />
		</Variant>
		<Variant title="status: connected">
			<VehicleStatus connected />
		</Variant>
		<Variant title="status: enabled">
			<VehicleStatus connected enabled />
		</Variant>
		<Variant title="status: charging">
			<VehicleStatus connected charging />
		</Variant>
		<Variant title="solar: pv enable">
			<VehicleStatus connected pvAction="enable" :pvRemainingInterpolated="90" />
		</Variant>
		<Variant title="solar: charging">
			<VehicleStatus connected enabled charging :sessionSolarPercentage="94" />
		</Variant>
		<Variant title="solar: pv disable">
			<VehicleStatus connected charging pvAction="disable" :pvRemainingInterpolated="90" />
		</Variant>
		<Variant title="solar: pv reduce phases">
			<VehicleStatus
				connected
				charging
				phaseAction="scale1p"
				:phaseRemainingInterpolated="181"
			/>
		</Variant>
		<Variant title="solar: pv increase phases">
			<VehicleStatus
				connected
				charging
				phaseAction="scale3p"
				:phaseRemainingInterpolated="44"
			/>
		</Variant>
		<Variant title="min soc">
			<VehicleStatus connected charging :minSoc="20" :vehicleSoc="10" />
		</Variant>
		<Variant title="plan: start soon">
			<VehicleStatus
				connected
				effectivePlanTime="2020-03-16T06:00:00Z"
				planProjectedStart="2020-03-16T02:00:00Z"
			/>
		</Variant>
		<Variant title="plan: start soon, charging">
			<VehicleStatus
				connected
				charging
				effectivePlanTime="2020-03-16T06:00:00Z"
				planProjectedStart="2020-03-16T02:00:00Z"
			/>
		</Variant>
		<Variant title="plan: active">
			<VehicleStatus
				connected
				charging
				planActive
				effectivePlanTime="2020-03-16T06:00:00Z"
				planProjectedStart="2020-03-16T02:00:00Z"
				planProjectedEnd="2020-03-16T05:12:00Z"
			/>
		</Variant>
		<Variant title="plan: active, not reachable">
			<VehicleStatus
				connected
				charging
				planActive
				effectivePlanTime="2020-03-16T06:00:00Z"
				planProjectedStart="2020-03-16T02:00:00Z"
				planProjectedEnd="2020-03-16T07:12:00Z"
				planTimeUnreachable
			/>
		</Variant>
		<Variant title="vehicle: climating">
			<VehicleStatus connected enabled vehicleClimaterActive />
		</Variant>
		<Variant title="vehicle: limit">
			<VehicleStatus connected enabled :vehicleSoc="33" :vehicleLimitSoc="70" />
		</Variant>
		<Variant title="vehicle: limit reached">
			<VehicleStatus connected enabled :vehicleSoc="70" :vehicleLimitSoc="70" />
		</Variant>
		<Variant title="vehicle: limit unreachable">
			<VehicleStatus
				connected
				enabled
				:vehicleSoc="40"
				:vehicleLimitSoc="70"
				:effectivePlanSoc="80"
			/>
		</Variant>
		<Variant title="smart cost: clean energy set">
			<VehicleStatus
				connected
				enabled
				charging
				:tariffCo2="600"
				:smartCostLimit="500"
				smartCostType="co2"
			/>
		</Variant>
		<Variant title="smart cost: clean energy next start">
			<VehicleStatus
				connected
				enabled
				charging
				:smartCostLimit="500"
				smartCostType="co2"
				smartCostNextStart="2024-06-26T19:00:00Z"
			/>
		</Variant>
		<Variant title="smart cost: clean energy active">
			<VehicleStatus
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
			<VehicleStatus
				connected
				enabled
				charging
				currency="CHF"
				:tariffGrid="0.32"
				:smartCostLimit="0.12"
				smartCostType="price"
			/>
		</Variant>
		<Variant title="smart cost: cheap but not connected">
			<VehicleStatus
				currency="EUR"
				:tariffGrid="0.091"
				:smartCostLimit="0.12"
				smartCostType="price"
			/>
		</Variant>
		<Variant title="smart cost: cheap energy next start">
			<VehicleStatus
				connected
				enabled
				charging
				currency="CHF"
				:smartCostLimit="0.12"
				smartCostType="price"
				smartCostNextStart="2024-06-27T19:00:00Z"
			/>
		</Variant>
		<Variant title="smart cost: cheap energy active">
			<VehicleStatus
				connected
				enabled
				charging
				currency="CHF"
				:tariffGrid="0.11"
				:smartCostLimit="0.12"
				smartCostType="price"
				smartCostActive
			/>
		</Variant>
		<Variant title="combination: minsoc, cheap">
			<VehicleStatus
				connected
				enabled
				charging
				currency="EUR"
				:smartCostLimit="0.15"
				smartCostType="price"
				:minSoc="20"
				:vehicleSoc="10"
			/>
		</Variant>
		<Variant title="combination: pv disable, plan">
			<VehicleStatus
				connected
				charging
				pvAction="disable"
				:pvRemainingInterpolated="181"
				effectivePlanTime="2020-03-26T08:12:00Z"
				planProjectedStart="2020-03-26T04:44:00Z"
			/>
		</Variant>
		<Variant title="combination: vehicle limit, plan">
			<VehicleStatus
				connected
				charging
				:sessionSolarPercentage="94"
				:vehicleLimitSoc="80"
				:effectiveLimitSoc="90"
				effectivePlanTime="2020-03-16T06:00:00Z"
				planProjectedStart="2020-03-16T02:00:00Z"
			/>
		</Variant>
		<Variant title="combination: maximal">
			<VehicleStatus
				connected
				charging
				:sessionSolarPercentage="94"
				:minSoc="20"
				:vehicleSoc="10"
				:tariffGrid="0.33"
				:smartCostLimit="0.2"
				smartCostType="price"
				smartCostNextStart="2020-03-16T17:42:00Z"
				effectivePlanTime="2020-03-16T06:00:00Z"
				planProjectedStart="2020-03-16T02:00:00Z"
			/>
		</Variant>
	</Story>
</template>
