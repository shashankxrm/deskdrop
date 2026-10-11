import type { Device } from "@/lib/api";

type Props = {
  devices: Device[];
  onRevoke: (id: string) => void;
  busyId?: string;
};

export function DeviceList({ devices, onRevoke, busyId }: Props) {
  if (devices.length === 0) {
    return <p>No devices.</p>;
  }
  return (
    <ul>
      {devices.map((d) => (
        <li key={d.id}>
          <strong>{d.name || "(unnamed)"}</strong>{" "}
          <span>{d.type}</span>{" "}
          <code>{d.id}</code>
          <button
            type="button"
            disabled={busyId === d.id}
            onClick={() => onRevoke(d.id)}
          >
            Unpair
          </button>
        </li>
      ))}
    </ul>
  );
}
