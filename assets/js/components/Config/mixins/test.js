const TEST_UNKNOWN = "unknown";
const TEST_SUCCESS = "success";
const TEST_FAILED = "failed";
const TEST_RUNNING = "running";

export default {
  data() {
    return {
      testState: TEST_UNKNOWN,
      testError: null,
      testResult: null,
      testTimeout: 15000, // 15s
    };
  },
  computed: {
    testRunning() {
      return this.testState === TEST_RUNNING;
    },
    testSuccess() {
      return this.testState === TEST_SUCCESS;
    },
    testFailed() {
      return this.testState === TEST_FAILED;
    },
    testUnknown() {
      return this.testState === TEST_UNKNOWN;
    },
  },
  methods: {
    resetTest() {
      this.testState = TEST_UNKNOWN;
      this.testError = null;
      this.testResult = null;
    },

    async test(testApi) {
      if (!this.$refs.form.reportValidity()) return false;
      this.testState = TEST_RUNNING;
      try {
        const res = await testApi();
        for (const [key, { error }] of Object.entries(res.data)) {
          if (error) {
            this.testState = TEST_FAILED;
            this.testResult = null;
            this.testError = `${key}: ${error}`;
            return false;
          }
        }
        this.testResult = res.data;
        this.testError = null;
        this.testState = TEST_SUCCESS;
        return true;
      } catch (e) {
        console.error(e);
        this.testState = TEST_FAILED;
        this.testResult = null;
        this.testError = e.response?.data?.error || e.message;
      }
      return false;
    },
    handleCreateError(e) {
      this.handleError(e, "create failed");
    },
    handleUpdateError(e) {
      this.handleError(e, "update failed");
    },
    handleRemoveError(e) {
      this.handleError(e, "remove failed");
    },
    handleError(e, msg) {
      console.error(e);
      let message = msg;
      const { error } = e.response.data || {};
      if (error) message += `: ${error}`;
      alert(message);
    },
  },
};
