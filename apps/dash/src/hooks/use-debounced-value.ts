import { useState } from "react";
import { useDebounce } from "react-use";

export function useDebouncedValue<T>(value: T, duration: number) {
  const [debouncedValue, setDebouncedValue] = useState(value);
  useDebounce(() => setDebouncedValue(value), duration, [value]);
  return debouncedValue;
}
