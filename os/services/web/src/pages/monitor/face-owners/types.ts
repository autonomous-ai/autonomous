// Local data types + constants for the Face Owners (Users) page. The shared
// FaceOwnerDetail/FaceOwnersDetail types stay in ../types — only the types
// specific to this page live here.

export interface CooldownEntry {
  person_id: string;
  kind: string;
  last_seen_ago: number;
  cooldown_remaining: number;
  cooldown_total: number;
}
export interface CooldownState {
  owners: CooldownEntry[];
  strangers: CooldownEntry[];
  owners_forget_s: number;
  strangers_forget_s: number;
}

export interface StrangerSample {
  filename: string;
  size_bytes: number;
  mtime: number;
}
export interface StrangerCluster {
  hash: string;
  sample_count: number;
  latest_mtime: number;
  samples: StrangerSample[];
}
export interface StrangersData {
  total: number;
  clusters: StrangerCluster[];
}

export interface FaceStrangerStat {
  stranger_id: string;
  count: number;
  first_seen: string;
  last_seen: string;
}

// Familiar-stranger threshold mirrors the device's _FAMILIAR_VISIT_THRESHOLD.
// At this count the device pushes an enroll prompt to the agent (one-shot).
export const FAMILIAR_VISIT_THRESHOLD = 2;
