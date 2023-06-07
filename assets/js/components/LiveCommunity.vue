<template>
	<div class="d-block d-lg-flex mb-2 justify-content-between">
		<SavingsTile
			class="text-accent2"
			icon="car"
			:title="$t('footer.community.power')"
			:value="chargePower.value"
			:valueFmt="fmtAnimation"
			:unit="chargePower.unit"
			:sub1="$t('footer.community.powerSub1', { totalClients, activeClients })"
			:sub2="$t('footer.community.powerSub2')"
		/>

		<SavingsTile
			class="text-accent1"
			icon="sun"
			:title="$t('footer.community.greenShare')"
			:value="greenShare"
			:valueFmt="fmtAnimation"
			unit="%"
			:sub1="$t('footer.community.greenShareSub1')"
			:sub2="$t('footer.community.greenShareSub2')"
		/>

		<SavingsTile
			class="text-accent3"
			icon="eco"
			:title="$t('footer.community.greenEnergy')"
			:value="greenEnergy.value"
			:valueFmt="fmtAnimation"
			:unit="greenEnergy.unit"
			:sub1="$t('footer.community.greenEnergySub1')"
			:sub2="$t('footer.community.greenEnergySub2')"
		/>
	</div>
</template>

<script>
import SavingsTile from "./SavingsTile.vue";

import formatter from "../mixins/formatter";
import communityApi from "../communityApi";

const UPDATE_INTERVAL_SECONDS = 10;

export default {
	name: "LiveCommunity",
	components: { SavingsTile },
	mixins: [formatter],
	props: {},
	data() {
		return {
			refresh: null,
			result: {},
		};
	},
	computed: {
		totalClients() {
			return this.result.totalClients;
		},
		activeClients() {
			return this.result.activeClients;
		},
		chargePower() {
			let { chargePower = 0 } = this.result;
			if (chargePower < 1e6) return { value: chargePower / 1e3, unit: "kW" };
			if (chargePower < 1e9) return { value: chargePower / 1e6, unit: "MW" };
			if (chargePower < 1e12) return { value: chargePower / 1e9, unit: "GW" };
			return { value: chargePower / 1e12, unit: "TW" };
		},
		greenShare() {
			const { chargePower, greenPower } = this.result;
			if (!chargePower) {
				return 0;
			}
			return (100 / chargePower) * greenPower;
		},
		greenEnergy() {
			const { greenEnergy = 0 } = this.result;
			if (greenEnergy < 1e3) return { value: greenEnergy, unit: "kWh" };
			if (greenEnergy < 1e6) return { value: greenEnergy / 1e3, unit: "MWh" };
			if (greenEnergy < 1e9) return { value: greenEnergy / 1e6, unit: "GWh" };
			return { value: greenEnergy / 1e9, unit: "TWh" };
		},
	},
	async mounted() {
		this.refresh = setInterval(this.update, UPDATE_INTERVAL_SECONDS * 1e3);
		await this.update();
	},
	unmounted() {
		clearInterval(this.refresh);
	},
	methods: {
		async update() {
			try {
				const response = await communityApi.get("total");
				this.result = response.data || {};
			} catch (err) {
				console.error(err);
			}
		},
		fmtAnimation(number) {
			let decimals = 0;
			if (number < 100) decimals = 1;
			if (number < 10) decimals = 2;
			return this.fmtNumber(number, decimals);
		},
	},
};
</script>
