import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useToast } from '../../../components/ToastProvider';
import { getApiErrorMessage } from '../../../utils/apiError';
import { deletePost } from '../api';
import { removePostFromCaches } from '../cacheHelpers';

export const useDeletePost = () => {
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  return useMutation({
    mutationFn: (postId: string) => deletePost(postId),
    onSuccess: (_, postId) => {
      removePostFromCaches(queryClient, postId);
      showToast('投稿を削除しました', 'success');
    },
    onError: (error) => {
      showToast(getApiErrorMessage(error, '投稿の削除に失敗しました'), 'error');
    },
  });
};
