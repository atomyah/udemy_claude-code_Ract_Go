import { api } from '../../api/client';
import type { AdminUserListResponse } from '../../api/types';
import type { PageParams } from '../posts/api';

export const getAdminUsers = ({ cursor, limit = 20 }: PageParams): Promise<AdminUserListResponse> =>
  api.get<AdminUserListResponse>('/admin/users', { params: { cursor, limit } }).then((res) => res.data);

export const suspendUser = (id: string): Promise<void> => api.put(`/admin/users/${id}/suspend`).then(() => undefined);

export const unsuspendUser = (id: string): Promise<void> =>
  api.delete(`/admin/users/${id}/suspend`).then(() => undefined);

export const adminDeletePost = (id: string): Promise<void> => api.delete(`/admin/posts/${id}`).then(() => undefined);
