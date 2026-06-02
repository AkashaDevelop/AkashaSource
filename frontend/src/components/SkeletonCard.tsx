interface SkeletonCardProps {
  lines?: number;
  showIcon?: boolean;
}

export function SkeletonCard({ lines = 2, showIcon = true }: SkeletonCardProps) {
  return (
    <div className="akasha-card p-5">
      <div className="flex justify-between items-start">
        <div className="flex-1 min-w-0">
          <div className="skeleton-shimmer h-3 w-1/3 rounded mb-3" />
          <div className="skeleton-shimmer h-7 w-1/2 rounded" />
        </div>
        {showIcon && (
          <div className="skeleton-shimmer w-10 h-10 rounded-xl flex-shrink-0 ml-3" />
        )}
      </div>
      {lines > 0 && (
        <div className="mt-4 space-y-2">
          {Array.from({ length: lines }).map((_, i) => (
            <div
              key={i}
              className="skeleton-shimmer h-3 rounded"
              style={{
                width: `${80 - i * 15}%`,
                animationDelay: `${i * 0.1}s`,
              }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export function SkeletonText({ width = '60%', height = 14 }: { width?: string; height?: number }) {
  return (
    <div
      className="skeleton-shimmer rounded"
      style={{ width, height: `${height}px` }}
    />
  );
}
