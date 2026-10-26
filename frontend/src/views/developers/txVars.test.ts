import { describe, expect, it } from 'vitest';
import { missingVars, txDataVars, withMissing } from './txVars';

describe('txDataVars', () => {
  it('finds top-level data keys across the subject and body, once each', () => {
    const subject = 'Your order #{{ .Tx.Data.order_id }} is confirmed';
    const body = '{{ range .Tx.Data.items }}{{ .name }}{{ end }} {{ .Tx.Data.total }} {{ .Tx.Data.order_id }}';
    expect(txDataVars(subject, body)).toEqual(['order_id', 'items', 'total']);
  });

  it('ignores subscriber fields and nested paths beyond the first key', () => {
    expect(txDataVars('{{ .Subscriber.FirstName }} {{ .Tx.Data.ship.city }}')).toEqual(['ship']);
  });
});

describe('missingVars', () => {
  it('reports the variables the data lacks', () => {
    expect(missingVars(['order_id', 'total'], { order_id: '1' })).toEqual(['total']);
  });
});

describe('withMissing', () => {
  it('adds an empty value for each missing variable and keeps filled ones', () => {
    const got = withMissing(['order_id', 'total', 'items'], { total: '$49.00', extra: 1 });
    expect(got).toEqual({
      total: '$49.00', extra: 1, order_id: '', items: '',
    });
  });

  it('returns the data unchanged when nothing is missing', () => {
    expect(withMissing(['a'], { a: 1 })).toEqual({ a: 1 });
  });
});
