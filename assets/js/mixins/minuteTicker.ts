import { defineComponent } from "vue";
import type { Timeout } from "@/types/evcc";

export default defineComponent({
  data() {
    return {
      everyMinute: new Date(),
      minuteInterval: null as Timeout,
    };
  },
  mounted() {
    this.minuteInterval = setInterval(() => {
      // force time-based computed data to update at least once a minute
      this.everyMinute = new Date();
    }, 1000 * 60);
  },
  beforeUnmount() {
    if (this.minuteInterval) {
      clearInterval(this.minuteInterval);
    }
  },
});
