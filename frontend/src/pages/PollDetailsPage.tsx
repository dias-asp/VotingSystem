import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { polls } from "../api/endpoints";
import { ApiError } from "../api/client";
import { useState } from "react";

export function PollDetailsPage() {
  const { id = "" } = useParams();
  const qc = useQueryClient();
  const [error, setError] = useState<string | null>(null);

  const details = useQuery({ queryKey: ["poll", id], queryFn: () => polls.get(id), enabled: !!id });
  const results = useQuery({
    queryKey: ["poll", id, "results"],
    queryFn: () => polls.results(id),
    enabled: !!id,
    refetchInterval: 5000,
  });
  const myVote = useQuery({ queryKey: ["poll", id, "me"], queryFn: () => polls.myVote(id), enabled: !!id });

  function refresh() {
    qc.invalidateQueries({ queryKey: ["poll", id] });
  }

  const cast = useMutation({
    mutationFn: (cand: string) => polls.cast(id, cand),
    onSuccess: refresh,
    onError: (e) => setError(e instanceof ApiError ? e.message : String(e)),
  });
  const revote = useMutation({
    mutationFn: (cand: string) => polls.revote(id, cand),
    onSuccess: refresh,
    onError: (e) => setError(e instanceof ApiError ? e.message : String(e)),
  });
  const cancel = useMutation({
    mutationFn: () => polls.cancel(id),
    onSuccess: refresh,
    onError: (e) => setError(e instanceof ApiError ? e.message : String(e)),
  });

  if (details.isLoading) return <p>Loading...</p>;
  if (details.error) return <p className="text-red-600">{(details.error as Error).message}</p>;
  if (!details.data) return null;

  const { poll, candidates } = details.data;
  const my = myVote.data;
  const activeVote = my?.has_vote && my.type !== "VOTE_CANCELLED";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">{poll.title}</h1>
        <p className="text-sm text-slate-600">Status: {poll.status}</p>
        {poll.description && <p className="mt-2 text-slate-700">{poll.description}</p>}
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 rounded p-3 text-sm">
          {error}{" "}
          <button onClick={() => setError(null)} className="underline ml-2">
            dismiss
          </button>
        </div>
      )}

      <section className="bg-white rounded shadow p-4">
        <h2 className="font-semibold mb-3">Your vote</h2>
        {my === undefined ? (
          <p>...</p>
        ) : !activeVote ? (
          <p className="text-sm text-slate-600">
            {my?.type === "VOTE_CANCELLED" ? "Your vote was cancelled. Pick a new candidate to re-vote." : "You haven't voted yet."}
          </p>
        ) : (
          <p className="text-sm">
            Voted for <strong>{candidates.find((c) => c.id === my.candidate_id)?.name ?? my.candidate_id}</strong>
            <button
              onClick={() => cancel.mutate()}
              disabled={cancel.isPending}
              className="ml-3 text-sm underline text-red-700"
            >
              Cancel vote
            </button>
          </p>
        )}
      </section>

      <section className="bg-white rounded shadow p-4">
        <h2 className="font-semibold mb-3">Candidates</h2>
        {candidates.length === 0 ? (
          <p className="text-sm text-slate-600">No candidates yet.</p>
        ) : (
          <ul className="space-y-2">
            {candidates.map((c) => {
              const count = results.data?.results?.[c.id] ?? 0;
              const isCurrent = activeVote && my?.candidate_id === c.id;
              return (
                <li key={c.id} className="flex items-center justify-between border rounded px-3 py-2">
                  <div>
                    <span className="font-medium">{c.name}</span>
                    {isCurrent && <span className="ml-2 text-xs text-green-700">(your choice)</span>}
                    <div className="text-xs text-slate-500">{count} votes</div>
                  </div>
                  {poll.status === "active" && (
                    <button
                      disabled={isCurrent || cast.isPending || revote.isPending}
                      onClick={() => {
                        setError(null);
                        if (my?.type === "VOTE_CANCELLED") revote.mutate(c.id);
                        else if (!my?.has_vote) cast.mutate(c.id);
                        else if (activeVote && my?.candidate_id !== c.id) {
                          // need to cancel first then change
                          cancel.mutate(undefined, {
                            onSuccess: () => revote.mutate(c.id),
                          });
                        }
                      }}
                      className="px-3 py-1 text-sm bg-slate-900 text-white rounded hover:bg-slate-700 disabled:opacity-50"
                    >
                      {isCurrent ? "Selected" : activeVote ? "Switch" : "Vote"}
                    </button>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </section>
    </div>
  );
}
