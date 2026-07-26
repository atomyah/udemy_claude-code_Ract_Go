import { useMutation } from '@tanstack/react-query';
import { adminDeletePost } from '../api';

export const useAdminDeletePost = () =>
  useMutation({
    mutationFn: (id: string) => adminDeletePost(id),
  });
