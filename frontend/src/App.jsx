import { useCallback, useEffect, useState } from 'react';
import { Toaster, toast } from 'react-hot-toast';

import { authApi, commonApi, setUnauthorizedHandler } from './api';
import { AuthPage } from './pages/AuthPage';
import { StudentDashboard } from './pages/StudentDashboard';
import { TeacherDashboard } from './pages/TeacherDashboard';

const TOKEN_KEY = 'quizlab_token_3130';

export default function App() {
  const [token, setToken] = useState(() => localStorage.getItem(TOKEN_KEY) || '');
  const [user, setUser] = useState(null);
  const [classes, setClasses] = useState([]);
  const [initializing, setInitializing] = useState(true);
  const [authLoading, setAuthLoading] = useState(false);

  const handleLogout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY);
    setToken('');
    setUser(null);
    toast.success('已退出登录');
  }, []);

  useEffect(() => {
    setUnauthorizedHandler(() => {
      handleLogout();
      toast.error('登录已过期，请重新登录');
    });
  }, [handleLogout]);

  const loadClasses = async () => {
    try {
      const data = await commonApi.getClasses();
      setClasses(data);
    } catch (error) {
      toast.error(error.message || '加载班级失败');
    }
  };

  const loadMe = async (nextToken) => {
    const me = await commonApi.getMe(nextToken);
    setUser(me);
  };

  const initialize = async () => {
    setInitializing(true);
    await loadClasses();
    if (token) {
      try {
        await loadMe(token);
      } catch {
        localStorage.removeItem(TOKEN_KEY);
        setToken('');
        setUser(null);
      }
    }
    setInitializing(false);
  };

  useEffect(() => {
    initialize();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const applyAuth = async (payloadPromise) => {
    setAuthLoading(true);
    try {
      const data = await payloadPromise;
      localStorage.setItem(TOKEN_KEY, data.token);
      setToken(data.token);
      setUser(data.user);
      toast.success(data.user.role === 'teacher' ? '教师登录成功' : '学生登录成功');
    } catch (error) {
      toast.error(error.message || '鉴权失败');
      throw error;
    } finally {
      setAuthLoading(false);
    }
  };

  const handleLogin = (payload) => applyAuth(authApi.login(payload));

  const handleRegister = (payload) => applyAuth(authApi.register(payload));

  if (initializing) {
    return (
      <div className="min-h-screen bg-board px-4 py-12">
        <div className="mx-auto max-w-lg animate-pulse rounded-3xl bg-white/80 p-8 shadow-card">
          <div className="h-5 w-2/3 rounded bg-slate-200" />
          <div className="mt-3 h-4 w-full rounded bg-slate-100" />
          <div className="mt-2 h-4 w-4/5 rounded bg-slate-100" />
        </div>
      </div>
    );
  }

  return (
    <>
      <Toaster position="top-right" toastOptions={{ duration: 2400 }} />
      {!user ? (
        <AuthPage classes={classes} onLogin={handleLogin} onRegister={handleRegister} loading={authLoading} />
      ) : user.role === 'teacher' ? (
        <TeacherDashboard user={user} token={token} onLogout={handleLogout} />
      ) : (
        <StudentDashboard user={user} token={token} onLogout={handleLogout} />
      )}
    </>
  );
}
