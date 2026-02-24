import type { AxiosResponse } from "axios";
import sleep from "@/utils/sleep";

export type TestState = {
  isUnknown: boolean;
  isSuccess: boolean;
  isError: boolean;
  isRunning: boolean;
  result: Record<string, any> | null;
  error: string | null;
  errorLine: number | null;
};

export const initialTestState = (): TestState => ({
  isUnknown: true,
  isSuccess: false,
  isError: false,
  isRunning: false,
  result: null,
  error: null,
  errorLine: null,
});

const MIN_TEST_DURATION = 500;

export const performTest = async (
  state: TestState,
  api: () => Promise<AxiosResponse<any, any>>,
  form: HTMLElement | undefined
) => {
  if (form && !(form as HTMLFormElement).reportValidity()) return false;
  state.isUnknown = false;
  state.isSuccess = false;
  state.isRunning = true;
  const startTime = Date.now();
  try {
    const res = await api();
    state.isError = false;
    state.error = null;
    state.errorLine = null;
    for (const [key, value] of Object.entries(res.data)) {
      const { error } = value as { error?: string };
      if (error) {
        state.isError = true;
        state.error = `${key}: ${error}`;
        return false;
      }
    }
    state.isSuccess = true;
    state.result = res.data;
    return true;
  } catch (e: any) {
    state.isError = true;
    state.error = e.response?.data?.error || e.message;
    state.errorLine = e.response?.data?.line || null;
  } finally {
    const elapsed = Date.now() - startTime;
    const remainingTime = MIN_TEST_DURATION - elapsed;
    if (remainingTime > 0) {
      await sleep(remainingTime);
    }
    state.isRunning = false;
  }
  return false;
};
