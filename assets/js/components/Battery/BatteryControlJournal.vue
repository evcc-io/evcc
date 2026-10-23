<template>
	<section
		v-for="(control, name) in visible"
		:key="name"
		class="alert alert-warning mb-4"
		role="alert"
		data-testid="pv-control-conflict"
	>
		<h2 class="h5">
			{{
				$t(
					`battery.control.${control.state === "confirming" ? "confirmingTitle" : control.state === "disabled" ? "disabled" : control.state === "control-conflict" ? "conflict" : "unavailable"}`
				)
			}}
		</h2>
		<p>
			{{
				$t(
					control.state === "confirming"
						? "battery.control.confirming"
						: "battery.control.description",
					{ battery: name }
				)
			}}
		</p>
		<p v-if="control.state === 'control-conflict' || control.state === 'disabled'">
			{{ $t("battery.control.question") }}
		</p>
		<details class="mb-3">
			<summary>{{ $t("battery.control.details") }}</summary>
			<p class="mb-1">
				{{ $t("battery.control.observed") }}: {{ control.actualMode }} /
				{{ control.nativeMode || "—" }}
			</p>
			<p class="mb-1">
				{{ $t("battery.control.expected") }}: {{ control.expectedPower ?? "—" }} W
			</p>
			<p class="mb-1">
				{{ $t("battery.control.observedAt") }}: {{ control.observedAt || "—" }}
			</p>
			<p v-if="control.observationValid === false" class="text-danger">
				{{ $t("battery.control.stale") }}
			</p>
			<p class="mb-1">{{ control.updated }}</p>
			<p class="mb-0">{{ control.reason }}</p>
		</details>
		<template
			v-if="control.revision && ['control-conflict', 'disabled'].includes(control.state)"
		>
			<p>{{ $t("battery.control.restoreNotice") }}</p>
			<div class="d-flex flex-wrap gap-2">
				<button
					v-if="control.state !== 'disabled'"
					class="btn btn-outline-secondary"
					:disabled="busy !== ''"
					data-testid="pv-disable"
					@click="decide(String(name), 'disable', control.revision)"
				>
					{{ $t("battery.control.disable") }}
				</button>
				<button
					class="btn btn-outline-primary"
					:disabled="busy !== ''"
					data-testid="pv-recover"
					@click="decide(String(name), 'recover', control.revision)"
				>
					{{ $t("battery.control.recover") }}
				</button>
			</div>
		</template>
		<p v-if="failure === String(name)" class="text-danger mt-2" role="status">
			{{ $t("battery.control.failed") }}
		</p>
		<p v-if="accepted === String(name)" class="mt-2" role="status">
			{{ $t("battery.control.accepted") }}
		</p>
	</section>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import api from "@/api";

type Control = {
	state: string;
	reason: string;
	actualMode: string;
	nativeMode: string;
	revision?: string;
	updated?: string;
	expectedPower?: number;
	observedAt?: string;
	observationValid?: boolean;
};

export default defineComponent({
	props: { controls: Object as PropType<Record<string, Control>> },
	data() {
		return { busy: "", failure: "", accepted: "" };
	},
	computed: {
		visible(): Record<string, Control> {
			return Object.fromEntries(
				Object.entries(this.controls ?? {}).filter(([, c]) =>
					[
						"confirming",
						"control-conflict",
						"disabled",
						"recovering",
						"safe-control-unavailable",
					].includes(c.state)
				)
			);
		},
	},
	methods: {
		async decide(battery: string, action: string, revision: string) {
			if (this.busy) return;
			this.busy = battery;
			this.failure = "";
			this.accepted = "";
			try {
				await api.post("batterypvcontrol", { battery, action, revision });
				this.accepted = battery;
			} catch {
				this.failure = battery;
			} finally {
				this.busy = "";
			}
		},
	},
});
</script>
