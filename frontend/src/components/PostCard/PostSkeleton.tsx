import { Box, Card, CardContent, Skeleton, Stack } from '@mui/material';

export const PostSkeleton = () => (
  <Card variant="outlined" sx={{ borderRadius: 0, borderLeft: 0, borderRight: 0, borderTop: 0 }}>
    <CardContent>
      <Stack direction="row" spacing={1.5}>
        <Skeleton variant="circular" width={40} height={40} />
        <Box sx={{ flexGrow: 1 }}>
          <Skeleton variant="text" width="40%" />
          <Skeleton variant="text" width="90%" />
          <Skeleton variant="text" width="60%" />
          <Skeleton variant="rectangular" height={200} sx={{ mt: 1, borderRadius: 2 }} />
        </Box>
      </Stack>
    </CardContent>
  </Card>
);
