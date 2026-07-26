import { useCreatePost } from '../hooks/useCreatePost';
import { ComposeBox } from './ComposeBox';

interface PostFormProps {
  onSuccess?: () => void;
}

export const PostForm = ({ onSuccess }: PostFormProps) => {
  const { mutate, isPending, error } = useCreatePost();

  return (
    <ComposeBox
      placeholder="いまどうしてる？"
      submitLabel="投稿"
      isPending={isPending}
      error={error}
      onSubmit={(payload, reset) =>
        mutate(payload, {
          onSuccess: () => {
            reset();
            onSuccess?.();
          },
        })
      }
    />
  );
};
