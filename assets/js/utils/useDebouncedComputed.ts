import { ref, watch, type Ref } from "vue";
import { debounce } from "./debounce";

export function useDebouncedComputed<T>(
  getter: () => T,
  deps: () => any,
  delay: number = 100
): Ref<T> {
  const result = ref<T>();
  const debouncedUpdate = debounce(() => {
    result.value = getter();
  }, delay);

  watch(deps, debouncedUpdate, { immediate: true, deep: true });

  return result as Ref<T>;
}
