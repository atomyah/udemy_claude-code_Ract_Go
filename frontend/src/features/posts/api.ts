import { api } from '../../api/client';
import type {
  LikeResponse,
  PostListResponse,
  PostResponse,
  RepostResponse,
  UpdatePostRequest,
} from '../../api/types';

export interface PageParams {
  cursor?: string;
  limit?: number;
}

export interface CreatePostPayload {
  content: string;
  media?: File[];
}

const toFormData = ({ content, media }: CreatePostPayload): FormData => {
  const formData = new FormData();
  formData.append('content', content);
  media?.forEach((file) => formData.append('media', file));
  return formData;
};

export const getHomeTimeline = ({ cursor, limit = 20 }: PageParams): Promise<PostListResponse> =>
  api.get<PostListResponse>('/posts/home', { params: { cursor, limit } }).then((res) => res.data);

export const getExploreTimeline = ({ cursor, limit = 20 }: PageParams): Promise<PostListResponse> =>
  api.get<PostListResponse>('/posts', { params: { cursor, limit } }).then((res) => res.data);

export const searchPosts = (q: string, { cursor, limit = 20 }: PageParams): Promise<PostListResponse> =>
  api.get<PostListResponse>('/search/posts', { params: { q, cursor, limit } }).then((res) => res.data);

export const createPost = (payload: CreatePostPayload): Promise<PostResponse> =>
  api
    .post<PostResponse>('/posts', toFormData(payload), {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((res) => res.data);

export const getPost = (id: string): Promise<PostResponse> =>
  api.get<PostResponse>(`/posts/${id}`).then((res) => res.data);

export const updatePost = (id: string, payload: UpdatePostRequest): Promise<PostResponse> =>
  api.put<PostResponse>(`/posts/${id}`, payload).then((res) => res.data);

export const deletePost = (id: string): Promise<void> => api.delete(`/posts/${id}`).then(() => undefined);

export const getComments = (id: string, { cursor, limit = 20 }: PageParams): Promise<PostListResponse> =>
  api.get<PostListResponse>(`/posts/${id}/comments`, { params: { cursor, limit } }).then((res) => res.data);

export const createComment = (id: string, payload: CreatePostPayload): Promise<PostResponse> =>
  api
    .post<PostResponse>(`/posts/${id}/comments`, toFormData(payload), {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((res) => res.data);

export const like = (id: string): Promise<LikeResponse> =>
  api.post<LikeResponse>(`/posts/${id}/like`).then((res) => res.data);

export const unlike = (id: string): Promise<LikeResponse> =>
  api.delete<LikeResponse>(`/posts/${id}/like`).then((res) => res.data);

export const repost = (id: string, payload: CreatePostPayload = { content: '' }): Promise<RepostResponse> =>
  api
    .post<RepostResponse>(`/posts/${id}/repost`, toFormData(payload), {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((res) => res.data);

export const unrepost = (id: string): Promise<RepostResponse> =>
  api.delete<RepostResponse>(`/posts/${id}/repost`).then((res) => res.data);

export const bookmark = (id: string): Promise<void> => api.post(`/posts/${id}/bookmark`).then(() => undefined);

export const unbookmark = (id: string): Promise<void> => api.delete(`/posts/${id}/bookmark`).then(() => undefined);

export const getBookmarks = ({ cursor, limit = 20 }: PageParams): Promise<PostListResponse> =>
  api.get<PostListResponse>('/bookmarks', { params: { cursor, limit } }).then((res) => res.data);
