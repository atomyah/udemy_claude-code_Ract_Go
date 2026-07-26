import { Dialog, DialogTitle } from '@mui/material';
import { createContext, useContext, useMemo, useState, type ReactNode } from 'react';
import { PostForm } from './components/PostForm';

interface ComposerContextValue {
  openComposer: () => void;
}

const ComposerContext = createContext<ComposerContextValue | undefined>(undefined);

export const ComposerProvider = ({ children }: { children: ReactNode }) => {
  const [open, setOpen] = useState(false);

  const value = useMemo<ComposerContextValue>(() => ({ openComposer: () => setOpen(true) }), []);

  return (
    <ComposerContext.Provider value={value}>
      {children}
      <Dialog open={open} onClose={() => setOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>投稿する</DialogTitle>
        <PostForm onSuccess={() => setOpen(false)} />
      </Dialog>
    </ComposerContext.Provider>
  );
};

export const useComposer = (): ComposerContextValue => {
  const context = useContext(ComposerContext);
  if (!context) {
    throw new Error('useComposer must be used within a ComposerProvider');
  }
  return context;
};
