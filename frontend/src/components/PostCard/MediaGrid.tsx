import { Box } from '@mui/material';
import type { MediaResponse } from '../../api/types';

interface MediaGridProps {
  media: MediaResponse[];
}

const GRID_TEMPLATE: Record<1 | 2 | 3 | 4, { gridTemplateColumns: string; gridTemplateRows: string }> = {
  1: { gridTemplateColumns: '1fr', gridTemplateRows: '1fr' },
  2: { gridTemplateColumns: '1fr 1fr', gridTemplateRows: '1fr' },
  3: { gridTemplateColumns: '1fr 1fr', gridTemplateRows: '1fr 1fr' },
  4: { gridTemplateColumns: '1fr 1fr', gridTemplateRows: '1fr 1fr' },
};

export const MediaGrid = ({ media }: MediaGridProps) => {
  const sorted = [...media].sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0));
  const video = sorted.find((item) => item.type === 'video');

  if (video) {
    return (
      <Box sx={{ mt: 1, borderRadius: 2, overflow: 'hidden' }}>
        <Box
          component="video"
          src={video.url}
          controls
          sx={{ width: '100%', maxHeight: 500, display: 'block', bgcolor: 'black' }}
        />
      </Box>
    );
  }

  if (sorted.length === 0) return null;
  const count = Math.min(sorted.length, 4) as 1 | 2 | 3 | 4;

  return (
    <Box
      sx={{
        mt: 1,
        display: 'grid',
        gap: '2px',
        borderRadius: 2,
        overflow: 'hidden',
        height: count === 1 ? 'auto' : 300,
        ...GRID_TEMPLATE[count],
      }}
    >
      {sorted.slice(0, 4).map((item, index) => (
        <Box
          key={item.id}
          component="img"
          src={item.url}
          alt=""
          sx={{
            width: '100%',
            height: count === 1 ? 'auto' : '100%',
            maxHeight: count === 1 ? 500 : undefined,
            objectFit: 'cover',
            gridRow: count === 3 && index === 0 ? 'span 2' : undefined,
          }}
        />
      ))}
    </Box>
  );
};
