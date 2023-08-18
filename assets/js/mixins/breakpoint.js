export default {
  data() {
    return {
      breakpoint: "md",
    };
  },
  methods: {
    updateBreakpoint() {
      const width = window.innerWidth;

      if (width <= 575) {
        this.breakpoint = "xs";
      } else if (width <= 767) {
        this.breakpoint = "sm";
      } else if (width <= 991) {
        this.breakpoint = "md";
      } else if (width <= 1199) {
        this.breakpoint = "lg";
      } else if (width <= 1399) {
        this.breakpoint = "xl";
      } else {
        this.breakpoint = "xxl";
      }
    },
  },
  mounted() {
    this.updateBreakpoint();
    window.addEventListener("resize", this.updateBreakpoint);
  },
  beforeDestroy() {
    window.removeEventListener("resize", this.updateBreakpoint);
  },
};
