import { useCallback, useEffect, useRef, useState } from 'react';
import { toast } from 'react-hot-toast';

export function useAsync(asyncFn, options = {}) {
  const { immediate = false, onSuccess, onError, showErrorToast = false } = options;

  const [data, setData] = useState(undefined);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const mountedRef = useRef(null);

  const execute = useCallback(
    async (...args) => {
      setLoading(true);
      setError(null);
      try {
        const result = await asyncFn(...args);
        if (mountedRef.current) {
          setData(result);
          if (onSuccess) {
            onSuccess(result);
          }
        }
        return result;
      } catch (err) {
        if (mountedRef.current) {
          setError(err);
          if (showErrorToast && err?.status !== 401 && err?.status !== 403) {
            toast.error(err.message || '请求失败');
          }
          if (onError) {
            onError(err);
          }
        }
        throw err;
      } finally {
        if (mountedRef.current) {
          setLoading(false);
        }
      }
    },
    [asyncFn, onSuccess, onError, showErrorToast]
  );

  const reset = useCallback(() => {
    setData(undefined);
    setError(null);
    setLoading(false);
  }, []);

  useEffect(() => {
    mountedRef.current = true;
    if (immediate) {
      execute();
    }
    return () => {
      mountedRef.current = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return { data, loading, error, execute, reset, setData };
}
