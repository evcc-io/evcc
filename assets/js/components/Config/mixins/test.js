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
        const result = [];
        for (const [key, { value, error }] of Object.entries(res.data.result)) {
          if (error) {
            this.testState = TEST_FAILED;
            this.testResult = null;
            this.testError = `${key}: ${error}`;
            return false;
          }
          result.push({ key, value });
        }
        this.testResult = result;
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
  },
};
