import { render, screen } from '@testing-library/react';
import { describe, expect, test } from 'vitest';
import type { MediaResponse } from '../../api/types';
import { MediaGrid } from './MediaGrid';

const signedUrl = (name: string) =>
  `https://storage.googleapis.com/bucket/posts/p/${name}.png?X-Goog-Signature=abc`;

const image = (name: string, sortOrder: number): MediaResponse => ({
  id: name,
  url: signedUrl(name),
  type: 'image',
  sort_order: sortOrder,
});

describe('MediaGrid', () => {
  test('メディアがないときは何も表示しない', () => {
    const { container } = render(<MediaGrid media={[]} />);

    expect(container).toBeEmptyDOMElement();
  });

  test('APIが返した署名付きURLをそのまま使い、sort_order の順に画像を表示する', () => {
    render(<MediaGrid media={[image('c', 2), image('a', 0), image('d', 3), image('b', 1)]} />);

    const srcs = screen.getAllByRole('img').map((img) => img.getAttribute('src'));

    expect(srcs).toEqual([signedUrl('a'), signedUrl('b'), signedUrl('c'), signedUrl('d')]);
  });
});
