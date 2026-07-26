export const queryKeys = {
  auth: {
    me: () => ['auth', 'me'] as const,
  },
  posts: {
    home: () => ['posts', 'home'] as const,
    homeLatest: () => ['posts', 'home', 'latest-check'] as const,
    explore: () => ['posts', 'explore'] as const,
    detail: (id: string) => ['posts', 'detail', id] as const,
    comments: (id: string) => ['posts', id, 'comments'] as const,
  },
  users: {
    profile: (handle: string) => ['users', 'profile', handle] as const,
    posts: (handle: string) => ['users', handle, 'posts'] as const,
    replies: (handle: string) => ['users', handle, 'replies'] as const,
    followers: (handle: string) => ['users', handle, 'followers'] as const,
    following: (handle: string) => ['users', handle, 'following'] as const,
  },
  notifications: {
    list: () => ['notifications', 'list'] as const,
    unreadCount: () => ['notifications', 'unread-count'] as const,
  },
  bookmarks: {
    list: () => ['bookmarks', 'list'] as const,
  },
  search: {
    users: (q: string) => ['search', 'users', q] as const,
    posts: (q: string) => ['search', 'posts', q] as const,
    hashtag: (tag: string) => ['search', 'hashtag', tag] as const,
  },
  admin: {
    users: () => ['admin', 'users'] as const,
  },
} as const;
