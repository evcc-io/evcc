import PlanExplanation from "./PlanExplanation.vue";
import api from "@/api";
import { onMounted, ref } from "vue";
import type { Meta, StoryFn } from "@storybook/vue3";

// votes go nowhere in storybook, so Yes, No and Send can be clicked through
api.post = (() => Promise.resolve({ data: {} })) as unknown as typeof api.post;

export default {
  title: "Optimize/PlanExplanation",
  component: PlanExplanation,
  parameters: {
    layout: "padded",
  },
} as Meta<typeof PlanExplanation>;

const explanation = `There is useful sunshine each day, while prices are lower around midday and rise in the evening. The plan avoids buying from the grid, fills storage from solar and uses it through the evenings.

- **Carport (blue e-Golf)**: Charges from solar today from 12:30 to 15:00, reaching 100%, and stays full.
- **battery**: Charges from solar today and tomorrow, then supplies evening use and ends at 61%.
- **Sungrow**: Covers evening use, then recharges from solar on Sunday and Monday and ends at 98%.

Extra solar is exported on Monday and Tuesday because the storage is full.`;

const plan = new Date().toISOString();
const explained = (minutes: number) => ({
  explanation,
  created: new Date(Date.now() - minutes * 60 * 1000).toISOString(),
  plan,
});

const Template: StoryFn<typeof PlanExplanation> = (args) => ({
  components: { PlanExplanation },
  setup() {
    return { args };
  },
  template: '<div class="container px-0"><PlanExplanation v-bind="args" /></div>',
});

// without sponsorship only the feedback line is shown
export const Feedback = Template.bind({});
Feedback.args = {};

// nothing explained yet
export const Initial = Template.bind({});
Initial.args = { sponsor: true };

// waiting for the model
export const Explaining = Template.bind({});
Explaining.args = { sponsor: true, explaining: true };

// the current plan is explained, a new explanation needs a new plan
export const Explained = Template.bind({});
Explained.args = { sponsor: true, explained: explained(1), plan };

// a newer plan arrived since
export const EarlierPlan = Template.bind({});
EarlierPlan.args = { sponsor: true, explained: explained(20) };

// updating an explanation for an earlier plan
export const Updating = Template.bind({});
Updating.args = { sponsor: true, explained: explained(20), explaining: true };

// older than an hour, shown like nothing was explained
export const Outdated = Template.bind({});
Outdated.args = { sponsor: true, explained: explained(90) };

// the optimizer service could not ask the model
export const Error = Template.bind({});
Error.args = { sponsor: true, error: "model returned 429: slow down" };

type VoteState = Partial<{ vote: string; voteComment: string; voteSent: boolean }>;

// the vote lives in the component, these stories set it after mounting
const WithVote =
  (state: VoteState): StoryFn<typeof PlanExplanation> =>
  (args) => ({
    components: { PlanExplanation },
    setup() {
      const card = ref();
      onMounted(() => Object.assign(card.value, state));
      return { args, card };
    },
    template: '<div class="container px-0"><PlanExplanation ref="card" v-bind="args" /></div>',
  });

// Yes was clicked
export const VotedYes = WithVote({ vote: "up", voteSent: true }).bind({});
VotedYes.args = { sponsor: true, explained: explained(1), plan };

// No was clicked, it counts already, a comment can follow
export const VotedNo = WithVote({ vote: "down" }).bind({});
VotedNo.args = { sponsor: true, explained: explained(1), plan };

// No with a comment typed, ready to send
export const VotedNoComment = WithVote({
  vote: "down",
  voteComment: "the times are off by an hour",
}).bind({});
VotedNoComment.args = { sponsor: true, explained: explained(1), plan };
