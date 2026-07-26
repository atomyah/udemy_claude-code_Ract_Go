import axios from 'axios';
import type { ErrorResponse } from '../api/types';

export const getApiErrorMessage = (error: unknown, fallback: string): string => {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as ErrorResponse | undefined;
    if (data?.message) return data.message;
  }
  return fallback;
};
