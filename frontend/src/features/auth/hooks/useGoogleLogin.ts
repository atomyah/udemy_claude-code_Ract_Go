import { useMutation } from '@tanstack/react-query';
import { useAuth } from '../AuthContext';
import { signInWithGoogle } from '../firebase';
import { loginWithGoogle } from '../api';

export const useGoogleLogin = () => {
  const { loginWithUser } = useAuth();

  return useMutation({
    mutationFn: async () => {
      const idToken = await signInWithGoogle();
      return loginWithGoogle({ id_token: idToken });
    },
    // トークンはバックエンドが HttpOnly Cookie で発行済み。ここではユーザー情報だけを反映する
    onSuccess: ({ user }) => {
      if (user) {
        loginWithUser(user);
      }
    },
  });
};
