import { FormEvent, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { admin, audit, polls } from "../api/endpoints";
import { ApiError } from "../api/client";
import type { Poll } from "../api/types";

export function AdminPage() {
  const qc = useQueryClient();
  const list = useQuery({ queryKey: ["polls"], queryFn: polls.list });
  const chain = useQuery({
    queryKey: ["chain", "verify"],
    queryFn: audit.chainVerify,
    refetchInterval: 10000,
  });

  const createPoll = useMutation({
    mutationFn: ({ title, desc }: { title: string; desc: string }) => admin.createPoll(title, desc),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["polls"] }),
  });

  const [title, setTitle] = useState("");
  const [desc, setDesc] = useState("");
  const [error, setError] = useState<string | null>(null);

  function submitPoll(e: FormEvent) {
    e.preventDefault();
    setError(null);
    createPoll.mutate(
      { title, desc },
      {
        onSuccess: () => {
          setTitle("");
          setDesc("");
        },
        onError: (e) => setError(e instanceof ApiError ? e.message : String(e)),
      }
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Admin</h1>

      <section className="bg-white rounded shadow p-4">
        <h2 className="font-semibold mb-2">Audit chain status</h2>
        {chain.isLoading ? (
          <p>...</p>
        ) : chain.error ? (
          <p className="text-red-600 text-sm">{(chain.error as Error).message}</p>
        ) : chain.data ? (
          <div className="text-sm space-y-1">
            <div>
              Valid: {chain.data.valid ? <span className="text-green-700">yes</span> : <span className="text-red-700">no</span>}
            </div>
            <div>Blocks: {chain.data.block_count}</div>
            <div>Last block id: {chain.data.last_block_id}</div>
            {chain.data.last_hash && (
              <div className="font-mono text-xs break-all">last_hash: {chain.data.last_hash}</div>
            )}
            {chain.data.error && <div className="text-red-700">error: {chain.data.error}</div>}
          </div>
        ) : null}
      </section>

      <section className="bg-white rounded shadow p-4">
        <h2 className="font-semibold mb-3">Create poll</h2>
        <form onSubmit={submitPoll} className="space-y-2">
          <input
            className="w-full border rounded px-3 py-2"
            placeholder="title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
          />
          <textarea
            className="w-full border rounded px-3 py-2"
            placeholder="description"
            value={desc}
            onChange={(e) => setDesc(e.target.value)}
            rows={2}
          />
          {error && <div className="text-sm text-red-600">{error}</div>}
          <button
            disabled={createPoll.isPending}
            className="bg-slate-900 text-white px-4 py-2 rounded hover:bg-slate-700 disabled:opacity-50"
          >
            Create
          </button>
        </form>
      </section>

      <section className="bg-white rounded shadow p-4">
        <h2 className="font-semibold mb-3">Polls</h2>
        {list.isLoading ? (
          <p>...</p>
        ) : list.data?.polls?.length ? (
          <div className="space-y-4">
            {list.data.polls.map((p) => (
              <PollAdminCard key={p.id} poll={p} />
            ))}
          </div>
        ) : (
          <p className="text-sm text-slate-600">No polls yet.</p>
        )}
      </section>
    </div>
  );
}

function PollAdminCard({ poll }: { poll: Poll }) {
  const qc = useQueryClient();
  const details = useQuery({ queryKey: ["poll", poll.id], queryFn: () => polls.get(poll.id) });
  const results = useQuery({
    queryKey: ["poll", poll.id, "results"],
    queryFn: () => polls.results(poll.id),
    refetchInterval: 5000,
  });

  const addCand = useMutation({
    mutationFn: (name: string) => admin.addCandidate(poll.id, name),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["poll", poll.id] }),
  });
  const setStatus = useMutation({
    mutationFn: (status: Poll["status"]) => admin.setStatus(poll.id, status),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["polls"] });
      qc.invalidateQueries({ queryKey: ["poll", poll.id] });
    },
  });

  const [candName, setCandName] = useState("");

  return (
    <div className="border rounded p-3">
      <div className="flex items-baseline justify-between">
        <div>
          <div className="font-medium">{poll.title}</div>
          <div className="text-xs text-slate-500">{poll.id}</div>
        </div>
        <div className="text-sm">
          <select
            value={poll.status}
            onChange={(e) => setStatus.mutate(e.target.value as Poll["status"])}
            className="border rounded px-2 py-1"
          >
            <option value="draft">draft</option>
            <option value="active">active</option>
            <option value="closed">closed</option>
          </select>
        </div>
      </div>

      <div className="mt-3">
        <div className="text-sm font-semibold mb-1">Candidates</div>
        {details.data?.candidates.length ? (
          <ul className="text-sm space-y-1">
            {details.data.candidates.map((c) => (
              <li key={c.id} className="flex justify-between">
                <span>{c.name}</span>
                <span className="text-slate-500">{results.data?.results?.[c.id] ?? 0} votes</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-slate-500">No candidates yet.</p>
        )}
        <form
          className="mt-2 flex gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (!candName.trim()) return;
            addCand.mutate(candName, { onSuccess: () => setCandName("") });
          }}
        >
          <input
            className="flex-1 border rounded px-2 py-1 text-sm"
            placeholder="candidate name"
            value={candName}
            onChange={(e) => setCandName(e.target.value)}
          />
          <button className="bg-slate-700 text-white text-sm px-3 py-1 rounded hover:bg-slate-600">
            Add
          </button>
        </form>
      </div>
    </div>
  );
}
