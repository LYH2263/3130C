import { apiRequest } from './client';

export const teacherApi = {
  getOverview(token) {
    return apiRequest('/teacher/overview', { token });
  },

  getQuestions(token) {
    return apiRequest('/teacher/questions', { token });
  },

  getClassStats(token) {
    return apiRequest('/teacher/class-stats', { token });
  },

  getAttempts(token, limit = 50) {
    return apiRequest(`/teacher/attempts?limit=${limit}`, { token });
  },

  createQuestion(token, payload) {
    return apiRequest('/teacher/questions', {
      method: 'POST',
      token,
      body: payload,
    });
  },

  updateQuestion(token, questionId, payload) {
    return apiRequest(`/teacher/questions/${questionId}`, {
      method: 'PUT',
      token,
      body: payload,
    });
  },

  deleteQuestion(token, questionId) {
    return apiRequest(`/teacher/questions/${questionId}`, {
      method: 'DELETE',
      token,
    });
  },

  uploadQuestions(token, formData) {
    return apiRequest('/teacher/questions/upload', {
      method: 'POST',
      token,
      body: formData,
      isForm: true,
    });
  },

  loadDashboard(token) {
    return Promise.all([
      teacherApi.getOverview(token),
      teacherApi.getQuestions(token),
      teacherApi.getClassStats(token),
      teacherApi.getAttempts(token),
    ]);
  },
};
