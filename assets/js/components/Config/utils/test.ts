import type { AxiosResponse } from "axios";
import sleep from "@/utils/sleep";
import { ADMIN_PASSWORD_REQUIRED } from "../DeviceModal/index";
import { reportValidityInModal } from "./reportValidityInModal";

export type TestState = {
  isUnknown: boolean;
  isSuccess: boolean;
  isError: boolean;
  isRunning: boolean;
  loginRequired: boolean;
  result: Record<string, any> | null;
  error: string | null;
  errorLine: number | null;
};

export const initialTestState = (): TestState => ({
  isUnknown: true,
  isSuccess: false,
  isError: false,
  isRunning: false,
  loginRequired: false,
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
  if (form && !reportValidityInModal(form as HTMLFormElement)) return false;
  state.isUnknown = false;
  state.isSuccess = false;
  state.isRunning = true;
  state.loginRequired = false;
  const startTime = Date.now();
  try {
    const res = await api();
    if (res.status === ADMIN_PASSWORD_REQUIRED) {
      state.isUnknown = true; // not testable until the admin password is provided
      return false;
    }
    state.isError = false;
    state.error = null;
    state.errorLine = null;
    for (const [key, value] of Object.entries(res.data)) {
      const { error, loginRequired } = value as { error?: string; loginRequired?: boolean };
      // waiting for the user to log in is a pending state, not a failure
      if (loginRequired) {
        state.loginRequired = true;
        state.result = res.data;
        return false;
      }
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
