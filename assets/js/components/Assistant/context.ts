import { ref } from "vue";

// situational context of the visible page, e.g. shown errors, handed to the model with each question.
// the widget lives at app level, so views publish their context here instead of passing a prop.
const assistantContext = ref("");

export default assistantContext;

export function setAssistantContext(context: string) {
	assistantContext.value = context;
}
