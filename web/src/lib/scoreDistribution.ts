import api from "@/lib/api";

export interface ScoreDistributionBucket {
  range: string;
  count: number;
  min_score: number;
  max_score: number;
}

const CACHE_TTL_MS = 30_000;

let cachedBuckets: ScoreDistributionBucket[] | null = null;
let lastFetchedAtMs = 0;
let inFlightRequest: Promise<ScoreDistributionBucket[]> | null = null;

function toNumber(value: unknown, fallback = 0): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function extractRawBuckets(responseData: unknown): unknown {
  if (Array.isArray(responseData)) return responseData;
  if (responseData && typeof responseData === "object") {
    const record = responseData as Record<string, unknown>;
    return record.buckets ?? record.distribution ?? [];
  }
  return [];
}

function normalizeBuckets(rawBuckets: unknown): ScoreDistributionBucket[] {
  if (!Array.isArray(rawBuckets)) return [];

  return rawBuckets.map((bucket, index) => {
    const record =
      bucket && typeof bucket === "object"
        ? (bucket as Record<string, unknown>)
        : {};

    const minScore = toNumber(record.min_score, index * 10);
    const maxScore = toNumber(record.max_score, minScore + 9);

    return {
      range: String(record.range ?? `${minScore}-${maxScore}`),
      count: toNumber(record.count),
      min_score: minScore,
      max_score: maxScore,
    };
  });
}

export async function getScoreDistributionCached(options?: {
  force?: boolean;
}) {
  const force = options?.force === true;

  if (force) {
    cachedBuckets = null;
    lastFetchedAtMs = 0;
  }

  const now = Date.now();
  if (!force && cachedBuckets && now - lastFetchedAtMs < CACHE_TTL_MS) {
    return cachedBuckets;
  }

  if (inFlightRequest) {
    return inFlightRequest;
  }

  inFlightRequest = (async () => {
    const { data } = await api.get("/dashboard/score-distribution");
    const buckets = normalizeBuckets(extractRawBuckets(data));
    cachedBuckets = buckets;
    lastFetchedAtMs = Date.now();
    return buckets;
  })().finally(() => {
    inFlightRequest = null;
  });

  return inFlightRequest;
}

export function invalidateScoreDistributionCache() {
  cachedBuckets = null;
  lastFetchedAtMs = 0;
}
