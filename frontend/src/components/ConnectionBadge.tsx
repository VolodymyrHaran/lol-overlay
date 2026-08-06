import type { ConnectionStatus } from "../hooks/useRoom";

type Props = {
  status: ConnectionStatus;
};

const statusConfig: Record<
  ConnectionStatus,
  {
    label: string;
    className: string;
  }
> = {
  connecting: {
    label: "Connecting",
    className:
      "border-yellow-500/40 bg-yellow-500/10 text-yellow-300",
  },
  connected: {
    label: "Connected",
    className:
      "border-emerald-500/40 bg-emerald-500/10 text-emerald-300",
  },
  reconnecting: {
    label: "Reconnecting",
    className:
      "border-orange-500/40 bg-orange-500/10 text-orange-300",
  },
  disconnected: {
    label: "Disconnected",
    className:
      "border-red-500/40 bg-red-500/10 text-red-300",
  },
};

function ConnectionBadge({ status }: Props) {
  const config = statusConfig[status];

  return (
    <div
      className={`flex items-center gap-2 rounded-full border px-3 py-1 text-sm font-medium ${config.className}`}
    >
      <span className="relative flex h-2 w-2">
        {status === "connected" && (
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-60" />
        )}

        <span className="relative inline-flex h-2 w-2 rounded-full bg-current" />
      </span>

      {config.label}
    </div>
  );
}

export default ConnectionBadge;