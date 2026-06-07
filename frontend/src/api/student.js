import { apiRequest } from './client';

export const studentApi = {
  getMistakes(token) {
    return apiRequest('/student/mistakes', { token });
  },

  getAttempts(token) {
    return apiRequest('/student/attempts', { token });
  },

  getQuestions(token, limit = 10) {
    return apiRequest(`/student/questions?limit=${limit}`, { token });
  },

  submitQuiz(token, answers) {
    return apiRequest('/student/submit', {
      method: 'POST',
      token,
      body: { answers },
    });
  },

  loadStudentData(token) {
    return Promise.all([
      studentApi.getMistakes(token),
      studentApi.getAttempts(token),
    ]);
  },
};
