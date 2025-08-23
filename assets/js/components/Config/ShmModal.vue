<template>
	<JsonModal
		id="shmModal"
		:title="$t('config.hems.shm.title')"
		:description="$t('config.hems.description')"
		docs="/docs/reference/configuration/hems"
		endpoint="/config/shm"
		state-key="shm"
		:disableRemove="true"
		data-testid="shm-modal"
		@changed="$emit('changed')"
	>
		<template #default="{ values }">
			<div class="mb-4">
				<label for="hemsSystemSelect" class="form-label">
					{{ $t("config.hems.shm.template") }}
				</label>
				<select
					name="shmSystem"
					id="shmSystemSelect"
					class="form-select w-100"
					:value="values.type"
					@input="(e) => (values.type = (e.target as HTMLInputElement).value)"
				>
					<option value="">{{ $t("config.hems.shm.noSystem") }}</option>
					<option value="shm">{{ $t("config.hems.shm.title") }}</option>
				</select>
			</div>
			<div v-if="values.type">
				<FormRow id="shmControl" :label="$t('config.hems.shm.labelControl')" optional>
					<div class="d-flex">
						<input
							id="shmControl"
							v-model="values.allowcontrol"
							class="form-check-input"
							type="checkbox"
						/>
						<label class="form-check-label ms-2" for="shmControl">
							{{ $t("config.hems.shm.labelAllowControl") }}
						</label>
					</div>
				</FormRow>
				<PropertyCollapsible>
					<template #advanced>
						<FormRow
							id="shmVendorid"
							:label="$t('config.hems.shm.labelVendorid')"
							:help="$t('config.hems.shm.descriptionVendorid')"
							example="12312312"
							optional
						>
							<input
								id="shmVendorid"
								v-model="values.vendorId"
								class="form-control"
								minlength="8"
								maxlength="8"
							/>
						</FormRow>
						<FormRow
							id="shmDeviceid"
							:label="$t('config.hems.shm.labelDeviceid')"
							:help="$t('config.hems.shm.descriptionDeviceid')"
							example="BBBBBBBBBBBB"
							optional
						>
							<input
								id="shmDeviceid"
								v-model="values.deviceId"
								class="form-control"
								minlength="10"
								maxlength="10"
							/> </FormRow
					></template>
				</PropertyCollapsible>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import FormRow from "./FormRow.vue";
import JsonModal from "./JsonModal.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";

export default {
	name: "ShmModal",
	components: { JsonModal, FormRow, PropertyCollapsible },
	emits: ["changed"],
};
</script>
