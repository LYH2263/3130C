import { apiRequest } from './client';

export const commonApi = {
  getClasses() {
    return apiRequest('/classes');
  },

  getMe(token) {
    return apiRequest('/me', { token });
  },
};
