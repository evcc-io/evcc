<template>
	<JsonModal
		id="hemsModal"
		:title="$t('config.hems.title')"
		:description="$t('config.hems.description')"
		docs="/docs/reference/configuration/hems"
		endpoint="/config/hems"
		state-key="hems"
		:disableRemove="true"
		:initalValues="{ Other: {} }"
		data-testid="hems-modal"
		@changed="$emit('changed')"
	>
		<template #default="{ values }">
			<div class="mb-4">
				<label for="hemsSystemSelect" class="form-label">
					{{ $t("config.hems.template") }}
				</label>
				<select
					name="hemsSystem"
					id="hemsSystemSelect"
					class="form-select w-100"
					:value="values.type"
					@input="(e) => (values.type = (e.target as HTMLInputElement).value)"
				>
					<option value="">{{ $t("config.hems.noSystem") }}</option>
					<option value="sma">Sunny Home Manager 2.0</option>
				</select>
			</div>
			<div v-if="values.type">
				<FormRow id="hemsControl" :label="$t('config.hems.labelControl')" optional>
					<div class="d-flex">
						<input
							id="hemsControl"
							v-model="values.Other.allowcontrol"
							class="form-check-input"
							type="checkbox"
						/>
						<label class="form-check-label ms-2" for="hemsControl">
							{{ $t("config.hems.labelAllowControl") }}
						</label>
					</div>
				</FormRow>
				<PropertyCollapsible>
					<template #advanced>
						<FormRow
							id="hemsVendorid"
							:label="$t('config.hems.labelVendorid')"
							:help="$t('config.hems.descriptionVendorid')"
							example="12312312"
							optional
						>
							<input
								id="hemsVendorid"
								v-model="values.Other.vendorId"
								class="form-control"
								minlength="8"
								maxlength="8"
							/>
						</FormRow>
						<FormRow
							id="hemsDeviceid"
							:label="$t('config.hems.labelDeviceid')"
							:help="$t('config.hems.descriptionDeviceid')"
							example="BBBBBBBBBBBB"
							optional
						>
							<input
								id="hemsDeviceid"
								v-model="values.Other.deviceId"
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
	name: "HemsModal",
	components: { JsonModal, FormRow, PropertyCollapsible },
	emits: ["changed"],
};
</script>
