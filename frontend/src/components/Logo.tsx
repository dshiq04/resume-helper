export function Logo({ size = 34 }: { size?: number }) {
  return (
    <span
      className="brand-logo"
      style={{ width: size, height: size, borderRadius: Math.max(9, size * 0.28) }}
      aria-hidden="true"
    >
      <svg viewBox="0 0 32 32" width={size * 0.62} height={size * 0.62} fill="none">
        <path
          d="M10 6.5h8.8L24 11.7v13.8H10z"
          stroke="currentColor"
          strokeWidth="1.7"
          strokeLinejoin="round"
        />
        <path
          d="M18.5 6.5v5.4H24"
          stroke="currentColor"
          strokeWidth="1.7"
          strokeLinejoin="round"
        />
        <path
          d="m13.4 18.1 2.6 2.6 5.3-5.4"
          stroke="currentColor"
          strokeWidth="1.9"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </span>
  );
}
