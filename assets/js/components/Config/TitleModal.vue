<template>
	<JsonModal
		id="titleModal"
		:title="$t('config.title.title')"
		endpoint="/config/site"
		data-testid="title-modal"
		disable-remove
		save-method="put"
		:transform-values="transformValues"
		@changed="$emit('changed')"
	>
		<template v-slot:default="{ values }">
			<FormRow
				id="siteTitle"
				:label="$t('config.title.label')"
				:help="$t('config.title.description')"
			>
				<input id="siteTitle" v-model="values.title" class="form-control" />
			</FormRow>
		</template>
	</JsonModal>
</template>

<script>
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";

export default {
	name: "TitleModal",
	components: { FormRow, JsonModal },
	emits: ["changed"],
	methods: {
		transformValues(values) {
			// only send the title, no meter refs
			return { title: values.title };
		},
	},
};
</script>
