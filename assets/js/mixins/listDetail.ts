import { defineComponent } from "vue";
import breakpoint, { MAX_WIDTH, isMobileWidth } from "./breakpoint";
import { hapticFeedback } from "@/utils/haptic";

// hash-driven list>detail pages (see Config.vue): consumers override `activeSlug`,
// list rows carry data-slug; pairs with TopHeader subpageTitle + fade-swap classes
export default defineComponent({
  mixins: [breakpoint],
  data() {
    return {
      listScroll: 0,
    };
  },
  computed: {
    mobile(): boolean {
      return isMobileWidth(MAX_WIDTH[this.breakpoint]);
    },
    activeSlug(): string | undefined {
      return undefined; // consumers override
    },
  },
  watch: {
    "$route.hash"(_: string, oldHash: string) {
      // new content dictates the scroll position
      if (this.mobile && this.activeSlug) {
        this.listScroll = window.scrollY;
        window.scrollTo(0, 0);
      }
      // back to the list: restore its scroll and focus the originating row
      if (this.mobile && !this.activeSlug && oldHash) {
        this.$nextTick(() => {
          window.scrollTo(0, this.listScroll);
          document
            .querySelector<HTMLElement>(`[data-slug="${oldHash.slice(1)}"]`)
            ?.focus({ preventScroll: true });
        });
      }
    },
  },
  methods: {
    goBack() {
      hapticFeedback("light");
      const back = this.$router.options.history.state["back"] as string | null;
      if (back?.startsWith(this.$route.path) && !back.includes("#")) {
        this.$router.back();
      } else {
        // direct deep link, browser back would leave the app
        this.$router.replace({
          path: this.$route.path,
          query: this.$route.query,
          hash: "",
        });
      }
    },
  },
});
