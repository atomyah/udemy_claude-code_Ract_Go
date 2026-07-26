import { api } from '../../api/client';
import type {
  AvatarResponse,
  BannerResponse,
  PostListResponse,
  SuccessResponse,
  UpdateProfileRequest,
  UserListResponse,
  UserResponse,
} from '../../api/types';
import type { PageParams } from '../posts/api';

export const getProfile = (handle: string): Promise<UserResponse> =>
  api.get<UserResponse>(`/users/${handle}`).then((res) => res.data);

export const getUserPosts = (handle: string, { cursor, limit = 20 }: PageParams): Promise<PostListResponse> =>
  api.get<PostListResponse>(`/users/${handle}/posts`, { params: { cursor, limit } }).then((res) => res.data);

export const getUserReplies = (handle: string, { cursor, limit = 20 }: PageParams): Promise<PostListResponse> =>
  api.get<PostListResponse>(`/users/${handle}/replies`, { params: { cursor, limit } }).then((res) => res.data);

export const getFollowers = (handle: string, { cursor, limit = 20 }: PageParams): Promise<UserListResponse> =>
  api.get<UserListResponse>(`/users/${handle}/followers`, { params: { cursor, limit } }).then((res) => res.data);

export const getFollowing = (handle: string, { cursor, limit = 20 }: PageParams): Promise<UserListResponse> =>
  api.get<UserListResponse>(`/users/${handle}/following`, { params: { cursor, limit } }).then((res) => res.data);

export const followUser = (handle: string): Promise<SuccessResponse> =>
  api.post<SuccessResponse>(`/users/${handle}/follow`).then((res) => res.data);

export const unfollowUser = (handle: string): Promise<void> => api.delete(`/users/${handle}/follow`).then(() => undefined);

export const updateProfile = (payload: UpdateProfileRequest): Promise<UserResponse> =>
  api.put<UserResponse>('/users/me', payload).then((res) => res.data);

export const uploadAvatar = (file: File): Promise<AvatarResponse> => {
  const formData = new FormData();
  formData.append('avatar', file);
  return api
    .put<AvatarResponse>('/users/me/avatar', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
    .then((res) => res.data);
};

export const uploadBanner = (file: File): Promise<BannerResponse> => {
  const formData = new FormData();
  formData.append('banner', file);
  return api
    .put<BannerResponse>('/users/me/banner', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
    .then((res) => res.data);
};
