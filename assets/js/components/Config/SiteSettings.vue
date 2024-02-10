<template>
	<div class="group p-4 pb-2">
		<form class="container mx-0 px-0">
			<FormRow id="siteTitle" label="Site title">
				<input
					id="siteTitle"
					v-model="modifiedFormData.title"
					class="form-control"
					placeholder="Home"
				/>
			</FormRow>
			<FormRow id="adminPassword" label="Admin password" class="wip">
				<input
					id="adminPassword"
					type="password"
					value="- not set -"
					class="form-control"
					disabled
				/>
			</FormRow>
			<FormRow id="sponsorToken" label="Sponsor token" class="wip">
				<textarea id="sponsorToken" class="form-control" rows="3" disabled value="tba" />
			</FormRow>
			<div class="my-4 d-flex justify-content-end">
				<button type="reset" class="btn btn-link text-muted" @click.prevent="reset">
					{{ $t("config.site.cancel") }}
				</button>
				<button
					type="submit"
					class="btn btn-primary"
					:disabled="!saveNeeded || saving"
					@click.prevent="save"
				>
					<span
						v-if="saving"
						class="spinner-border spinner-border-sm"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("config.site.save") }}
				</button>
			</div>
		</form>
	</div>
</template>

<script>
import api from "../../api";
import FormRow from "../FormRow.vue";

export default {
	name: "SiteSettings",
	components: { FormRow },
	data() {
		return {
			originalFormData: { title: "" },
			modifiedFormData: { title: "" },
			saving: false,
		};
	},
	emits: ["site-changed"],
	async mounted() {
		await this.load();
	},
	computed: {
		saveNeeded() {
			return JSON.stringify(this.modifiedFormData) !== JSON.stringify(this.originalFormData);
		},
	},
	methods: {
		async load() {
			try {
				const { data } = await api.get("/config/site");
				this.originalFormData.title = data.result.title;
			} catch (e) {
				console.error(e);
			}
			this.modifiedFormData = { ...this.originalFormData };
		},
		async save() {
			this.saving = true;
			try {
				await api.put("/config/site", this.modifiedFormData);
				this.$emit("site-changed");
			} catch (e) {
				console.error(e);
			}
			this.saving = false;
			await this.load();
		},
		reset() {
			this.modifiedFormData = { ...this.originalFormData };
		},
	},
};
</script>

<style scoped>
.group {
	border-radius: 1rem;
	box-shadow: 0 0 0 0 var(--evcc-gray-50);
	color: var(--evcc-default-text);
	background: var(--evcc-box);
	padding: 1rem 1rem 0.5rem;
	display: block;
	list-style-type: none;
	min-height: 10rem;
	margin-bottom: 5rem;
	border: 1px solid var(--evcc-gray-50);
	transition: box-shadow var(--evcc-transition-fast) linear;
}

.group:hover {
	border-color: var(--evcc-gray);
}

.group:focus-within {
	box-shadow: 0 0 1rem 0 var(--evcc-gray-50);
}

.wip {
	opacity: 0.2;
}
</style>
