<template>
	<div class="btn-group btn-group-toggle bg-white shadow-none">
		<label
			class="btn btn-outline-primary value disabled caption font-weight-bold"
			v-if="caption"
		>
			Modus
		</label>
		<label class="btn btn-outline-primary" :class="{ active: mode == 'off', first: !caption }">
			<input type="radio" value="off" v-on:click="setTargetMode('off')" />Stop
		</label>
		<label class="btn btn-outline-primary" :class="{ active: mode == 'now' }">
			<input type="radio" value="now" v-on:click="setTargetMode('now')" />Sofort
		</label>
		<label
			class="btn btn-outline-primary"
			:class="{ active: mode == 'minpv' }"
			v-if="pvConfigured"
		>
			<input type="radio" value="minpv" v-on:click="setTargetMode('minpv')" />
			<span class="d-inline d-lg-none">Min</span>
			<span class="d-none d-lg-inline">Min + PV</span>
		</label>
		<label
			class="btn btn-outline-primary"
			:class="{ active: mode == 'pv' }"
			v-if="pvConfigured"
		>
			<input type="radio" value="pv" v-on:click="setTargetMode('pv')" />
			<span class="d-inline d-md-none">PV</span>
			<span class="d-none d-md-inline">Nur PV</span>
		</label>
	</div>
</template>

<script>
export default {
	name: "Mode",
	props: {
		mode: String,
		pvConfigured: Boolean,
		caption: Boolean,
	},
	methods: {
		setTargetMode: function (mode) {
			this.$emit("updated", mode);
		},
	},
};
</script>
