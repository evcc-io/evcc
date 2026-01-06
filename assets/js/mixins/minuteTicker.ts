import { defineComponent } from "vue";
import type { Timeout } from "@/types/evcc";

export default defineComponent({
  data() {
    return {
      currentTime: new Date(),
      minuteTickerInterval: null as Timeout,
    };
  },
  mounted() {
    this.minuteTickerInterval = setInterval(() => {
      // force time-based computed data to update at least once a minute
      this.currentTime = new Date();
    }, 1000 * 60);
  },
  beforeUnmount() {
    if (this.minuteTickerInterval) {
      clearInterval(this.minuteTickerInterval);
    }
  },
});
