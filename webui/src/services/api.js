// src/services/api.js
import axios from 'axios';
import config from '@/config';

// Create a pre-configured axios instance
const api = axios.create({
  baseURL: config.apiBaseUrl,
});

export default api; 