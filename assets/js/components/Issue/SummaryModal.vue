<template>
	<GenericModal
		id="issueSummaryModal"
		data-testid="issue-summary-modal"
		:title="$t('issue.summary.title')"
		:size="isTwoStepMode ? 'lg' : 'md'"
		:autofocus="false"
	>
		<!-- Instructions (only shown in two-step mode) -->
		<div v-if="isTwoStepMode" class="alert alert-secondary mb-4">
			<strong>{{ $t("issue.summary.instructions") }}</strong>
		</div>

		<!-- Step 1: Create Issue -->
		<div :class="{ 'mb-4': isTwoStepMode }">
			<h6 v-if="isTwoStepMode" class="mb-2">
				{{ $tt("issue.summary.stepOne") }}
			</h6>
			<p class="text-muted small mb-3">
				{{
					isTwoStepMode
						? $t("issue.summary.step1Description")
						: $t("issue.summary.singleStepDescription")
				}}
			</p>
			<div class="d-flex justify-content-start">
				<a
					:href="githubUrl"
					target="_blank"
					class="btn"
					:class="buttonClass"
					@click="clearSessionStorage"
				>
					{{ $tt("issue.summary.confirmationButton") }}
				</a>
			</div>
		</div>

		<hr v-if="isTwoStepMode" class="my-4" />

		<!-- Step 2: Copy Additional Information (only shown in two-step mode) -->
		<div v-if="isTwoStepMode" class="mb-4">
			<h6 class="mb-2">{{ $t("issue.summary.stepTwo") }}</h6>
			<p class="text-muted small mb-3">{{ $t("issue.summary.step2Description") }}</p>
			<div class="d-flex justify-content-start mb-4">
				<CopyButton :content="additional" :targetElement="$refs['summaryTextarea']">
					<template #default="{ copy, copied, copying }">
						<button
							type="button"
							class="btn"
							:class="
								helpType === 'discussion'
									? 'btn-outline-success'
									: 'btn-outline-danger'
							"
							:disabled="copying"
							@click="copy"
						>
							{{
								copied ? $t("issue.summary.copied") : $t("issue.summary.copyButton")
							}}
						</button>
					</template>
				</CopyButton>
			</div>
			<div class="mb-2">
				<textarea
					ref="summaryTextarea"
					:value="additional"
					class="form-control font-monospace border-secondary textarea--tiny"
					:rows="additionalRows"
					readonly
					style="white-space: pre; overflow-wrap: normal"
					data-testid="issue-summary-modal-textarea"
				></textarea>
			</div>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import GenericModal from "@/components/Helper/GenericModal.vue";
import CopyButton from "@/components/Helper/CopyButton.vue";
import { generateGitHubContent, generateGitHubUrl } from "./template";
import type { GitHubContent, HelpType, IssueData, Sections } from "./types";

export default defineComponent({
	name: "SummaryModal",
	components: {
		GenericModal,
		CopyButton,
	},
	props: {
		helpType: { type: String as PropType<HelpType>, required: true },
		buttonClass: { type: String, required: true },
		issueData: { type: Object as PropType<IssueData>, required: true },
		sections: { type: Object as PropType<Sections>, required: true },
	},
	emits: ["submitted"],
	computed: {
		content(): GitHubContent {
			return generateGitHubContent(this.issueData, this.sections);
		},
		githubUrl(): string {
			return generateGitHubUrl(this.helpType, this.issueData.title, this.content.body);
		},
		additional(): string {
			return this.content.additional || "";
		},
		isTwoStepMode(): boolean {
			return !!this.content.additional;
		},
		additionalRows(): number {
			const lines = this.additional.split("\n").length;
			return Math.max(26, lines);
		},
	},
	methods: {
		clearSessionStorage() {
			this.$emit("submitted");
		},
		$tt(key: string): string {
			return this.$t(`${key}${this.helpType === "discussion" ? "Discussion" : "Issue"}`);
		},
	},
});
</script>
