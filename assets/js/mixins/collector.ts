import { defineComponent } from "vue";
import type { UiState } from "@/store";

export default defineComponent({
  methods: {
    // collect all target component properties from current instance
    collectProps(component: any, state?: UiState) {
      const data: Record<string, any> = {};
      for (const prop in component.props) {
        const p = prop as keyof UiState;
        // check in optional state
        if (state && p in state) {
          data[p] = state[p];
        }
        // check in current instance
        if (p in this) {
          data[p] = (this as Partial<UiState>)[p];
        }
      }
      return data;
    },
  },
});
