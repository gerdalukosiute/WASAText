// src/services/api.js
import axios from 'axios';

// Create a pre-configured axios instance
const api = axios.create({
  baseURL: __API_URL__,
	timeout: 1000 * 5
});

// Add request interceptor for debugging
api.interceptors.request.use(
  (config) => {
    console.log(`API Request: ${config.method.toUpperCase()} ${config.url}`, config)
    return config
  },
  (error) => {
    console.error("API Request Error:", error)
    return Promise.reject(error)
  },
 )
 
 // Add response interceptor for debugging
 api.interceptors.response.use(
  (response) => {
    console.log(`API Response: ${response.status} ${response.config.url}`, response.data)
    return response
  },
  (error) => {
    console.error("API Response Error:", error)
    return Promise.reject(error)
  },
 ) 

export default api; 