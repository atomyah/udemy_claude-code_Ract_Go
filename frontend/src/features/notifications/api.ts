import { api } from '../../api/client';
import type { NotificationListResponse } from '../../api/types';
import type { PageParams } from '../posts/api';

export const getNotifications = ({ cursor, limit = 20 }: PageParams): Promise<NotificationListResponse> =>
  api.get<NotificationListResponse>('/notifications', { params: { cursor, limit } }).then((res) => res.data);

export const markAllRead = (): Promise<void> => api.put('/notifications/read').then(() => undefined);
