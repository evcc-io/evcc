export default {
  methods: {
    // collect all target component properties from current instance
    collectProps: function (component, state) {
      let data = {};
      for (var p in component.props) {
        // check in optional state
        if (state && p in state) {
          data[p] = state[p];
        }
        // check in current instance
        if (p in this) {
          data[p] = this[p];
        }
      }
      return data;
    },
  },
};
