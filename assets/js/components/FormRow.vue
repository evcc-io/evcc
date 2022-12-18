<template>
	<div class="mb-4 row">
		<label :for="id" class="col-sm-4 col-form-label pt-0 pt-sm-1">
			{{ label }}
		</label>
		<div class="col-sm-8 pe-0">
			<slot />
		</div>
	</div>
</template>

<script>
export default {
	name: "FormRow",
	props: {
		id: String,
		label: String,
	},
};
</script>
