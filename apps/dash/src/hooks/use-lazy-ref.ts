import { useRef } from "react";

export function useLazyRef<T>(fn: () => T) {
  const ref = useRef<null | T>(null);

  if (ref.current === null) {
    ref.current = fn();
  }

  return ref as React.RefObject<T>;
}
