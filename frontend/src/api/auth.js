import { apiRequest } from './client';

export const authApi = {
  login(payload) {
    return apiRequest('/auth/login', {
      method: 'POST',
      body: payload,
    });
  },

  register(payload) {
    return apiRequest('/auth/register', {
      method: 'POST',
      body: payload,
    });
  },
};
