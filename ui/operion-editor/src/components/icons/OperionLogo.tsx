import type { SVGProps } from "react";

export function OperionLogo(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 100 24"
      fill="currentColor"
      aria-label="Operion Flow Logo"
      {...props}
    >
      <path
        d="M0 12C0 5.373 5.373 0 12 0H88C94.627 0 100 5.373 100 12C100 18.627 94.627 24 88 24H12C5.373 24 0 18.627 0 12Z"
        fill="hsl(var(--primary))"
      />
      <text
        x="50%"
        y="50%"
        dy=".3em"
        textAnchor="middle"
        fill="hsl(var(--primary-foreground))"
        fontSize="10"
        fontWeight="bold"
        className="font-headline"
      >
        Operion
      </text>
    </svg>
  );
}
