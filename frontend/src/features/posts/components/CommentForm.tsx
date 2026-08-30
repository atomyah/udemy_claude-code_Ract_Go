import type { RefObject } from 'react';
import { useCreateComment } from '../hooks/useCreateComment';
import { ComposeBox } from './ComposeBox';

interface CommentFormProps {
  postId: string;
  inputRef?: RefObject<HTMLTextAreaElement | null>;
}

export const CommentForm = ({ postId, inputRef }: CommentFormProps) => {
  const { mutate, isPending, error } = useCreateComment(postId);

  return (
    <ComposeBox
      compact
      testIdPrefix="comment-form"
      inputRef={inputRef}
      placeholder="返信を投稿"
      submitLabel="返信"
      isPending={isPending}
      error={error}
      onSubmit={(payload, reset) => mutate(payload, { onSuccess: reset })}
    />
  );
};
