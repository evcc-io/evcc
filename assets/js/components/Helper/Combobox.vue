<template>
	<div class="position-relative" @focusout="onFocusOut">
		<input
			:id="id"
			ref="input"
			:value="modelValue"
			type="text"
			role="combobox"
			class="form-select"
			:class="{ 'is-invalid': invalid, 'has-clear-button': showClearButton }"
			autocomplete="off"
			aria-autocomplete="list"
			aria-haspopup="listbox"
			:aria-expanded="expanded"
			:aria-controls="listId"
			:aria-activedescendant="activeId"
			:placeholder="placeholder"
			:required="required"
			:pattern="pattern"
			:title="title"
			:disabled="disabled"
			@focus="show"
			@click="show"
			@input="onInput"
			@keydown="onKeydown"
		/>
		<button
			v-if="showClearButton"
			type="button"
			class="form-control-clear"
			tabindex="-1"
			:aria-label="$t('config.general.clear')"
			:disabled="disabled"
			@click="clear"
		></button>
		<div
			v-show="expanded"
			class="dropdown-menu show w-100 py-0"
			data-bs-popper="static"
			@mousedown.prevent
		>
			<div
				:id="listId"
				ref="list"
				role="listbox"
				class="overflow-y-auto overflow-x-hidden options"
				:aria-label="$t('config.combobox.suggestions')"
			>
				<div
					v-for="group in groups"
					:key="group.key"
					role="group"
					class="group"
					:aria-label="showHeadings ? group.name : undefined"
				>
					<div v-if="showHeadings" class="dropdown-header" aria-hidden="true">
						{{ group.name }}
					</div>
					<div
						v-for="option in group.options"
						:id="optionId(option)"
						:key="option.value"
						role="option"
						class="dropdown-item option py-2"
						:class="{ 'option--active': option.value === activeValue }"
						:aria-selected="option.value === activeValue"
						@click="select(option)"
					>
						<div class="fw-bold text-truncate option-value">{{ option.value }}</div>
						<div
							v-if="option.label"
							class="small text-muted text-truncate option-label"
						>
							{{ option.label }}
						</div>
						<Badge v-if="option.used" variant="muted" class="option-badge">
							{{ $t("config.combobox.used") }}
						</Badge>
						<div v-if="option.hint" class="small text-muted text-truncate option-hint">
							{{ option.hint }}
						</div>
					</div>
				</div>
			</div>
			<div
				v-if="loading"
				class="dropdown-item-text small text-muted d-flex align-items-center gap-2 py-2 searching"
			>
				<span class="spinner-border spinner-border-sm" aria-hidden="true"></span>
				{{ $t("config.combobox.searching") }}
			</div>
		</div>
		<div role="status" class="visually-hidden">{{ status }}</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Badge from "./Badge.vue";
import type { ServiceOption } from "@/types/evcc";

interface Group {
	key: string;
	name: string;
	options: ServiceOption[];
}

// unused entries first, otherwise keep order
const usedLast = (options: ServiceOption[]) => [
	...options.filter((o) => !o.used),
	...options.filter((o) => o.used),
];

export default defineComponent({
	name: "Combobox",
	components: { Badge },
	props: {
		id: { type: String, required: true },
		modelValue: { type: String, default: "" },
		options: { type: Array as PropType<ServiceOption[]>, default: () => [] },
		placeholder: String,
		required: Boolean,
		pattern: String,
		title: String,
		disabled: Boolean,
		invalid: Boolean,
		loading: Boolean,
	},
	emits: ["update:modelValue"],
	data() {
		return {
			open: false,
			// only narrow the list once the user types, a selected value shows all options
			filtering: false,
			activeValue: null as string | null,
		};
	},
	computed: {
		listId() {
			return `${this.id}-listbox`;
		},
		showClearButton() {
			return !!this.modelValue;
		},
		filteredOptions(): ServiceOption[] {
			const query = this.modelValue?.toLowerCase();
			if (!this.filtering || !query) return this.options;
			return this.options.filter(({ value, label, hint }) =>
				[value, label, hint].some((text) => text?.toLowerCase().includes(query))
			);
		},
		groups(): Group[] {
			const matching = this.filteredOptions.filter((o) => o.match);
			const other = this.filteredOptions.filter((o) => !o.match);
			return [
				{ key: "matching", name: this.$t("config.combobox.matching"), options: matching },
				{ key: "other", name: this.$t("config.combobox.other"), options: other },
			]
				.filter((g) => g.options.length > 0)
				.map((g) => ({
					...g,
					name: `${g.name} (${g.options.length})`,
					options: usedLast(g.options),
				}));
		},
		showHeadings() {
			return this.groups.length > 1;
		},
		visibleOptions(): ServiceOption[] {
			return this.groups.flatMap((g) => g.options);
		},
		expanded(): boolean {
			// boolean props passed as undefined stay undefined
			return this.open && (this.visibleOptions.length > 0 || !!this.loading);
		},
		activeId() {
			const active = this.visibleOptions.find((o) => o.value === this.activeValue);
			return this.expanded && active ? this.optionId(active) : undefined;
		},
		status() {
			if (!this.expanded) return "";
			const count = this.$t("config.combobox.count", { count: this.visibleOptions.length });
			return this.loading ? `${count} ${this.$t("config.combobox.searching")}` : count;
		},
	},
	methods: {
		optionId(option: ServiceOption) {
			return `${this.id}-option-${this.options.indexOf(option)}`;
		},
		show() {
			if (this.disabled) return;
			this.open = true;
		},
		hide() {
			this.open = false;
			this.filtering = false;
			this.activeValue = null;
		},
		select(option: ServiceOption) {
			this.$emit("update:modelValue", option.value);
			this.hide();
		},
		clear() {
			this.$emit("update:modelValue", "");
			(this.$refs["input"] as HTMLInputElement).focus();
		},
		onInput(e: Event) {
			this.filtering = true;
			this.activeValue = null;
			this.open = true;
			this.$emit("update:modelValue", (e.target as HTMLInputElement).value);
		},
		onFocusOut(e: FocusEvent) {
			if (!(this.$el as HTMLElement).contains(e.relatedTarget as Node | null)) {
				this.hide();
			}
		},
		activate(index: number) {
			const options = this.visibleOptions;
			if (!options.length) return;
			const option = options[(index + options.length) % options.length];
			this.activeValue = option?.value ?? null;
			this.$nextTick(() => {
				(this.$refs["list"] as HTMLElement)
					.querySelector('[aria-selected="true"]')
					?.scrollIntoView({ block: "nearest" });
			});
		},
		onKeydown(e: KeyboardEvent) {
			const index = this.visibleOptions.findIndex((o) => o.value === this.activeValue);
			const active = this.expanded ? this.visibleOptions[index] : undefined;

			switch (e.key) {
				case "ArrowDown":
				case "ArrowUp": {
					e.preventDefault();
					const wasOpen = this.expanded;
					this.open = true;
					const down = e.key === "ArrowDown";
					if (!wasOpen || index < 0) {
						this.activate(down ? 0 : -1);
					} else {
						this.activate(index + (down ? 1 : -1));
					}
					break;
				}
				case "Enter":
					// without active option the form submits
					if (!active) return;
					e.preventDefault();
					this.select(active);
					break;
				case "Escape":
					if (!this.expanded) return;
					// keep the surrounding modal open
					e.stopPropagation();
					this.hide();
					break;
			}
		},
	},
});
</script>

<style scoped>
.options {
	max-height: 20rem;
}
/* address and name left, badge and brand right, brand is cut first */
.option {
	display: grid;
	grid-template-columns: minmax(0, 1fr) fit-content(50%);
	column-gap: 1rem;
	align-items: center;
}
.option-value {
	grid-area: 1 / 1;
}
.option-label {
	grid-area: 2 / 1;
}
.option-badge {
	grid-area: 1 / 2;
	justify-self: end;
}
.option-hint {
	grid-area: 2 / 2;
	text-align: end;
}
.searching {
	border-top: var(--bs-dropdown-border-width) solid var(--bs-dropdown-divider-bg);
}
.group + .group {
	border-top: var(--bs-dropdown-border-width) solid var(--bs-dropdown-divider-bg);
}
.option {
	min-height: 44px;
	cursor: pointer;
}
.option--active {
	color: var(--bs-dropdown-link-hover-color);
	background-color: var(--bs-dropdown-link-hover-bg);
}
</style>
