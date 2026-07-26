import { useMutation } from '@tanstack/react-query';
import { useAuth } from '../AuthContext';

interface LoginInput {
  email: string;
  password: string;
}

export const useLogin = () => {
  const { login } = useAuth();

  return useMutation({
    mutationFn: ({ email, password }: LoginInput) => login(email, password),
  });
};
