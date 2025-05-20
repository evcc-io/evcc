import type { AxiosResponse } from "axios";

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

export const performTest = async (
	state: TestState,
	api: () => Promise<AxiosResponse<any, any>>,
	form: HTMLElement | undefined
) => {
	if (form && !(form as HTMLFormElement).reportValidity()) return false;
	state.isUnknown = false;
	state.isSuccess = false;
	state.isError = false;
	state.isRunning = true;
	state.errorLine = null;
	try {
		const res = await api();
		for (const [key, value] of Object.entries(res.data.result)) {
			const { error } = value as { error?: string };
			if (error) {
				state.isError = true;
				state.error = `${key}: ${error}`;
				return false;
			}
		}
		state.isSuccess = true;
		state.result = res.data.result;
		return true;
	} catch (e: any) {
		state.isError = true;
		state.error = e.response?.data?.error || e.message;
		state.errorLine = e.response?.data?.line || null;
	} finally {
		state.isRunning = false;
	}
	return false;
};
