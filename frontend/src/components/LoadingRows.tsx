interface LoadingRowsProps {
  cols: number;
  rows?: number;
}

export default function LoadingRows({ cols, rows = 5 }: LoadingRowsProps) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <tr key={i}>
          {Array.from({ length: cols }).map((_, j) => (
            <td key={j} style={{ padding: '12px 16px' }}>
              <div
                className="skeleton-shimmer"
                style={{
                  height: '14px',
                  borderRadius: '7px',
                  width: j === 0 ? '60%' : j === cols - 1 ? '40%' : '80%',
                  animationDelay: `${i * 0.07}s`,
                }}
              />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}
