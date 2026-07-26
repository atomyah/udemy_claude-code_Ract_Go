import { useMutation } from '@tanstack/react-query';
import { useAuth } from '../AuthContext';
import { signInWithGoogle } from '../firebase';
import { loginWithGoogle } from '../api';

export const useGoogleLogin = () => {
  const { loginWithToken } = useAuth();

  return useMutation({
    mutationFn: async () => {
      const idToken = await signInWithGoogle();
      return loginWithGoogle({ id_token: idToken });
    },
    onSuccess: ({ access_token, user }) => {
      if (access_token && user) {
        loginWithToken(access_token, user);
      }
    },
  });
};
