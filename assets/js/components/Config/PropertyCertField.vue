<template>
	<div>
		<textarea :id="id" v-model="value" class="form-control" rows="3" />
		<div class="d-flex justify-content-end">
			<button
				type="button"
				class="btn btn-link btn-sm text-muted pe-0"
				@click="openFilePicker"
			>
				{{ $t("config.general.readFromFile") }}
			</button>
			<input
				type="file"
				ref="fileInput"
				class="d-none"
				@change="readFile"
				accept=".crt,.pem,.cer,.csr,.key"
			/>
		</div>
	</div>
</template>

<script>
export default {
	name: "PropertyCertField",
	props: {
		id: String,
		modelValue: String,
	},
	emits: ["update:modelValue"],
	computed: {
		value: {
			get() {
				return this.modelValue;
			},
			set(value) {
				this.$emit("update:modelValue", value);
			},
		},
	},
	methods: {
		openFilePicker() {
			this.$refs.fileInput.click();
		},
		readFile(event) {
			const file = event.target.files[0];
			const reader = new FileReader();
			reader.onload = (e) => {
				this.value = e.target.result;
			};
			reader.readAsText(file);
		},
	},
};
</script>
