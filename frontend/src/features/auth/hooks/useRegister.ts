import { useMutation } from '@tanstack/react-query';
import type { RegisterRequest } from '../../../api/types';
import { useAuth } from '../AuthContext';

export const useRegister = () => {
  const { register } = useAuth();

  return useMutation({
    mutationFn: (payload: RegisterRequest) => register(payload),
  });
};
