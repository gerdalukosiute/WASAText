// src/services/api.js
import axios from 'axios';

// Create a pre-configured axios instance
const api = axios.create({
  baseURL: __API_URL__,
	timeout: 1000 * 5
});

export default api; 