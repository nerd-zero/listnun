import { z } from 'zod';

const COLOR_SCHEMA = z
  .string()
  .regex(/^#[0-9a-fA-F]{6}$/)
  .nullable()
  .optional();

const FONT_FAMILY_SCHEMA = z
  .enum([
    'MODERN_SANS',
    'BOOK_SANS',
    'ORGANIC_SANS',
    'GEOMETRIC_SANS',
    'HEAVY_SANS',
    'ROUNDED_SANS',
    'MODERN_SERIF',
    'BOOK_SERIF',
    'MONOSPACE',
  ])
  .nullable()
  .optional();

/** The email's content width in px when a template doesn't set one. */
export const DEFAULT_LAYOUT_WIDTH = 600;
/** Narrowest and widest content widths the builder offers. */
export const MIN_LAYOUT_WIDTH = 480;
export const MAX_LAYOUT_WIDTH = 800;

const EmailLayoutPropsSchema = z.object({
  backdropColor: COLOR_SCHEMA,
  borderColor: COLOR_SCHEMA,
  borderRadius: z.number().optional().nullable(),
  width: z.number().int().min(MIN_LAYOUT_WIDTH).max(MAX_LAYOUT_WIDTH).optional().nullable(),
  canvasColor: COLOR_SCHEMA,
  textColor: COLOR_SCHEMA,
  fontFamily: FONT_FAMILY_SCHEMA,
  childrenIds: z.array(z.string()).optional().nullable(),
  outlook: z.boolean().optional().nullable(),
});

export default EmailLayoutPropsSchema;

export type EmailLayoutProps = z.infer<typeof EmailLayoutPropsSchema>;
