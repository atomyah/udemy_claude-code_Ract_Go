import { Alert, Snackbar } from '@mui/material';
import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from 'react';

export type ToastSeverity = 'success' | 'error' | 'info' | 'warning';

interface Toast {
  id: number;
  message: string;
  severity: ToastSeverity;
}

interface ToastContextValue {
  showToast: (message: string, severity?: ToastSeverity) => void;
}

const ToastContext = createContext<ToastContextValue | undefined>(undefined);

const AUTO_HIDE_DURATION = 4000;

export const ToastProvider = ({ children }: { children: ReactNode }) => {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const nextId = useRef(0);
  const current = toasts[0];

  const showToast = useCallback((message: string, severity: ToastSeverity = 'info') => {
    nextId.current += 1;
    setToasts((prev) => [...prev, { id: nextId.current, message, severity }]);
  }, []);

  const closeCurrent = useCallback(() => {
    setToasts((prev) => prev.slice(1));
  }, []);

  const value = useMemo(() => ({ showToast }), [showToast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      {current && (
        <Snackbar
          key={current.id}
          open
          autoHideDuration={AUTO_HIDE_DURATION}
          onClose={(_event, reason) => {
            if (reason !== 'clickaway') closeCurrent();
          }}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
        >
          <Alert
            severity={current.severity}
            variant="filled"
            onClose={closeCurrent}
            data-testid="toast"
            sx={{ width: '100%' }}
          >
            {current.message}
          </Alert>
        </Snackbar>
      )}
    </ToastContext.Provider>
  );
};

export const useToast = (): ToastContextValue => {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
};
