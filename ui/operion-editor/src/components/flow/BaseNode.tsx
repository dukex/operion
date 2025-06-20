export default function Node({
  icon,
  color,
  background,
  children,
  title,
}: {
  icon: React.ReactNode;
  color: string;
  background: string;
  children: React.ReactNode;
  title: string;
}) {
  return (
    <div
      className={`border rounded-3xl p-1 flex flex-row items-center`}
      style={{ borderColor: color, backgroundColor: background, color: color }}
    >
      <div
        className={`rounded-3xl p-4 mr-2 text-white`}
        style={{ backgroundColor: color }}
      >
        {icon}
      </div>
      <div className="flex flex-col pr-2 text-base">
        <span className="text-sm" style={{ textTransform: "capitalize" }}>
          {title}
        </span>

        <div className="font-bold">{children}</div>
      </div>
    </div>
  );
}
