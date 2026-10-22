import { alpha, createTheme, type Shadows } from '@mui/material/styles';

/**
 * Listnun builder theme.
 *
 * The values mirror the admin app's --ln-* tokens (frontend/src/assets/tokens.css)
 * one to one, so the builder chrome and the app around it never drift apart.
 * The mapping is:
 *
 *   palette.primary.main          -> --ln-ink
 *   palette.primary.contrastText  -> --ln-on-ink
 *   palette.background.default    -> --ln-bg
 *   palette.background.paper      -> --ln-card
 *   palette.text.primary          -> --ln-text
 *   palette.text.secondary        -> --ln-text-3
 *   palette.divider               -> --ln-border
 *   palette.action.hover          -> --ln-hover
 *   palette.action.selected       -> --ln-muted
 *   OutlinedInput border          -> --ln-border-strong
 *   shape.borderRadius            -> radius md, 6px
 *   Dialog paper radius           -> radius lg, 10px
 *   focus-visible outline         -> --ln-ring
 *
 * Only the chrome is themed. The email canvas keeps the customer's own
 * styles, so nothing here may reach into the rendered email.
 */

export type ThemeMode = 'light' | 'dark';

const FONT_FAMILY = "'IBM Plex Sans', system-ui, -apple-system, 'Segoe UI', sans-serif";
const MONO_FONT_FAMILY = "'IBM Plex Mono', ui-monospace, SFMono-Regular, Menlo, monospace";

/** The amber focus ring, shared with the admin app's PrimeVue preset. */
export const RING = '#d97d3d';

/** Status base colours. These are for dots, bars and icons, not for text. */
const SUCCESS = '#3f8f6c';
const WARNING = '#d97d3d';
const ERROR = '#c1503b';

interface ModeColors {
  ink: string;
  onInk: string;
  bg: string;
  card: string;
  muted: string;
  hover: string;
  text: string;
  textSecondary: string;
  border: string;
  borderStrong: string;
}

const LIGHT: ModeColors = {
  ink: '#1b1e24',
  onInk: '#f6f3ec',
  bg: '#f6f3ec',
  card: '#ffffff',
  muted: '#efebe1',
  hover: '#f3f0e8',
  text: '#1b1e24',
  textSecondary: '#6b6f76',
  border: '#dedad0',
  borderStrong: '#cbc6b9',
};

const DARK: ModeColors = {
  ink: '#ece9e1',
  onInk: '#14171c',
  bg: '#14171c',
  card: '#1b1f26',
  muted: '#181b21',
  hover: '#22262e',
  text: '#ece9e1',
  textSecondary: '#9599a0',
  border: '#2b2f36',
  borderStrong: '#3a3f47',
};

export const listnunTheme = (mode: ThemeMode) => {
  const d = mode === 'dark';
  const c = d ? DARK : LIGHT;

  return createTheme({
    palette: {
      mode,
      primary: { main: c.ink, contrastText: c.onInk },
      success: { main: SUCCESS },
      warning: { main: WARNING },
      error: { main: ERROR },
      background: { default: c.bg, paper: c.card },
      text: { primary: c.text, secondary: c.textSecondary },
      divider: c.border,
      action: { hover: c.hover, selected: c.muted },
    },
    typography: {
      fontFamily: FONT_FAMILY,
      fontSize: 14,
      button: { textTransform: 'none', fontWeight: 500 },
      body1: { fontSize: 14 },
      body2: { fontSize: 12 },
      overline: { fontSize: 11, fontWeight: 600, letterSpacing: '0.05em' },
      caption: { fontSize: 12, letterSpacing: 0 },
    },
    shape: { borderRadius: 6 },
    shadows: [
      'none',
      '0 1px 2px rgba(20, 23, 28, .06)',
      '0 2px 6px rgba(20, 23, 28, .08)',
      '0 4px 12px rgba(20, 23, 28, .10)',
      '0 8px 24px rgba(20, 23, 28, .12)',
      // MUI wants 25 levels; the builder only uses the first four.
      ...Array(20).fill('none'),
    ] as Shadows,
    components: {
      MuiCssBaseline: {
        styleOverrides: {
          // The amber focus ring, matching the admin app's PrimeVue focusRing.
          ':focus-visible': { outline: `2px solid ${RING}`, outlineOffset: 2 },
          // Text fields are excluded: they get the ring on the field root
          // (see MuiOutlinedInput) so it wraps the whole field instead of
          // drawing a second box inside it.
          'input:focus-visible, textarea:focus-visible': { outline: 'none' },
          address: { fontStyle: 'normal' },
          fieldset: { border: 'none', padding: 0 },
          pre: {
            fontFamily: MONO_FONT_FAMILY,
            whiteSpace: 'pre-wrap',
            fontSize: 12,
          },
        },
      },
      MuiSvgIcon: {
        styleOverrides: {
          // The builder may keep MUI icons, but they are sized 18px to match
          // the PrimeIcons in the admin app.
          fontSizeSmall: { fontSize: 18 },
          fontSizeMedium: { fontSize: 18 },
        },
      },
      MuiButtonBase: {
        defaultProps: { disableRipple: true },
        styleOverrides: {
          root: {
            '&.MuiButton-containedSecondary.Mui-disabled': {
              backgroundColor: c.muted,
            },
          },
        },
      },
      MuiButton: {
        defaultProps: { disableElevation: true },
        styleOverrides: {
          // A minimum, not a fixed height: buttons with more than one line
          // (the add-block tiles: icon over label) must be able to grow.
          root: { minHeight: 36, padding: '0 14px' },
          outlinedPrimary: {
            borderColor: c.borderStrong,
            color: c.text,
            '&:hover, &:active, &:focus': { borderColor: c.textSecondary, color: c.text },
          },
          containedSecondary: {
            backgroundColor: c.card,
            border: `1px solid ${c.borderStrong}`,
            color: c.text,
            '&:hover, &:active, &:focus': {
              backgroundColor: c.card,
              borderColor: c.textSecondary,
              color: c.text,
            },
          },
        },
      },
      MuiIconButton: {
        styleOverrides: {
          edgeStart: { marginLeft: -8 },
          colorSecondary: { color: c.textSecondary },
        },
      },
      MuiButtonGroup: { defaultProps: { disableElevation: true } },
      MuiToggleButton: {
        styleOverrides: {
          root: { paddingLeft: 12, paddingRight: 12 },
        },
      },
      MuiPaper: {
        defaultProps: { elevation: 2, square: true },
        styleOverrides: { root: { backgroundImage: 'none' } },
      },
      MuiDialog: {
        defaultProps: { fullWidth: true },
        styleOverrides: { paper: { borderRadius: 10 } },
      },
      MuiDialogTitle: {
        defaultProps: { variant: 'h4' },
        styleOverrides: { root: { paddingTop: 24, paddingBottom: 8 } },
      },
      MuiDialogContent: {
        styleOverrides: { root: { paddingTop: 8, paddingBottom: 16 } },
      },
      MuiDialogActions: {
        styleOverrides: {
          root: {
            borderTop: '1px solid',
            borderTopColor: c.border,
            marginTop: 20,
            padding: '12px 24px',
          },
        },
      },
      MuiOutlinedInput: {
        styleOverrides: {
          // Light mode: the field is the card colour. Dark mode: the field is
          // the darker page colour, so it reads as recessed against the card.
          root: {
            background: d ? '#14171c' : '#ffffff',
            // The ring wraps the whole field, so it goes on the root and
            // fires on focus-within rather than on the inner input.
            '&.Mui-focused': { outline: `2px solid ${RING}`, outlineOffset: 2 },
          },
          notchedOutline: { borderColor: c.borderStrong },
        },
      },
      MuiInput: {
        styleOverrides: {
          underline: {
            '&::before': { borderBottomColor: c.borderStrong },
            '&:hover:not(.Mui-disabled, .Mui-error):before': { borderBottomColor: c.textSecondary },
            '&.Mui-focused::after': { borderBottomColor: c.ink },
          },
        },
      },
      MuiInputBase: {
        styleOverrides: {
          input: {
            fontSize: 14,
            '&.Mui-disabled': {
              WebkitTextFillColor: 'inherit',
              color: c.textSecondary,
            },
          },
        },
      },
      MuiInputLabel: {
        defaultProps: { shrink: true },
        styleOverrides: {
          root: { color: c.textSecondary, '&.Mui-focused': { color: c.text } },
        },
      },
      MuiInputAdornment: {
        styleOverrides: {
          root: {
            '& .MuiTypography-root': { fontSize: 14, color: c.textSecondary },
          },
        },
      },
      MuiTabs: {
        defaultProps: { variant: 'scrollable' },
        styleOverrides: { indicator: { height: 1, backgroundColor: c.text } },
      },
      MuiTab: {
        styleOverrides: {
          root: {
            textTransform: 'none',
            minWidth: 16,
            paddingLeft: 12,
            paddingRight: 12,
            fontSize: 14,
            fontFamily: FONT_FAMILY,
            lineHeight: 1.5,
            fontWeight: 500,
            transition: 'color 0.2s',
            '&.Mui-selected': { color: c.text },
            '&:hover': { color: c.text },
          },
        },
      },
      MuiTooltip: {
        styleOverrides: {
          tooltip: { fontSize: 12, backgroundColor: alpha(c.text, 0.9) },
        },
      },
      MuiSlider: {
        styleOverrides: {
          root: { height: 1 },
          track: { height: 1, border: 'none' },
          rail: { height: 1, backgroundColor: c.borderStrong },
          mark: { backgroundColor: c.borderStrong },
          markActive: { height: 0 },
          thumb: {
            height: 16,
            width: 16,
            cursor: 'col-resize',
            '&:hover, &.Mui-active, &.Mui-focusVisible': {
              boxShadow: `0 0 0 4px ${alpha(c.ink, 0.2)}`,
            },
            '&:before': { display: 'none' },
          },
        },
      },
      MuiAlert: {
        styleOverrides: {
          root: { fontSize: 14 },
          action: { paddingTop: 0, marginRight: 0 },
          filledSuccess: { backgroundColor: SUCCESS },
        },
      },
      MuiAvatar: {
        styleOverrides: {
          root: { textTransform: 'uppercase', fontSize: 14 },
        },
      },
    },
  });
};

export default listnunTheme;
