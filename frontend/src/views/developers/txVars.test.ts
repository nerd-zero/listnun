import { describe, expect, it } from 'vitest';
import { missingVars, txDataVars } from './txVars';

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
