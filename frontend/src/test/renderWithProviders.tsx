import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, type RenderOptions, type RenderResult } from '@testing-library/react';
import type { ReactElement, ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { ToastProvider } from '../components/ToastProvider';

interface RenderWithProvidersOptions extends Omit<RenderOptions, 'wrapper'> {
  /** 初期URL（React Router のルーティングを伴うテストで使う） */
  route?: string;
}

/** テストごとにリトライ・キャッシュを無効にしたQueryClientを作る */
const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0, staleTime: 0 },
      mutations: { retry: false },
    },
  });

/**
 * アプリで必須のProvider（QueryClient / Router / Toast）を適用してレンダリングする。
 * AuthContext は呼び出し側で vi.mock するため含めない。
 */
export const renderWithProviders = (
  ui: ReactElement,
  { route = '/', ...options }: RenderWithProvidersOptions = {},
): RenderResult => {
  const queryClient = createTestQueryClient();

  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[route]}>
        <ToastProvider>{children}</ToastProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );

  return render(ui, { wrapper: Wrapper, ...options });
};
