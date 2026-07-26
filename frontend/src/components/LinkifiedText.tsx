import { Link } from '@mui/material';
import { Fragment } from 'react';
import { Link as RouterLink } from 'react-router-dom';

interface LinkifiedTextProps {
  text: string;
}

const TOKEN_PATTERN = /(#[\p{L}\p{N}_]+|@[A-Za-z0-9_]+)/gu;

export const LinkifiedText = ({ text }: LinkifiedTextProps) => {
  const parts = text.split(TOKEN_PATTERN);

  return (
    <>
      {parts.map((part, index) => {
        if (part.startsWith('#')) {
          return (
            <Link
              key={index}
              component={RouterLink}
              to={`/explore?q=${encodeURIComponent(part)}`}
              underline="hover"
              onClick={(e) => e.stopPropagation()}
            >
              {part}
            </Link>
          );
        }
        if (part.startsWith('@')) {
          return (
            <Link
              key={index}
              component={RouterLink}
              to={`/${part.slice(1)}`}
              underline="hover"
              onClick={(e) => e.stopPropagation()}
            >
              {part}
            </Link>
          );
        }
        return <Fragment key={index}>{part}</Fragment>;
      })}
    </>
  );
};
