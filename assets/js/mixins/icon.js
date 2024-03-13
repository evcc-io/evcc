export default {
  props: {
    size: {
      type: String,
      validator: function (value) {
        return ["s", "m", "l", "xl"].includes(value);
      },
    },
  },
  computed: {
    svgStyle() {
      const sizes = {
        s: "24px",
        m: "32px",
        l: "48px",
        xl: "64px",
      };
      const size = sizes[this.size] || sizes.s;
      return { display: "block", width: size, height: size };
    },
  },
};
