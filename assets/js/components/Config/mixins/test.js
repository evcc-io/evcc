const TEST_UNKNOWN = "unknown";
const TEST_SUCCESS = "success";
const TEST_FAILED = "failed";
const TEST_RUNNING = "running";

export default {
  data() {
    return {
      testResult: "",
      testState: TEST_UNKNOWN,
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
      this.testResult = null;
    },

    async test(testApi) {
      if (!this.$refs.form.reportValidity()) return false;
      this.testState = TEST_RUNNING;
      try {
        await testApi();
        this.testState = TEST_SUCCESS;
        this.testResult = null;
        return true;
      } catch (e) {
        console.error(e);
        this.testState = TEST_FAILED;
        this.testResult = e.response?.data?.error || e.message;
      }
      return false;
    },
  },
};
