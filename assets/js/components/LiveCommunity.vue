<template>
	<div class="d-block d-lg-flex mb-2 justify-content-between">
		<SavingsTile
			class="text-accent2"
			icon="car"
			:title="$t('footer.community.power')"
			:value="chargePower"
			:valueFmt="numberAnimationFmt"
			:animationDuration="animationDuration"
			unit="kW"
			:sub1="$t('footer.community.powerSub1', { totalClients, activeClients })"
			:sub2="$t('footer.community.powerSub2')"
		/>

		<SavingsTile
			class="text-accent1"
			icon="sun"
			:title="$t('footer.community.greenShare')"
			:value="greenShare"
			:valueFmt="numberAnimationFmt"
			:animationDuration="animationDuration"
			unit="%"
			:sub1="$t('footer.community.greenShareSub1')"
			:sub2="$t('footer.community.greenShareSub2')"
		/>

		<SavingsTile
			class="text-accent3"
			icon="eco"
			:title="$t('footer.community.greenEnergy')"
			:value="greenEnergyMWh"
			:valueFmt="numberAnimationFmt"
			:animationDuration="animationDuration"
			unit="MWh"
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
			animationDuration: 0.5,
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
			const { chargePower = 0 } = this.result;
			return chargePower / 1e3;
		},
		greenShare() {
			const { chargePower, greenPower } = this.result;
			if (!chargePower) {
				return 0;
			}
			return (100 / chargePower) * greenPower;
		},
		greenEnergyMWh() {
			const { greenEnergy = 0 } = this.result;
			return greenEnergy / 1e3;
		},
	},
	async mounted() {
		this.refresh = setInterval(this.update, UPDATE_INTERVAL_SECONDS * 1e3);
		await this.update();
		this.$nextTick(() => {
			this.animationDuration = UPDATE_INTERVAL_SECONDS;
		});
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
		numberAnimationFmt(number) {
			return this.fmtNumber(number, 1);
		},
	},
};
</script>
