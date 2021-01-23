<template>
	<div class="row">
		<div class="col">
			<!-- <h4 class="my-4">{{ label }} ({{ type }})</h4> -->
			<form>
				<Field v-for="(field, idx) in fields" v-bind="field" :key="idx"></Field>
				<button type="submit" class="btn btn-primary btn-small">Test</button>
			</form>
		</div>
	</div>
</template>

<script>
import Field from "./Field";

export default {
	name: "Element",
	components: { Field },
	props: {
		configclass: String,
		type: String,
		label: String,
		fields: Array,
	},
	data: function () {
		return {};
	},
};
</script>
