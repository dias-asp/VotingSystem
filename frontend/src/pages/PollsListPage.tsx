import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { polls } from "../api/endpoints";

export function PollsListPage() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["polls"],
    queryFn: polls.list,
  });

  if (isLoading) return <p>Loading...</p>;
  if (error) return <p className="text-red-600">{(error as Error).message}</p>;

  const list = data?.polls ?? [];

  return (
    <div>
      <h1 className="text-2xl font-semibold mb-4">Polls</h1>
      {list.length === 0 ? (
        <p className="text-slate-600">No polls yet.</p>
      ) : (
        <ul className="space-y-3">
          {list.map((p) => (
            <li key={p.id} className="bg-white rounded shadow p-4">
              <Link to={`/polls/${p.id}`} className="text-lg font-medium hover:underline">
                {p.title}
              </Link>
              <div className="text-sm text-slate-600 mt-1">
                Status: <StatusBadge status={p.status} />
              </div>
              {p.description && <p className="text-sm text-slate-700 mt-2">{p.description}</p>}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const cls =
    status === "active"
      ? "bg-green-100 text-green-800"
      : status === "closed"
      ? "bg-slate-200 text-slate-700"
      : "bg-amber-100 text-amber-800";
  return <span className={`px-2 py-0.5 rounded text-xs ${cls}`}>{status}</span>;
}
