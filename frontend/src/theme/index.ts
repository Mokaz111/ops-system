import { createTheme } from '@mui/material/styles';
import { colors } from './colors';

const theme = createTheme({
  palette: {
    primary: colors.primary,
    secondary: colors.secondary,
    error: colors.error,
    warning: colors.warning,
    success: colors.success,
    info: colors.info,
    background: {
      default: colors.background.default,
      paper: colors.background.paper,
    },
    text: {
      primary: colors.text.primary,
      secondary: colors.text.secondary,
      disabled: colors.text.disabled,
    },
    divider: colors.grey[300],
  },
  typography: {
    fontFamily: '"Google Sans", "Roboto", "Helvetica Neue", Arial, sans-serif',
    h4: { fontWeight: 500, fontSize: '1.5rem', letterSpacing: 0 },
    h5: { fontWeight: 500, fontSize: '1.25rem', letterSpacing: 0 },
    h6: { fontWeight: 500, fontSize: '1rem', letterSpacing: 0.15 },
    subtitle1: { fontWeight: 500, fontSize: '0.875rem', letterSpacing: 0.1 },
    subtitle2: { fontWeight: 500, fontSize: '0.8125rem', letterSpacing: 0.1 },
    body1: { fontSize: '0.875rem', letterSpacing: 0.2 },
    body2: { fontSize: '0.8125rem', letterSpacing: 0.25 },
    button: { fontWeight: 500, fontSize: '0.875rem', textTransform: 'none' as const, letterSpacing: 0.25 },
    caption: { fontSize: '0.75rem', letterSpacing: 0.4, color: colors.text.secondary },
  },
  shape: { borderRadius: 8 },
  shadows: [
    'none',
    '0 1px 2px 0 rgba(60,64,67,0.3), 0 1px 3px 1px rgba(60,64,67,0.15)',
    '0 1px 2px 0 rgba(60,64,67,0.3), 0 2px 6px 2px rgba(60,64,67,0.15)',
    '0 1px 3px 0 rgba(60,64,67,0.3), 0 4px 8px 3px rgba(60,64,67,0.15)',
    '0 2px 3px 0 rgba(60,64,67,0.3), 0 6px 10px 4px rgba(60,64,67,0.15)',
    '0 4px 4px 0 rgba(60,64,67,0.3), 0 8px 12px 6px rgba(60,64,67,0.15)',
    ...Array(19).fill('0 4px 4px 0 rgba(60,64,67,0.3), 0 8px 12px 6px rgba(60,64,67,0.15)'),
  ] as any,
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          backgroundColor: colors.background.surface,
        },
      },
    },
    MuiButton: {
      defaultProps: { disableElevation: true },
      styleOverrides: {
        root: {
          borderRadius: 20,
          padding: '6px 24px',
          fontWeight: 500,
        },
        contained: {
          '&:hover': { boxShadow: '0 1px 2px 0 rgba(60,64,67,0.3), 0 1px 3px 1px rgba(60,64,67,0.15)' },
        },
        outlined: {
          borderColor: colors.grey[300],
          color: colors.primary.main,
          '&:hover': { backgroundColor: `${colors.primary.main}08`, borderColor: colors.primary.main },
        },
      },
    },
    MuiCard: {
      defaultProps: { elevation: 0 },
      styleOverrides: {
        root: {
          border: `1px solid ${colors.grey[200]}`,
          borderRadius: 8,
          '&:hover': { boxShadow: '0 1px 2px 0 rgba(60,64,67,0.3), 0 2px 6px 2px rgba(60,64,67,0.15)' },
        },
      },
    },
    MuiPaper: {
      defaultProps: { elevation: 0 },
      styleOverrides: {
        root: { backgroundImage: 'none' },
        outlined: { borderColor: colors.grey[200] },
      },
    },
    MuiTableHead: {
      styleOverrides: {
        root: {
          '& .MuiTableCell-head': {
            fontWeight: 500,
            color: colors.text.secondary,
            backgroundColor: colors.grey[50],
            borderBottom: `1px solid ${colors.grey[200]}`,
            fontSize: '0.75rem',
            textTransform: 'uppercase',
            letterSpacing: 0.5,
          },
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: {
          borderBottom: `1px solid ${colors.grey[100]}`,
          padding: '12px 16px',
          fontSize: '0.875rem',
        },
      },
    },
    MuiTableRow: {
      styleOverrides: {
        root: {
          '&:hover': { backgroundColor: `${colors.primary.main}04` },
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: { fontWeight: 500, fontSize: '0.75rem' },
        filledSuccess: { backgroundColor: '#e6f4ea', color: '#137333' },
        filledError: { backgroundColor: '#fce8e6', color: '#b3261e' },
        filledWarning: { backgroundColor: '#fef7e0', color: '#e37400' },
        filledInfo: { backgroundColor: '#e8f0fe', color: '#1a73e8' },
      },
    },
    MuiTextField: {
      defaultProps: { size: 'small', variant: 'outlined' },
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': {
            borderRadius: 8,
            '& fieldset': { borderColor: colors.grey[300] },
            '&:hover fieldset': { borderColor: colors.grey[400] },
          },
        },
      },
    },
    MuiDialog: {
      styleOverrides: {
        paper: { borderRadius: 12, boxShadow: '0 4px 4px 0 rgba(60,64,67,0.3), 0 8px 12px 6px rgba(60,64,67,0.15)' },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: { border: 'none' },
      },
    },
    MuiListItemButton: {
      styleOverrides: {
        root: {
          borderRadius: '0 20px 20px 0',
          marginRight: 12,
          '&.Mui-selected': {
            backgroundColor: '#e8f0fe',
            color: colors.primary.main,
            '& .MuiListItemIcon-root': { color: colors.primary.main },
            '&:hover': { backgroundColor: '#d2e3fc' },
          },
          '&:hover': { backgroundColor: colors.grey[100] },
        },
      },
    },
    MuiListItemIcon: {
      styleOverrides: {
        root: { minWidth: 40, color: colors.text.secondary },
      },
    },
    MuiAppBar: {
      defaultProps: { elevation: 0 },
      styleOverrides: {
        root: {
          backgroundColor: colors.background.default,
          borderBottom: `1px solid ${colors.grey[200]}`,
          color: colors.text.primary,
        },
      },
    },
    MuiTab: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 500,
          minWidth: 90,
        },
      },
    },
    MuiIconButton: {
      styleOverrides: {
        root: { borderRadius: '50%' },
      },
    },
  },
});

export default theme;
