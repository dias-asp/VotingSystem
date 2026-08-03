import { api } from "./client";
import type {
  Candidate,
  ChainVerifyResponse,
  MyVote,
  Poll,
  PollDetails,
  RegisterResponse,
  ResultsResponse,
  TokenPair,
} from "./types";

export const auth = {
  register: (email: string, password: string) =>
    api<RegisterResponse>("/api/auth/auth/register", {
      method: "POST",
      auth: false,
      body: JSON.stringify({ email, password }),
    }),
  login: (email: string, password: string) =>
    api<TokenPair>("/api/auth/auth/login", {
      method: "POST",
      auth: false,
      body: JSON.stringify({ email, password }),
    }),
  logout: (refresh_token: string) =>
    api<void>("/api/auth/auth/logout", {
      method: "POST",
      auth: false,
      body: JSON.stringify({ refresh_token }),
    }),
};

export const polls = {
  list: () => api<{ polls: Poll[] }>("/api/polls"),
  get: (id: string) => api<PollDetails>(`/api/polls/${id}`),
  results: (id: string) => api<ResultsResponse>(`/api/polls/${id}/results`),
  myVote: (id: string) => api<MyVote>(`/api/polls/${id}/me`),
  cast: (poll_id: string, candidate_id: string) =>
    api<void>("/api/vote", { method: "POST", body: JSON.stringify({ poll_id, candidate_id }) }),
  revote: (poll_id: string, candidate_id: string) =>
    api<void>("/api/revote", { method: "POST", body: JSON.stringify({ poll_id, candidate_id }) }),
  cancel: (poll_id: string) =>
    api<void>("/api/cancel-vote", { method: "POST", body: JSON.stringify({ poll_id }) }),
};

export const admin = {
  createPoll: (title: string, description: string) =>
    api<Poll>("/api/admin/polls", { method: "POST", body: JSON.stringify({ title, description }) }),
  addCandidate: (poll_id: string, name: string, id?: string) =>
    api<Candidate>(`/api/admin/polls/${poll_id}/candidates`, {
      method: "POST",
      body: JSON.stringify({ name, id: id ?? "" }),
    }),
  setStatus: (poll_id: string, status: Poll["status"]) =>
    api<void>(`/api/admin/polls/${poll_id}/status`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
    }),
};

export const audit = {
  chainVerify: () => api<ChainVerifyResponse>("/api/audit/chain/verify", { auth: false }),
};
