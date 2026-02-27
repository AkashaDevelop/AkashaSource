import { useThemeStore } from '../store/theme';

export default function ThemeToggle() {
  const { theme, toggleTheme } = useThemeStore();
  const isDark = theme === 'dark';

  return (
    <button
      onClick={toggleTheme}
      aria-label="切换主题"
      style={{
        position: 'relative',
        width: '52px',
        height: '28px',
        borderRadius: '14px',
        border: '1px solid var(--border-strong)',
        background: isDark
          ? 'linear-gradient(135deg, #1e1040, #0d1a2e)'
          : 'linear-gradient(135deg, #fef3c7, #dbeafe)',
        cursor: 'pointer',
        transition: 'background 0.4s ease, border-color 0.3s ease',
        outline: 'none',
        flexShrink: 0,
      }}
    >
      {/* Stars (dark) / Clouds (light) background dots */}
      <span style={{
        position: 'absolute',
        inset: 0,
        borderRadius: '14px',
        overflow: 'hidden',
        opacity: isDark ? 1 : 0,
        transition: 'opacity 0.3s ease',
        pointerEvents: 'none',
      }}>
        {[
          { top: '4px', left: '8px', size: '2px', color: 'rgba(167,139,250,0.9)' },
          { top: '8px', left: '16px', size: '1.5px', color: 'rgba(34,211,238,0.8)' },
          { top: '5px', left: '24px', size: '1.5px', color: 'rgba(251,191,36,0.7)' },
        ].map((s, i) => (
          <span key={i} style={{
            position: 'absolute',
            top: s.top,
            left: s.left,
            width: s.size,
            height: s.size,
            borderRadius: '50%',
            background: s.color,
          }} />
        ))}
      </span>

      {/* Sliding thumb */}
      <span style={{
        position: 'absolute',
        top: '3px',
        left: isDark ? '27px' : '3px',
        width: '20px',
        height: '20px',
        borderRadius: '50%',
        background: isDark
          ? 'linear-gradient(135deg, #c4b5fd, #a78bfa)'
          : 'linear-gradient(135deg, #fbbf24, #f59e0b)',
        boxShadow: isDark
          ? '0 0 8px rgba(167,139,250,0.6)'
          : '0 0 8px rgba(251,191,36,0.5)',
        transition: 'left 0.3s cubic-bezier(0.34,1.56,0.64,1), background 0.3s ease, box-shadow 0.3s ease',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: '11px',
      }}>
        {isDark ? '🌙' : '☀️'}
      </span>
    </button>
  );
}
