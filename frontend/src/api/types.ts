export type Role = "user" | "admin";

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

export interface RegisterResponse {
  id: string;
  email: string;
  role: Role;
}

export interface Poll {
  id: string;
  title: string;
  description: string;
  status: "draft" | "active" | "closed";
  created_at: string;
}

export interface Candidate {
  id: string;
  poll_id: string;
  name: string;
}

export interface PollDetails {
  poll: Poll;
  candidates: Candidate[];
}

export interface MyVote {
  poll_id: string;
  candidate_id?: string;
  type?: "VOTE_CAST" | "VOTE_CANCELLED" | "VOTE_CHANGED";
  has_vote: boolean;
}

export interface ResultsResponse {
  poll_id: string;
  results: Record<string, number>;
}

export interface ChainVerifyResponse {
  valid: boolean;
  block_count: number;
  last_block_id: number;
  last_hash: string;
  error?: string;
}
