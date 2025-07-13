<template>
	<div>
		<label :for="id" class="form-control cursor-pointer">
			<div class="hstack gap-3">
				{{ $t("config.general.selectFile") }}
				<div class="vr"></div>
				<span class="text-truncate">{{ computedFileName }}</span>
			</div>
		</label>

		<input
			:id="id"
			type="file"
			class="d-none"
			:accept="accepted.join(', ')"
			:required="required"
			@change="onFileChange"
		/>

		<div>
			<span v-if="invalidFileSelected" class="text-danger">
				{{ $t("config.general.invalidFileSelected") }}
			</span>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	name: "PropertyFileField",
	props: {
		id: String,
		accepted: { type: Array as PropType<string[]>, default: () => [] },
		required: Boolean,
	},
	emits: ["fileChanged"],
	data() {
		return {
			file: null as File | null,
			invalidFileSelected: false,
		};
	},
	computed: {
		computedFileName() {
			return this.file ? this.file.name : this.$t("config.general.noFileSelected");
		},
	},
	methods: {
		reset() {
			this.file = null;
			this.invalidFileSelected = false;
		},
		onFileChange(event: Event) {
			const file = (event.target as HTMLInputElement).files?.item(0);
			if (file) {
				if (this.accepted.some((a) => file.name.endsWith(a))) {
					this.file = file;
					this.invalidFileSelected = false;
					this.$emit("fileChanged", file);
				} else {
					this.invalidFileSelected = true;
				}
			}
		},
	},
});
</script>
