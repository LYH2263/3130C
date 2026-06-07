import { useCallback } from 'react';

import { useAsync } from './useAsync';

export function useApi(apiFn, options = {}) {
  const { immediate = false, deps = [], ...restOptions } = options;

  const asyncFn = useCallback(apiFn, [apiFn]);

  const result = useAsync(asyncFn, {
    ...restOptions,
    immediate: false,
  });

  const { execute } = result;

  const refetch = useCallback(
    (...args) => execute(...args),
    [execute]
  );

  return {
    ...result,
    refetch,
  };
}
