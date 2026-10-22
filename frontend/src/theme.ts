import Aura from '@primeuix/themes/aura';
import { definePreset } from '@primeuix/themes';

// Listnun preset: warm paper and ink, carried over from the console.
// Ink is the only brand colour; amber, green and red carry status. The
// app's --ln-* CSS variables (assets/tokens.css) use the same values so
// page CSS and PrimeVue components never drift apart.
const ink = {
  50: '#f6f4ef',
  100: '#ece9e1',
  200: '#d6d3cb',
  300: '#b3b2ad',
  400: '#8b8e94',
  500: '#6b6f76',
  600: '#4a4e56',
  700: '#33373e',
  800: '#1b1e24',
  900: '#14171c',
  950: '#0c0e11',
};

const Listnun = definePreset(Aura, {
  primitive: {
    ink,
    borderRadius: {
      none: '0', xs: '2px', sm: '4px', md: '6px', lg: '10px', xl: '12px',
    },
  },
  semantic: {
    primary: {
      50: '{ink.50}',
      100: '{ink.100}',
      200: '{ink.200}',
      300: '{ink.300}',
      400: '{ink.400}',
      500: '{ink.500}',
      600: '{ink.600}',
      700: '{ink.700}',
      800: '{ink.800}',
      900: '{ink.900}',
      950: '{ink.950}',
    },
    focusRing: {
      width: '2px',
      style: 'solid',
      color: '#d97d3d',
      offset: '2px',
      shadow: 'none',
    },
    formField: {
      borderRadius: '6px',
      // Design spec: 12px side padding. Height is set in patterns.scss
      // (40px on pages, 36px in dialogs) so it doesn't hang on line-height.
      paddingX: '12px',
      paddingY: '9px',
    },
    colorScheme: {
      light: {
        surface: {
          0: '#ffffff',
          50: '#f6f3ec',
          100: '#efebe1',
          200: '#dedad0',
          300: '#cbc6b9',
          400: '#9599a0',
          500: '#6b6f76',
          600: '#4a4e56',
          700: '#33373e',
          800: '#1b1e24',
          900: '#14171c',
          950: '#0c0e11',
        },
        primary: {
          color: '{primary.800}',
          contrastColor: '#f6f3ec',
          hoverColor: '{primary.700}',
          activeColor: '{primary.900}',
        },
        highlight: {
          background: '#efebe1',
          focusBackground: '#e6e1d5',
          color: '#1b1e24',
          focusColor: '#1b1e24',
        },
        mask: {
          background: 'rgba(20,23,28,.4)',
          color: '{surface.200}',
        },
        formField: {
          background: '{surface.0}',
          disabledBackground: '{surface.100}',
          filledBackground: '{surface.50}',
          filledHoverBackground: '{surface.50}',
          filledFocusBackground: '{surface.0}',
          borderColor: '{surface.300}',
          hoverBorderColor: '{surface.400}',
          focusBorderColor: '{primary.color}',
          invalidBorderColor: '#c1503b',
          color: '{surface.800}',
          disabledColor: '{surface.400}',
          placeholderColor: '{surface.400}',
          invalidPlaceholderColor: '#a33f2c',
          floatLabelColor: '{surface.500}',
          floatLabelFocusColor: '{primary.color}',
          floatLabelActiveColor: '{surface.500}',
          floatLabelInvalidColor: '#a33f2c',
          iconColor: '{surface.500}',
          shadow: 'none',
        },
        text: {
          color: '{surface.800}',
          hoverColor: '{surface.900}',
          mutedColor: '{surface.500}',
          hoverMutedColor: '{surface.600}',
        },
        content: {
          background: '{surface.0}',
          hoverBackground: '#f3f0e8',
          borderColor: '{surface.200}',
          color: '{text.color}',
          hoverColor: '{text.hoverColor}',
        },
        overlay: {
          select: {
            background: '{surface.0}', borderColor: '{surface.200}', color: '{text.color}',
          },
          popover: {
            background: '{surface.0}', borderColor: '{surface.200}', color: '{text.color}',
          },
          modal: {
            background: '{surface.0}', borderColor: '{surface.200}', color: '{text.color}',
          },
        },
        list: {
          option: {
            focusBackground: '#f3f0e8',
            selectedBackground: '{highlight.background}',
            selectedFocusBackground: '{highlight.focusBackground}',
            color: '{text.color}',
            focusColor: '{text.hoverColor}',
            selectedColor: '{highlight.color}',
            selectedFocusColor: '{highlight.focusColor}',
            icon: { color: '{surface.400}', focusColor: '{surface.500}' },
          },
          optionGroup: { background: 'transparent', color: '{text.mutedColor}' },
        },
        navigation: {
          item: {
            focusBackground: '#f3f0e8',
            activeBackground: '#efebe1',
            color: '{text.color}',
            focusColor: '{text.hoverColor}',
            activeColor: '{text.hoverColor}',
            icon: { color: '{surface.500}', focusColor: '{surface.600}', activeColor: '{surface.800}' },
          },
          submenuLabel: { background: 'transparent', color: '{text.mutedColor}' },
          submenuIcon: { color: '{surface.400}', focusColor: '{surface.500}', activeColor: '{surface.500}' },
        },
      },
      dark: {
        surface: {
          0: '#ece9e1',
          50: '#d6d3cb',
          100: '#c7c3ba',
          200: '#b3b2ad',
          300: '#9599a0',
          400: '#9599a0',
          500: '#6b6f76',
          600: '#3a3f47',
          700: '#2b2f36',
          800: '#22262e',
          900: '#1b1f26',
          950: '#14171c',
        },
        primary: {
          color: '{primary.100}',
          contrastColor: '#14171c',
          hoverColor: '{primary.200}',
          activeColor: '{primary.50}',
        },
        highlight: {
          background: '#22262e',
          focusBackground: '#2b2f36',
          color: '#ece9e1',
          focusColor: '#ece9e1',
        },
        mask: {
          background: 'rgba(0,0,0,.6)',
          color: '{surface.200}',
        },
        formField: {
          background: '{surface.950}',
          disabledBackground: '{surface.800}',
          filledBackground: '{surface.900}',
          filledHoverBackground: '{surface.900}',
          filledFocusBackground: '{surface.950}',
          borderColor: '{surface.600}',
          hoverBorderColor: '{surface.500}',
          focusBorderColor: '{primary.color}',
          invalidBorderColor: '#e58a76',
          color: '{surface.0}',
          disabledColor: '{surface.500}',
          placeholderColor: '{surface.500}',
          invalidPlaceholderColor: '#e58a76',
          floatLabelColor: '{surface.400}',
          floatLabelFocusColor: '{primary.color}',
          floatLabelActiveColor: '{surface.400}',
          floatLabelInvalidColor: '#e58a76',
          iconColor: '{surface.400}',
          shadow: 'none',
        },
        text: {
          color: '{surface.0}',
          hoverColor: '#f6f3ec',
          mutedColor: '{surface.400}',
          hoverMutedColor: '{surface.100}',
        },
        content: {
          background: '{surface.900}',
          hoverBackground: '{surface.800}',
          borderColor: '{surface.700}',
          color: '{text.color}',
          hoverColor: '{text.hoverColor}',
        },
        overlay: {
          select: {
            background: '{surface.900}', borderColor: '{surface.700}', color: '{text.color}',
          },
          popover: {
            background: '{surface.900}', borderColor: '{surface.700}', color: '{text.color}',
          },
          modal: {
            background: '{surface.900}', borderColor: '{surface.700}', color: '{text.color}',
          },
        },
        list: {
          option: {
            focusBackground: '{surface.800}',
            selectedBackground: '{highlight.background}',
            selectedFocusBackground: '{highlight.focusBackground}',
            color: '{text.color}',
            focusColor: '{text.hoverColor}',
            selectedColor: '{highlight.color}',
            selectedFocusColor: '{highlight.focusColor}',
            icon: { color: '{surface.500}', focusColor: '{surface.400}' },
          },
          optionGroup: { background: 'transparent', color: '{text.mutedColor}' },
        },
        navigation: {
          item: {
            focusBackground: '{surface.800}',
            activeBackground: '{surface.800}',
            color: '{text.color}',
            focusColor: '{text.hoverColor}',
            activeColor: '{text.hoverColor}',
            icon: { color: '{surface.400}', focusColor: '{surface.100}', activeColor: '{surface.0}' },
          },
          submenuLabel: { background: 'transparent', color: '{text.mutedColor}' },
          submenuIcon: { color: '{surface.500}', focusColor: '{surface.400}', activeColor: '{surface.400}' },
        },
      },
    },
  },
  components: {
    tag: {
      colorScheme: {
        light: {
          success: { background: 'rgba(63,143,108,.12)', color: '#2f6f53' },
          warn: { background: 'rgba(217,125,61,.14)', color: '#9a5320' },
          danger: { background: 'rgba(193,80,59,.12)', color: '#a33f2c' },
          secondary: { background: '#efebe1', color: '#4a4e56' },
          info: { background: '#efebe1', color: '#4a4e56' },
        },
        dark: {
          success: { background: 'rgba(63,143,108,.18)', color: '#6cc39b' },
          warn: { background: 'rgba(217,125,61,.18)', color: '#e8a06b' },
          danger: { background: 'rgba(193,80,59,.2)', color: '#e58a76' },
          secondary: { background: '#22262e', color: '#c7c3ba' },
          info: { background: '#22262e', color: '#c7c3ba' },
        },
      },
    },
    button: {
      colorScheme: {
        light: {
          root: {
            danger: {
              background: '#c1503b',
              hoverBackground: '#a33f2c',
              activeBackground: '#8f3626',
              borderColor: '#c1503b',
              hoverBorderColor: '#a33f2c',
              activeBorderColor: '#8f3626',
              color: '#ffffff',
              hoverColor: '#ffffff',
              activeColor: '#ffffff',
            },
          },
        },
      },
    },
  },
});

export default Listnun;
